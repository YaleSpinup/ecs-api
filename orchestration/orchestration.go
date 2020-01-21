// Package orchestration brings together the other components of the API into a
// single orchestration interface for creating and deleting ecs services
package orchestration

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/secretsmanager"

	"github.com/aws/aws-sdk-go/service/ecs"
	"github.com/aws/aws-sdk-go/service/servicediscovery"
	log "github.com/sirupsen/logrus"
)

// ServiceOrchestrationInput encapsulates a single request for a service
type ServiceOrchestrationInput struct {
	// https://docs.aws.amazon.com/sdk-for-go/api/service/ecs/#CreateClusterInput
	Cluster *ecs.CreateClusterInput
	// https://docs.aws.amazon.com/sdk-for-go/api/service/ecs/#RegisterTaskDefinitionInput
	TaskDefinition *ecs.RegisterTaskDefinitionInput
	// map of container definition names to private repository credentials
	// https://docs.aws.amazon.com/sdk-for-go/api/service/secretsmanager/#CreateSecretInput
	Credentials map[string]*secretsmanager.CreateSecretInput
	// https://docs.aws.amazon.com/sdk-for-go/api/service/ecs/#CreateServiceInput
	Service *ecs.CreateServiceInput
	// https://docs.aws.amazon.com/sdk-for-go/api/service/servicediscovery/#CreateServiceInput
	ServiceRegistry *servicediscovery.CreateServiceInput
}

// ServiceOrchestrationOutput is the output structure for service orchestration
type ServiceOrchestrationOutput struct {
	// https://docs.aws.amazon.com/sdk-for-go/api/service/ecs/#Cluster
	Cluster *ecs.Cluster
	// https://docs.aws.amazon.com/sdk-for-go/api/service/ecs/#TaskDefinition
	TaskDefinition *ecs.TaskDefinition
	// https://docs.aws.amazon.com/sdk-for-go/api/service/ecs/#Service
	Service *ecs.Service
	// https://docs.aws.amazon.com/sdk-for-go/api/service/servicediscovery/#Service
	ServiceDiscoveryService *servicediscovery.Service
}

// ServiceOrchestrationUpdateInput is in the input for service orchestration updates.  The following are supported:
//   service: desired count, deployment configuration, network configuration and task definition can be updated
//	 tags: will be applied to all resources
type ServiceOrchestrationUpdateInput struct {
	// https://docs.aws.amazon.com/sdk-for-go/api/service/ecs/#UpdateServiceInput
	// Service *ecs.UpdateServiceInput
	Tags               []*ecs.Tag
	ForceNewDeployment bool
}

// ServiceOrchestrationUpdateOutput is the output for service orchestration updates
type ServiceOrchestrationUpdateOutput struct {
	Service *ecs.Service
}

// ServiceDeleteInput encapsulates a request to delete a service with optional recursion
type ServiceDeleteInput struct {
	Cluster   *string
	Service   *string
	Recursive bool
}

// CreateService takes service orchestration input, builds up a service and returns the service orchestration output
func (o *Orchestrator) CreateService(ctx context.Context, input *ServiceOrchestrationInput) (*ServiceOrchestrationOutput, error) {
	log.Debugf("got service orchestration input object:\n %+v", input.Service)
	if input.Service == nil {
		return nil, errors.New("service definition is required")
	}

	output := &ServiceOrchestrationOutput{}
	cluster, err := o.processCluster(ctx, input)
	if err != nil {
		return nil, err
	}
	output.Cluster = cluster

	creds, err := o.processRepositoryCredentials(ctx, input)
	if err != nil {
		return nil, err
	}
	log.Debugf("%+v", creds)

	td, err := o.processTaskDefinition(ctx, input)
	if err != nil {
		return nil, err
	}
	output.TaskDefinition = td

	sr, err := o.processServiceRegistry(ctx, input)
	if err != nil {
		return nil, err
	}
	output.ServiceDiscoveryService = sr

	service, err := o.processService(ctx, input)
	if err != nil {
		return nil, err
	}
	output.Service = service

	return output, nil
}

