package orchestration

import (
	"context"
	"errors"
	"path"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/arn"
	"github.com/aws/aws-sdk-go/service/ecs"
	"github.com/aws/aws-sdk-go/service/secretsmanager"
	log "github.com/sirupsen/logrus"
)

// processRepositoryCredentialsCreate processes the Credentials portion of the input.  If the credentials are defined as input,
// they are created in the secretsmanager service and the ARN is applied to the task definition as repository credentials.
func (o *Orchestrator) processRepositoryCredentialsCreate(ctx context.Context, input *ServiceOrchestrationInput) (map[string]*secretsmanager.CreateSecretOutput, rollbackFunc, error) {
	rbfunc := defaultRbfunc("processRepositoryCredentialsCreate")

	if len(input.Credentials) == 0 {
		log.Debugf("no private repository credentials passed")
		return nil, rbfunc, nil
	}

	cluster := ""
	if input.Cluster != nil && input.Cluster.ClusterName != nil {
		cluster = aws.StringValue(input.Cluster.ClusterName) + "/"
	}

	// prefix for secret names is 'spinup/org/clustername'
	prefix := "spinup/" + o.Org + "/" + cluster

	creds, err := o.createRepostitoryCredentials(ctx, prefix, input.Credentials, input.Tags)
	if err != nil {
		return nil, rbfunc, err
	}

	for _, cd := range input.TaskDefinition.ContainerDefinitions {
		containerName := aws.StringValue(cd.Name)
		log.Debugf("processing container definition %s", containerName)

		if secret, ok := creds[containerName]; ok {
			log.Infof("setting repository credentials secret for container definition: %s to %s", containerName, aws.StringValue(secret.ARN))
			cd.SetRepositoryCredentials(&ecs.RepositoryCredentials{CredentialsParameter: secret.ARN})
		} else {
			log.Infof("assuming container definition %s references a public image, no credentials included", containerName)
		}
	}

	rbfunc = func(ctx context.Context) error {
		for _, secret := range creds {
			id := aws.StringValue(secret.ARN)

			log.Debugf("rolling back secret %s", id)

			out, err := o.SecretsManager.DeleteSecret(ctx, id, 0)
			if err != nil {
				log.Errorf("failed deleting secret %s: %s", id, err)
			}

			log.Infof("successfully rolled back secret: %s", aws.StringValue(out.ARN))
		}

		return nil
	}

	log.Debugf("returning creds: %+v", creds)

	return creds, rbfunc, nil
}

// processTaskRepositoryCredentialsCreate processes the Credentials portion of the input for a task.  If the credentials are defined
// as input, they are created as secrets in the secretsmanager service and the ARN is applied to the task definition as repository credentials.
func (o *Orchestrator) processTaskRepositoryCredentialsCreate(ctx context.Context, input *TaskDefCreateOrchestrationInput) (map[string]*secretsmanager.CreateSecretOutput, rollbackFunc, error) {
	rbfunc := defaultRbfunc("processTaskRepositoryCredentialsCreate")

	if len(input.Credentials) == 0 {
		log.Debugf("no private repository credentials passed")
		return nil, rbfunc, nil
	}

	cluster := ""
	if input.Cluster != nil && input.Cluster.ClusterName != nil {
		cluster = aws.StringValue(input.Cluster.ClusterName) + "/"
	}

	// prefix for secret names is 'spinup/org/clustername'
	prefix := "spinup/" + o.Org + "/" + cluster

	creds, err := o.createRepostitoryCredentials(ctx, prefix, input.Credentials, input.Tags)
	if err != nil {
		return nil, rbfunc, err
	}

	for _, cd := range input.TaskDefinition.ContainerDefinitions {
		containerName := aws.StringValue(cd.Name)
		log.Debugf("processing container definition %s", containerName)

		if secret, ok := creds[containerName]; ok {
			log.Infof("setting repository credentials secret for container definition: %s to %s", containerName, aws.StringValue(secret.ARN))
			cd.SetRepositoryCredentials(&ecs.RepositoryCredentials{CredentialsParameter: secret.ARN})
		} else {
			log.Infof("assuming container definition %s references a public image, no credentials included", containerName)
		}
	}

	rbfunc = func(ctx context.Context) error {
		for _, secret := range creds {
			id := aws.StringValue(secret.ARN)

			log.Debugf("rolling back secret %s", id)

			out, err := o.SecretsManager.DeleteSecret(ctx, id, 0)
			if err != nil {
				log.Errorf("failed deleting secret %s: %s", id, err)
			}

			log.Infof("successfully rolled back secret: %s", aws.StringValue(out.ARN))
		}

		return nil
	}

	log.Debugf("returning creds: %+v", creds)

	return creds, rbfunc, nil
}

