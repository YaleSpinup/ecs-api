package orchestration

import (
	"context"

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
			if input.Service != nil && input.Service.Cluster != nil {
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

// processRepositoryCredentialsUpdate processes the passed in repository credentials and applies them appropriately.  If the
// container definition already has credentials set, assume we are updating the credentials in place. In that case, we shouldn't
// need to set the repository credentials on the input since they are already applied to the container def.  Otherwise, assume
// the credentials are new (maybe the container def is new too).  In this case, we create the secret and apply the result to the input.
func (o *Orchestrator) processRepositoryCredentialsUpdate(ctx context.Context, input *ServiceOrchestrationUpdateInput) (map[string]interface{}, error) {
	if len(input.Credentials) == 0 {
		log.Debugf("no private repository credentials passed")
		return nil, nil
	}

	client := o.SecretsManager
	creds := make(map[string]interface{}, len(input.Credentials))
	for _, cd := range input.TaskDefinition.ContainerDefinitions {
		containerName := aws.StringValue(cd.Name)
		log.Debugf("processing container definition %s", containerName)
		if secret, ok := input.Credentials[containerName]; ok {
			// if the credentials parameter is specified in the container definition, assume we are updating
			// the credentia in place.  otherwise, assume we are creating a *new* secret/credential
			if cd.RepositoryCredentials != nil && cd.RepositoryCredentials.CredentialsParameter != nil {
				secretArn := cd.RepositoryCredentials.CredentialsParameter
				log.Infof("updating repository credentials secret '%s' for container definition: %s", containerName, aws.StringValue(secretArn))

				secretUpdate := secretsmanager.PutSecretValueInput{
					ClientRequestToken: secret.ClientRequestToken,
					SecretId:           secretArn,
					SecretString:       secret.SecretString,
				}

				out, err := client.UpdateSecret(ctx, &secretUpdate)
				if err != nil {
					return nil, err
				}

				log.Debugf("output: %+v", out)
				creds[containerName] = out
			} else {
				log.Infof("creating new repository credentials secret for container definition: %s", containerName)

				smTags := make([]*secretsmanager.Tag, len(input.TaskDefinition.Tags))
				for i, t := range input.TaskDefinition.Tags {
					smTags[i] = &secretsmanager.Tag{Key: t.Key, Value: t.Value}
				}
				secret.Tags = smTags

				org := ""
				if o.Org != "" {
					org = o.Org + "/"
				}

				cluster := ""
				if input.Service != nil && input.Service.Cluster != nil {
					cluster = aws.StringValue(input.Service.Cluster) + "/"
				}

				secret.Name = aws.String("spinup/" + org + cluster + aws.StringValue(secret.Name))

				out, err := client.CreateSecret(ctx, secret)
				if err != nil {
					return nil, err
				}

				log.Debugf("output: %+v", out)

				log.Infof("setting repository credentials secret for container definition: %s to %s", containerName, aws.StringValue(out.ARN))

				cd.SetRepositoryCredentials(&ecs.RepositoryCredentials{CredentialsParameter: out.ARN})
				creds[containerName] = out
			}
		} else {
			log.Infof("assuming container definition %s references a public image, no credentials included", containerName)
		}
	}

	return creds, nil
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