// DeleteService takes a service orchestrator, service name and a cluster to delete and removes
// the service and the service registry
func (o *Orchestrator) DeleteService(ctx context.Context, input *ServiceDeleteInput) (*ServiceOrchestrationOutput, error) {
	service, err := getService(ctx, o.ECS.Service, input.Cluster, input.Service)
	if err != nil {
		return nil, err
	}

	log.Debugf("processing delete of service %+v", service)

	taskDefinition, err := getTaskDefinition(ctx, o.ECS.Service, service.TaskDefinition)
	if err != nil {
		return nil, err
	}

	log.Infof("removing service '%s'", aws.StringValue(service.ServiceArn))

	err = deleteService(ctx, o.ECS.Service, input)
	if err != nil {
		log.Errorf("error deleting service %s", err)
		return nil, err
	}

	// recursively remove the service registry and the cluster if it's empty
	// TODO: this should return a 202, not a 200
	if input.Recursive {
		log.Infof("removing '%s' dependencies recursively, asynchronously", aws.StringValue(service.ServiceArn))
		go func() {
			cleanupCtx := context.Background()

			// TODO: if we want to share repository credentials, we need to look for multiple
			// container definitions using the same credentials.
			for _, cd := range taskDefinition.ContainerDefinitions {
				log.Debugf("cleaning '%s' container definition '%s' components", aws.StringValue(service.ServiceArn), aws.StringValue(cd.Name))
				if cd.RepositoryCredentials != nil && aws.StringValue(cd.RepositoryCredentials.CredentialsParameter) != "" {
					credsArn := aws.StringValue(cd.RepositoryCredentials.CredentialsParameter)
					_, err = o.SecretsManager.DeleteSecret(cleanupCtx, credsArn, 0)
					if err != nil {
						log.Errorf("failed to delete secretsmanager secret '%s' for %s", credsArn, aws.StringValue(service.ServiceArn))
					} else {
						log.Infof("successfully deleted secretsmanager secret '%s'", credsArn)
					}
				}
			}

			cluCtx, cluCancel := context.WithTimeout(cleanupCtx, 120*time.Second)
			defer cluCancel()

			cluChan := deleteClusterWithRetry(cluCtx, o.ECS.Service, service.ClusterArn)

			// wait for a done context
			select {
			case <-cluCtx.Done():
				log.Errorf("timeout waiting for successful cluster %s deletion", aws.StringValue(service.ClusterArn))
			case <-cluChan:
				log.Infof("successfully deleted cluster %s", aws.StringValue(service.ClusterArn))
			}

			if len(service.ServiceRegistries) > 0 {
				for _, r := range service.ServiceRegistries {
					srCtx, srCancel := context.WithTimeout(cleanupCtx, 120*time.Second)
					defer srCancel()

					srChan := deleteServiceRegistryWithRetry(srCtx, o.ServiceDiscovery.Service, r.RegistryArn)

					// wait for a done context
					select {
					case <-srCtx.Done():
						log.Errorf("timeout waiting for successful service registry %s deletion", aws.StringValue(r.RegistryArn))
					case out := <-srChan:
						if out == "success" {
							log.Infof("successfully deleted service registry %s", aws.StringValue(r.RegistryArn))
						}
					}
				}
			}
		}()
	}

	return &ServiceOrchestrationOutput{Service: service}, nil
}

// UpdateService updates a service and related services
func (o *Orchestrator) UpdateService(ctx context.Context, cluster, service string, input *ServiceOrchestrationUpdateInput) (*ServiceOrchestrationUpdateOutput, error) {
	if input.Tags == nil && !input.ForceNewDeployment {
		return nil, errors.New("expected update")
	}

	output := &ServiceOrchestrationUpdateOutput{}
	svc, err := getService(ctx, o.ECS.Service, aws.String(cluster), aws.String(service))
	if err != nil {
		return nil, err
	}
	output.Service = svc

	// if we have tags to update
	if input.Tags != nil {
		newTags := []*ecs.Tag{
			&ecs.Tag{
				Key:   aws.String("spinup:org"),
				Value: aws.String(o.Org),
			},
		}

		for _, t := range input.Tags {
			if aws.StringValue(t.Key) != "spinup:org" && aws.StringValue(t.Key) != "yale:org" {
				newTags = append(newTags, t)
			}

			if aws.StringValue(t.Key) == "spinup:org" || aws.StringValue(t.Key) == "yale:org" {
				if aws.StringValue(t.Value) != o.Org {
					msg := fmt.Sprintf("%s/%s is not a part of our org (%s)", cluster, service, o.Org)
					return output, errors.New(msg)
				}
			}
		}
		input.Tags = newTags

		// tag service
		if _, err = o.ECS.Service.TagResourceWithContext(ctx, &ecs.TagResourceInput{
			ResourceArn: svc.ServiceArn,
			Tags:        input.Tags,
		}); err != nil {
			return output, err
		}

		// tag cluster
		if _, err = o.ECS.Service.TagResourceWithContext(ctx, &ecs.TagResourceInput{
			ResourceArn: svc.ClusterArn,
			Tags:        input.Tags,
		}); err != nil {
			return output, err
		}
	}

	// if we are forcing a new deployment
	if input.ForceNewDeployment {
		out, err := o.ECS.Service.UpdateServiceWithContext(ctx, &ecs.UpdateServiceInput{
			ForceNewDeployment: aws.Bool(true),
			Service:            svc.ServiceName,
			Cluster:            svc.ClusterArn,
		})
		if err != nil {
			return output, err
		}
		output.Service = out.Service
	}

	return output, nil
}
