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
func (o *Orchestrator) processRepositoryCredentials(ctx context.Context, input *ServiceOrchestrationInput) (map[string]*secretsmanager.CreateSecretOutput, error) {
	if len(input.Credentials) == 0 {
		log.Debugf("no private repository credentials passed")
		return nil, nil
	}

	client := o.SecretsManager
	creds := make(map[string]*secretsmanager.CreateSecretOutput, len(input.Credentials))
	for _, cd := range input.TaskDefinition.ContainerDefinitions {
		name := aws.StringValue(cd.Name)
		log.Debugf("processing container definition %s", name)
		if cd.RepositoryCredentials != nil {
			log.Infof("using respository credentials referenced in container definition %s: %s", name, cd.RepositoryCredentials.String())
		} else if secret, ok := input.Credentials[name]; ok {
			log.Infof("creating repository credentials secret for container definition: %s", name)

			secret.Tags = o.processSecretsmanagerTags(secret.Tags)
			out, err := client.CreateSecret(ctx, secret)
			if err != nil {
				return nil, err
			}

			log.Debugf("output: %+v", out)

			log.Infof("setting repository credentials secret for container definition: %s to %s", name, aws.StringValue(out.ARN))

			cd.SetRepositoryCredentials(&ecs.RepositoryCredentials{CredentialsParameter: out.ARN})

			creds[name] = out
		} else {
			log.Infof("assuming container definition %s references a public image, no credentials included", name)
		}
	}

	return creds, nil
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
		name := aws.StringValue(cd.Name)
		log.Debugf("processing container definition %s", name)
		if secret, ok := input.Credentials[name]; ok {
			if cd.RepositoryCredentials != nil && cd.RepositoryCredentials.CredentialsParameter != nil {
				secretArn := cd.RepositoryCredentials.CredentialsParameter
				log.Infof("updating repository credentials secret '%s' for container definition: %s", name, aws.StringValue(secretArn))

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
				creds[name] = out
			} else {
				log.Infof("creating new repository credentials secret for container definition: %s", name)

				secret.Tags = o.processSecretsmanagerTags(secret.Tags)
				out, err := client.CreateSecret(ctx, secret)
				if err != nil {
					return nil, err
				}

				log.Debugf("output: %+v", out)

				log.Infof("setting repository credentials secret for container definition: %s to %s", name, aws.StringValue(out.ARN))

				cd.SetRepositoryCredentials(&ecs.RepositoryCredentials{CredentialsParameter: out.ARN})
				creds[name] = out
			}
		} else {
			log.Infof("assuming container definition %s references a public image, no credentials included", name)
		}
	}

	return creds, nil
}

func (o *Orchestrator) processSecretsmanagerTags(tags []*secretsmanager.Tag) []*secretsmanager.Tag {
	log.Debugf("processing secretsmanager tag list %+v", tags)
	newTags := []*secretsmanager.Tag{
		&secretsmanager.Tag{
			Key:   aws.String("spinup:org"),
			Value: aws.String(o.Org),
		},
	}

	for _, t := range tags {
		if aws.StringValue(t.Key) != "spinup:org" && aws.StringValue(t.Key) != "yale:org" {
			newTags = append(newTags, t)
		}
	}
	log.Debugf("returning processed secretsmanager tag list %+v", newTags)
	return newTags
}
