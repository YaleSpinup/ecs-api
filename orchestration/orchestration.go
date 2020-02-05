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
	// https://docs.aws.amazon.com/sdk-for-go/api/service/ecs/#Cluster
	ClusterName string
	// https://docs.aws.amazon.com/sdk-for-go/api/service/ecs/#RegisterTaskDefinitionInput
	TaskDefinition *ecs.RegisterTaskDefinitionInput
	// map of container definition names to private repository credentials
	// https://docs.aws.amazon.com/sdk-for-go/api/service/secretsmanager/#CreateSecretInput
	Credentials map[string]*secretsmanager.CreateSecretInput
	// https://docs.aws.amazon.com/sdk-for-go/api/service/ecs/#UpdateServiceInput
	Service            *ecs.UpdateServiceInput
	Tags               []*ecs.Tag
	ForceNewDeployment bool
}

// ServiceOrchestrationUpdateOutput is the output for service orchestration updates
type ServiceOrchestrationUpdateOutput struct {
	Service        *ecs.Service
	TaskDefinition *ecs.TaskDefinition
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
	service, err := o.ECS.GetService(ctx, aws.StringValue(input.Cluster), aws.StringValue(input.Service))
	if err != nil {
		return nil, err
	}

	log.Debugf("processing delete of service %+v", service)

	log.Infof("removing service '%s'", aws.StringValue(service.ServiceArn))

	if err = o.ECS.DeleteService(ctx, &ecs.DeleteServiceInput{
		Cluster: input.Cluster,
		Service: input.Service,
		Force:   aws.Bool(true),
	}); err != nil {
		log.Errorf("error deleting service %s", err)
		return nil, err
	}

	// recursively remove the service registry and the cluster if it's empty
	// TODO: this should return a 202, not a 200
	if input.Recursive {
		log.Infof("removing '%s' dependencies recursively, asynchronously", aws.StringValue(service.ServiceArn))
		go func() {
			cleanupCtx := context.Background()

			cluCtx, cluCancel := context.WithTimeout(cleanupCtx, 120*time.Second)
			defer cluCancel()

			cluChan := o.ECS.DeleteClusterWithRetry(cluCtx, service.ClusterArn)

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

					srChan := o.ServiceDiscovery.DeleteServiceRegistryWithRetry(srCtx, r.RegistryArn)

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

			// get the active task definition to find the task definition family
			taskDefinition, err := o.ECS.GetTaskDefinition(cleanupCtx, service.TaskDefinition)
			if err != nil {
				log.Errorf("failed to get active task definition '%s': %s", aws.StringValue(service.TaskDefinition), err)
			} else {
				// list all of the revisions in the task definition family
				taskDefinitionRevisions, err := o.ECS.ListTaskDefinitionRevisions(cleanupCtx, taskDefinition.Family)
				if err != nil {
					log.Errorf("failed to get a list of task definition revisions to delete")
				} else {
					for _, revision := range taskDefinitionRevisions {

						taskDefinition, err := o.ECS.GetTaskDefinition(cleanupCtx, aws.String(revision))
						if err != nil {
							log.Errorf("failed to get task definition revisions '%s' to delete: %s", revision, err)
							continue
						}

						deletedCredentials := make(map[string]struct{})
						// for each task definition revision in the task definition family, delete any existing repository credentials, keeping track
						// of ones we delete so we don't try to re-delete them.
						// TODO: if we want to share repository credentials, we need to look for multiple container definitions using the same credentials.
						for _, cd := range taskDefinition.ContainerDefinitions {
							log.Debugf("cleaning '%s' container definition '%s' components", aws.StringValue(service.ServiceArn), aws.StringValue(cd.Name))

							if cd.RepositoryCredentials != nil && aws.StringValue(cd.RepositoryCredentials.CredentialsParameter) != "" {
								credsArn := aws.StringValue(cd.RepositoryCredentials.CredentialsParameter)

								if _, ok := deletedCredentials[credsArn]; !ok {
									_, err = o.SecretsManager.DeleteSecret(cleanupCtx, credsArn, 0)
									if err != nil {
										log.Errorf("failed to delete secretsmanager secret '%s' for %s", credsArn, aws.StringValue(service.ServiceArn))
									} else {
										deletedCredentials[credsArn] = struct{}{}
										log.Infof("successfully deleted secretsmanager secret '%s'", credsArn)
									}
								}
							}
						}

						out, err := o.ECS.DeleteTaskDefinition(cleanupCtx, aws.String(revision))
						if err != nil {
							log.Errorf("failed to delete task definition '%s': %s", revision, err)
						} else {
							log.Debugf("successfully deleted task definition revision %s: %+v", revision, out)
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
	if input.Service == nil && input.TaskDefinition == nil && input.Tags == nil && !input.ForceNewDeployment {
		return nil, errors.New("expected update")
	}

	output := &ServiceOrchestrationUpdateOutput{}
	activeSvc, err := o.ECS.GetService(ctx, cluster, service)
	if err != nil {
		return nil, err
	}
	output.Service = activeSvc

	if input.TaskDefinition != nil {
		activeTdef, err := o.ECS.GetTaskDefinition(ctx, activeSvc.TaskDefinition)
		if err != nil {
			return nil, err
		}

		// for each container definition in the active task definition, if the active container deinition *has* repository
		// credentials defined, check for an incoming container definition with the same name that doesn't already have the
		// credential defined (ie. an override) and set the credentials parameter from the active container definition
		for _, activeCdef := range activeTdef.ContainerDefinitions {
			if activeCdef.RepositoryCredentials != nil {
				for _, newCdef := range input.TaskDefinition.ContainerDefinitions {
					if aws.StringValue(activeCdef.Name) == aws.StringValue(newCdef.Name) {
						if newCdef.RepositoryCredentials == nil {
							log.Debugf("setting repo credentials from active task def/container def: %+v", activeCdef.RepositoryCredentials)
							newCdef.SetRepositoryCredentials(activeCdef.RepositoryCredentials)
						}
						break
					}
				}
			}
		}

		// set cluster
		input.ClusterName = cluster

		td, err := o.processTaskDefinitionUpdate(ctx, input)
		if err != nil {
			return nil, err
		}

		log.Debugf("orchestrated create of task definition: %+v", td)

		if input.Service == nil {
			input.Service = &ecs.UpdateServiceInput{}
		}

		// apply new task definition ARN to the service update
		input.Service.TaskDefinition = td.TaskDefinitionArn
		output.TaskDefinition = td
	}

	if input.Service != nil {
		// set cluster and service, disallow assigning public IP
		u := input.Service
		u.Cluster = activeSvc.ClusterArn
		u.Service = activeSvc.ServiceArn
		if u.NetworkConfiguration != nil && u.NetworkConfiguration.AwsvpcConfiguration != nil {
			u.NetworkConfiguration.AwsvpcConfiguration.AssignPublicIp = aws.String("DISABLED")
		}

		out, err := o.ECS.UpdateService(ctx, u)
		if err != nil {
			return output, err
		}

		// override active service with new service
		activeSvc = out.Service

		output.Service = out.Service
	} else if input.ForceNewDeployment {
		out, err := o.ECS.UpdateService(ctx, &ecs.UpdateServiceInput{
			ForceNewDeployment: aws.Bool(true),
			Service:            activeSvc.ServiceName,
			Cluster:            activeSvc.ClusterArn,
		})
		if err != nil {
			return output, err
		}
		output.Service = out.Service
	}

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
			ResourceArn: activeSvc.ServiceArn,
			Tags:        input.Tags,
		}); err != nil {
			return output, err
		}

		// tag task definition
		if _, err = o.ECS.Service.TagResourceWithContext(ctx, &ecs.TagResourceInput{
			ResourceArn: activeSvc.TaskDefinition,
			Tags:        input.Tags,
		}); err != nil {
			return output, err
		}

		// tag cluster
		if _, err = o.ECS.Service.TagResourceWithContext(ctx, &ecs.TagResourceInput{
			ResourceArn: activeSvc.ClusterArn,
			Tags:        input.Tags,
		}); err != nil {
			return output, err
		}
	}

	return output, nil
}