// processRepositoryCredentialsUpdate processes the repository credentials for the container definitions inside the task definition.
//
// If the active container definition HAS repostory credentials set
// ...AND the input has Credentials defined for the container definition
// ...AND the input has repository credentials set for the container definition
// ...THEN update the secret at the (active) ARN in place with the given Credentials and apply to the container definition
//
// If the active container definition HAS repository credentials set
// ...AND the input has Credentials defined for the container definition
// ...AND the input doesn't have repository credentials set for the container definition
// ...THEN set the input repository credentials to the ARN for the active container definition repository credentials
// ...AND update the secret at the ARN in place with the given Credentials
//
// If the active container definition HAS repository credentials set
// ...AND the input doesn't have Credentials defined for the container definition
// ...AND the input has repository credentials set for the container definition
// ...THEN override the repository credentials with the ARN of the active repository credentials
//
// If the active container definition HAS repository credentials set
// ...AND the input doesn't have repository credentials set
// ...AND the input doesn't have Credentials defined for the container definition
// ...THEN delete the secret at the ARN for the active container definition
//
// If the active container definition doesn't exist or doesn't have repository credentials set
// ...AND the input has Credentials defined for the container definition
// ...AND the input has repository credentials defined for the container definition
// ...THEN update the secret in place or fail if it doesn't exist
// Note: (this case shouldn't happen)
//
// If the active container doesn't exist or doesn't have repository credentials set
// ...AND the input has Credentials defined for the container definition
// ...THEN create a new secret and apply the resulting ARN to the repository credentials for the input
//
// If the active container doesn't exist or doesn't have repository credentials set
// ...AND the input doesn't have Credentials defined for the container definition
// ...THEN assume public image, no secrets are created, no repository credentials are applied
//
func (o *Orchestrator) processRepositoryCredentialsUpdate(ctx context.Context, input *ServiceOrchestrationUpdateInput, active *ServiceOrchestrationUpdateOutput) error {
	if active == nil || active.TaskDefinition == nil || active.TaskDefinition.ContainerDefinitions == nil {
		return errors.New("cannot process nil active input")
	}

	activeRepositoryCredentials := containterDefinitionCredsMap(active.TaskDefinition.ContainerDefinitions)
	log.Debugf("active repository credentials: %+v", activeRepositoryCredentials)

	inputRepositoryCredentials := containterDefinitionCredsMap(input.TaskDefinition.ContainerDefinitions)
	log.Debugf("input repository credentials: %+v", inputRepositoryCredentials)

	inputCredentials := input.Credentials
	log.Debugf("input credentials %+v", inputCredentials)

	org := ""
	if o.Org != "" {
		org = o.Org + "/"
	}

	cluster := ""
	if input.ClusterName != "" {
		cluster = input.ClusterName + "/"
	} else if input.Service != nil && input.Service.Cluster != nil {
		cluster = aws.StringValue(input.Service.Cluster) + "/"
	}

	// prefix is spinup/ss/spinup-000001/
	prefix := "spinup/" + org + cluster

	client := o.SecretsManager
	creds := make(map[string]interface{}, len(input.Credentials))
	markedForDeletion := []string{}
	for _, cd := range input.TaskDefinition.ContainerDefinitions {
		containerName := aws.StringValue(cd.Name)
		log.Debugf("processing container definition %s repository credentials", containerName)

		activeRepositoryCredential, hasActiveRepositoryCredential := activeRepositoryCredentials[containerName]
		inputRepositoryCredentials, hasInputRepositoryCredential := inputRepositoryCredentials[containerName]
		inputCredential, hasInputCredential := inputCredentials[containerName]

		// if there are active repository credentials and no input repository credentials or input credentials,
		// delete the secret at the active repository credentials
		if hasActiveRepositoryCredential && !hasInputRepositoryCredential && !hasInputCredential {
			log.Warnf("active %s container has repository credentials (%s) but updated definition doesn't, marking credentials for deletion", containerName, activeRepositoryCredential)
			markedForDeletion = append(markedForDeletion, activeRepositoryCredential)

			// if there are active repository credentials, set the input repository credentials to the active repository credentials
		} else if hasActiveRepositoryCredential {
			log.Debugf("overriding input repository credentials with active repository credentials")
			cd.RepositoryCredentials = &ecs.RepositoryCredentials{
				CredentialsParameter: aws.String(activeRepositoryCredential),
			}
			hasInputRepositoryCredential = true
			inputRepositoryCredentials = activeRepositoryCredential
		}

		// if the input contains repository credentials ARN, parse the ARN to see if it contains our prefix
		// if it doesn't contain the prefix, get the value and set the credentials to be created new and
		// mark the original root level credentials for deletion
		if hasInputRepositoryCredential {
			parsedArn, err := arn.Parse(inputRepositoryCredentials)
			if err != nil {
				return err
			}

			if !strings.HasPrefix(parsedArn.Resource, "secret:"+prefix) {
				log.Warnf("secret %s lives at the root, migrating", inputRepositoryCredentials)

				// if we don't have any new credentials from the user, set them up from the existing secret
				if !hasInputCredential {
					secretValue, err := o.SecretsManager.GetValue(ctx, inputRepositoryCredentials)
					if err != nil {
						return err
					}

					// remove any other prefix from the secretValue.Name
					// ie. some secrets seem to be under spinup/org instead of spinup/org/spaceid and those
					//  will get created created as spinup/org/spaceid/spinup/org/secretname if not cleaned
					p, n := path.Split(aws.StringValue(secretValue.Name))
					if p != "" {
						log.Infof("removing existing path %s from migrated secret name", p)
					}

					inputCredential = &secretsmanager.CreateSecretInput{
						Name:         aws.String(n),
						SecretString: secretValue.SecretString,
					}
					hasInputCredential = true
				}

				// clear the input credentials arn from the task definition input to create a new secret
				hasInputRepositoryCredential = false
				cd.RepositoryCredentials = nil

				log.Warnf("marking root level secret for deletion %s", inputRepositoryCredentials)
				markedForDeletion = append(markedForDeletion, inputRepositoryCredentials)
			}
		}

		// if there is an input credential and an input repository credential, update the input repository
		// credential with the active credential.  else if there's an input credential and no input repository
		// credential, create a new secret
		if hasInputCredential && hasInputRepositoryCredential {
			secretArn := cd.RepositoryCredentials.CredentialsParameter

			log.Infof("updating repository credentials secret '%s' for container definition: %s", containerName, aws.StringValue(secretArn))

			secretUpdate := secretsmanager.PutSecretValueInput{
				ClientRequestToken: inputCredential.ClientRequestToken,
				SecretId:           secretArn,
				SecretString:       inputCredential.SecretString,
			}

			out, err := client.UpdateSecret(ctx, &secretUpdate)
			if err != nil {
				return err
			}

			log.Debugf("update secret output for %s: %+v", containerName, out)

			creds[containerName] = out
		} else if hasInputCredential {
			log.Infof("creating new repository credentials secret for container definition: %s", containerName)

			smTags := make([]*secretsmanager.Tag, len(input.TaskDefinition.Tags))
			for i, t := range input.TaskDefinition.Tags {
				smTags[i] = &secretsmanager.Tag{Key: t.Key, Value: t.Value}
			}
			inputCredential.Tags = smTags

			inputCredential.Name = aws.String(prefix + aws.StringValue(inputCredential.Name))

			out, err := client.CreateSecret(ctx, inputCredential)
			if err != nil {
				return err
			}

			log.Debugf("create secret output for %s: %+v", containerName, out)

			log.Infof("setting repository credentials secret for container definition: %s to %s", containerName, aws.StringValue(out.ARN))

			cd.RepositoryCredentials = &ecs.RepositoryCredentials{
				CredentialsParameter: out.ARN,
			}

			creds[containerName] = out
		} else {
			log.Infof("no changes to repository credentials for %s", containerName)
		}
	}

	log.Debugf("processed update of repository credentials: %+v", creds)

	for _, m := range markedForDeletion {
		log.Infof("deleting secrets mamanger secret %s (marked for deletion)", m)
		if _, err := client.DeleteSecret(ctx, m, 0); err != nil {
			return err
		}
	}

	active.Credentials = creds

	return nil
}

