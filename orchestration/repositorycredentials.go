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

			newTags := []*secretsmanager.Tag{
				&secretsmanager.Tag{
					Key:   aws.String("spinup:org"),
					Value: aws.String(o.Org),
				},
			}

			for _, t := range secret.Tags {
				if aws.StringValue(t.Key) != "spinup:org" && aws.StringValue(t.Key) != "yale:org" {
					newTags = append(newTags, t)
				}
			}
			secret.Tags = newTags

			out, err := client.CreateSecret(ctx, secret)
			if err != nil {
				return nil, err
			}

			log.Debugf("output: %+v", out)

			log.Infof("setting repository credentials secret for container definition: %s to %s", name, aws.StringValue(out.ARN))
			cd.SetRepositoryCredentials(&ecs.RepositoryCredentials{
				CredentialsParameter: out.ARN,
			})
			creds[name] = out
		} else {
			log.Infof("assuming container definition %s references a public image, no credentials included", name)
		}
	}

	return creds, nil
}
