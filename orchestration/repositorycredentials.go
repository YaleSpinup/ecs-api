package orchestration

import (
	"context"
	"errors"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ecs"
	"github.com/aws/aws-sdk-go/service/secretsmanager"
	log "github.com/sirupsen/logrus"
)

// processRepositoryCredentials processes the Credentials portion of the input.  If existing repository credentials are
// provided with the task definition, they are used.  Otherwise, if the credentials are defined as input, they are created
// in the secretsmanager service.  If neither is true, nil is returned.
func (o *Orchestrator) processRepositoryCredentials(ctx context.Context, input *ServiceOrchestrationInput) (map[string]*secretsmanager.CreateSecretOutput, rollbackFunc, error) {
	rbfunc := func(ctx context.Context) error {
		log.Infof("processRepositoryCredentials rollback, nothing to do")
		return nil
	}

	if len(input.Credentials) == 0 {
		log.Debugf("no private repository credentials passed")
		return nil, rbfunc, nil
	}

	client := o.SecretsManager
	creds := make(map[string]*secretsmanager.CreateSecretOutput, len(input.Credentials))
	for _, cd := range input.TaskDefinition.ContainerDefinitions {
		containerName := aws.StringValue(cd.Name)
		log.Debugf("processing container definition %s", containerName)

		if cd.RepositoryCredentials != nil {
			log.Infof("using respository credentials referenced in container definition %s: %s", containerName, cd.RepositoryCredentials.String())
		} else if secret, ok := input.Credentials[containerName]; ok {
			log.Infof("creating repository credentials secret for container definition: %s", containerName)

			secret.Tags = o.processSecretsmanagerTags(input.Tags)

			org := ""
			if o.Org != "" {
				org = o.Org + "/"
			}

			cluster := ""
			if input.Cluster != nil && input.Cluster.ClusterName != nil {
				cluster = aws.StringValue(input.Cluster.ClusterName) + "/"
			} else if input.Service != nil && input.Service.Cluster != nil {
				cluster = aws.StringValue(input.Service.Cluster) + "/"
			}

			secret.Name = aws.String("spinup/" + org + cluster + aws.StringValue(secret.Name))

			out, err := client.CreateSecret(ctx, secret)
			if err != nil {
				return nil, rbfunc, err
			}

			log.Infof("setting repository credentials secret for container definition: %s to %s", containerName, aws.StringValue(out.ARN))

			cd.SetRepositoryCredentials(&ecs.RepositoryCredentials{CredentialsParameter: out.ARN})

			creds[containerName] = out
		} else {
			log.Infof("assuming container definition %s references a public image, no credentials included", containerName)
		}
	}

	rbfunc = func(ctx context.Context) error {
		for _, secret := range creds {
			id := aws.StringValue(secret.ARN)

			log.Debugf("rolling back secret %s", id)

			out, err := client.DeleteSecret(ctx, id, 0)
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

	client := o.SecretsManager
	creds := make(map[string]interface{}, len(input.Credentials))
	for _, cd := range input.TaskDefinition.ContainerDefinitions {
		containerName := aws.StringValue(cd.Name)
		log.Debugf("processing container definition %s repository credentials", containerName)

		activeRepositoryCredential, hasActiveRepositoryCredential := activeRepositoryCredentials[containerName]
		_, hasInputRepositoryCredential := inputRepositoryCredentials[containerName]
		inputCredential, hasInputCredential := inputCredentials[containerName]

		// if there are active repository credentials and no input repository credentials or input credentials,
		// delete the secret at the active repository credentials
		if hasActiveRepositoryCredential && !hasInputRepositoryCredential && !hasInputCredential {
			log.Warnf("active %s container has repository credentials (%s) but updated definition doesn't, deleting credentials", containerName, activeRepositoryCredential)
			if _, err := client.DeleteSecret(ctx, activeRepositoryCredential, 0); err != nil {
				return err
			}
			// if there are active repository credentials, set the input repository credentials to the active repository credentials
		} else if hasActiveRepositoryCredential {
			log.Debugf("overriding input repository credentials with active repository credentials")
			cd.RepositoryCredentials = &ecs.RepositoryCredentials{
				CredentialsParameter: aws.String(activeRepositoryCredential),
			}
			hasInputRepositoryCredential = true
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

			inputCredential.Name = aws.String("spinup/" + org + cluster + aws.StringValue(inputCredential.Name))

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
			log.Infof("nothing to do for %s", containerName)
		}
	}

	log.Debugf("processed update of repository credentials: %+v", creds)

	active.Credentials = creds

	return nil
}

func (o *Orchestrator) processSecretsmanagerTags(tags []*Tag) []*secretsmanager.Tag {
	log.Debugf("processing secretsmanager tag list %+v", tags)

	smTags := make([]*secretsmanager.Tag, len(tags))
	for i, t := range tags {
		smTags[i] = &secretsmanager.Tag{Key: t.Key, Value: t.Value}
	}

	log.Debugf("returning processed secretsmanager tag list %+v", smTags)

	return smTags
}

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