// createRepostitoryCredentials takes the map of container names to secret inputs and creates the given secrets in secretsmanager with the prefix
func (o *Orchestrator) createRepostitoryCredentials(ctx context.Context, prefix string, input map[string]*secretsmanager.CreateSecretInput, tags []*Tag) (map[string]*secretsmanager.CreateSecretOutput, error) {
	log.Debugf("creating repository credentials with prefix %s: %+v", prefix, input)

	creds := make(map[string]*secretsmanager.CreateSecretOutput, len(input))

	for containerName, secretInput := range input {
		log.Infof("creating repository credentials secret for %s", containerName)

		secretInput.Tags = secretsmanagerTags(tags)

		if !strings.HasSuffix(prefix, "/") {
			prefix = prefix + "/"
		}

		secretInput.Name = aws.String(prefix + aws.StringValue(secretInput.Name))

		out, err := o.SecretsManager.CreateSecret(ctx, secretInput)
		if err != nil {
			log.Errorf("boom! %s", err)
			return nil, err
		}

		creds[containerName] = out
	}

	return creds, nil
}

// containterDefinitionCredsMap maps the container definition names to the ARN
func containterDefinitionCredsMap(containerDefinitions []*ecs.ContainerDefinition) map[string]string {
	creds := map[string]string{}
	for _, cd := range containerDefinitions {
		if cd.RepositoryCredentials != nil && cd.RepositoryCredentials.CredentialsParameter != nil {
			name := aws.StringValue(cd.Name)
			credsArn := aws.StringValue(cd.RepositoryCredentials.CredentialsParameter)
			creds[name] = credsArn
		}
	}
	return creds
}
