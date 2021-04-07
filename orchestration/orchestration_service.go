// Package orchestration brings together the other components of the API into a
// single orchestration interface for creating and deleting ecs services
package orchestration

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/arn"
	"github.com/aws/aws-sdk-go/aws/awsutil"
	"github.com/aws/aws-sdk-go/service/iam"
	"github.com/aws/aws-sdk-go/service/secretsmanager"

	"github.com/aws/aws-sdk-go/service/ecs"
	"github.com/aws/aws-sdk-go/service/servicediscovery"
	log "github.com/sirupsen/logrus"
)

type Tag struct {
	Key   *string
	Value *string
}

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
	// slice of tags to be applied to all resources
	Tags []*Tag
}

// ServiceOrchestrationOutput is the output structure for service orchestration
type ServiceOrchestrationOutput struct {
	// https://docs.aws.amazon.com/sdk-for-go/api/service/ecs/#Cluster
	Cluster *ecs.Cluster
	// map of container definition names to private repository credentials
	// https://docs.aws.amazon.com/sdk-for-go/api/service/secretsmanager/#CreateSecretOutput
	Credentials map[string]*secretsmanager.CreateSecretOutput
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
	Tags               []*Tag
	ForceNewDeployment bool
}

// ServiceOrchestrationUpdateOutput is the output for service orchestration updates
type ServiceOrchestrationUpdateOutput struct {
	Service        *ecs.Service
	TaskDefinition *ecs.TaskDefinition
	Credentials    map[string]interface{}
	Tags           []*Tag
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

	ct, err := cleanTags(o.Org, input.Tags)
	if err != nil {
		return nil, err
	}
	input.Tags = ct

	// setup err var, rollback function list and defer execution, note that we depend on the err variable defined above this
	var rollBackTasks []rollbackFunc
	defer func() {
		if err != nil {
			log.Errorf("recovering from error: %s, executing %d rollback tasks", err, len(rollBackTasks))
			go rollBack(&rollBackTasks)
		}
	}()

	output := &ServiceOrchestrationOutput{}
	cluster, rbfunc, err := o.processServiceCluster(ctx, input)
	if err != nil {
		return nil, err
	}
	output.Cluster = cluster
	rollBackTasks = append(rollBackTasks, rbfunc)

	creds, rbfunc, err := o.processRepositoryCredentials(ctx, input)
	if err != nil {
		return nil, err
	}
	output.Credentials = creds
	rollBackTasks = append(rollBackTasks, rbfunc)

	td, rbfunc, err := o.processTaskDefinition(ctx, input)
	if err != nil {
		return nil, err
	}
	output.TaskDefinition = td
	rollBackTasks = append(rollBackTasks, rbfunc)

	sr, rbfunc, err := o.processServiceRegistry(ctx, input)
	if err != nil {
		return nil, err
	}
	output.ServiceDiscoveryService = sr
	rollBackTasks = append(rollBackTasks, rbfunc)

	service, rbfunc, err := o.processService(ctx, input)
	if err != nil {
		return nil, err
	}
	output.Service = service
	rollBackTasks = append(rollBackTasks, rbfunc)

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

			deletedCluster, err := o.deleteCluster(cleanupCtx, service.ClusterArn)
			if err != nil {
				log.Errorf("failed cleaning up cluster: %s", err)
			}

			// if we cleaned up the cluster, we should also cleanup the default task execution role
			if deletedCluster {
				executionRoleName := fmt.Sprintf("%s-ecsTaskExecution", aws.StringValue(input.Cluster))
				if err := o.deleteDefaultTaskExecutionRole(cleanupCtx, executionRoleName); err != nil {
					log.Errorf("failed to cleanup default task execution role: %s", err)
				}
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

	// set cluster
	input.ClusterName = cluster

	active := &ServiceOrchestrationUpdateOutput{}

	// get active service
	svc, err := o.ECS.GetService(ctx, cluster, service)
	if err != nil {
		return nil, err
	}
	// GetService doesn't include tag information, lets add it
	tags, err := o.ECS.ListTags(ctx, aws.StringValue(svc.ServiceArn))
	if err != nil {
		return nil, err
	}
	svc.Tags = tags
	active.Service = svc

	// get the active task def
	tdef, err := o.ECS.GetTaskDefinition(ctx, active.Service.TaskDefinition)
	if err != nil {
		return nil, err
	}
	active.TaskDefinition = tdef

	if input.TaskDefinition != nil {

		// if the tags are empty for the task definition, apply the existing tags
		if input.TaskDefinition.Tags == nil {
			input.TaskDefinition.Tags = active.Service.Tags
		}

		// updates active.Credentials
		if err := o.processRepositoryCredentialsUpdate(ctx, input, active); err != nil {
			return nil, err
		}

		// updates active.TaskDefinition
		if err := o.processTaskDefinitionUpdate(ctx, input, active); err != nil {
			return nil, err
		}
	}

	// process updating the service
	// updates active.Service
	if err = o.processServiceUpdate(ctx, input, active); err != nil {
		return nil, err
	}

	// if the input tags are passed, clean them and use them, otherwise set to the active service tags
	if input.Tags != nil {
		input.Tags, err = cleanTags(o.Org, input.Tags)
		if err != nil {
			return nil, err
		}
	} else {
		inputTags := make([]*Tag, len(tags))
		for i, t := range tags {
			inputTags[i] = &Tag{Key: t.Key, Value: t.Value}
		}
		input.Tags = inputTags
	}

	// updates active.Tags
	if err := o.processTagsUpdate(ctx, active, input.Tags); err != nil {
		return nil, err
	}

	return active, nil
}

func (o *Orchestrator) processTagsUpdate(ctx context.Context, active *ServiceOrchestrationUpdateOutput, tags []*Tag) error {
	log.Debugf("processing tags update with tags list %s", awsutil.Prettify(tags))

	// tag all of our resources
	ecsTags := make([]*ecs.Tag, len(tags))
	iamTags := make([]*iam.Tag, len(tags))
	smTags := make([]*secretsmanager.Tag, len(tags))
	for i, t := range tags {
		ecsTags[i] = &ecs.Tag{Key: t.Key, Value: t.Value}
		iamTags[i] = &iam.Tag{Key: t.Key, Value: t.Value}
		smTags[i] = &secretsmanager.Tag{Key: t.Key, Value: t.Value}
	}

	// tag service
	if err := o.ECS.TagResource(ctx, &ecs.TagResourceInput{
		ResourceArn: active.Service.ServiceArn,
		Tags:        ecsTags,
	}); err != nil {
		return err
	}

	// tag task definition
	if err := o.ECS.TagResource(ctx, &ecs.TagResourceInput{
		ResourceArn: active.Service.TaskDefinition,
		Tags:        ecsTags,
	}); err != nil {
		return err
	}

	// tag cluster
	if err := o.ECS.TagResource(ctx, &ecs.TagResourceInput{
		ResourceArn: active.Service.ClusterArn,
		Tags:        ecsTags,
	}); err != nil {
		return err
	}

	// tag secrets
	for _, containerDef := range active.TaskDefinition.ContainerDefinitions {
		repositoryCredentials := containerDef.RepositoryCredentials
		if repositoryCredentials != nil && repositoryCredentials.CredentialsParameter != nil {
			credentialsArn := aws.StringValue(repositoryCredentials.CredentialsParameter)
			if err := o.SecretsManager.UpdateSecretTags(ctx, credentialsArn, smTags); err != nil {
				return err
			}
		}
	}

	// get the ecs task execution role arn from the active task definition
	ecsTaskExecutionRoleArn, err := arn.Parse(aws.StringValue(active.TaskDefinition.ExecutionRoleArn))
	if err != nil {
		return err
	}

	// determine the ecs task execution role name from the arn
	ecsTaskExecutionRoleName := ecsTaskExecutionRoleArn.Resource[strings.LastIndex(ecsTaskExecutionRoleArn.Resource, "/")+1:]

	// tag the ecs task execution role
	if err := o.IAM.TagRole(ctx, ecsTaskExecutionRoleName, iamTags); err != nil {
		return err
	}

	// set the active tasks to the input tasks for output
	active.Tags = tags
	// end tagging

	return err
}

func cleanTags(org string, tags []*Tag) ([]*Tag, error) {
	cleanTags := []*Tag{
		{
			Key:   aws.String("spinup:org"),
			Value: aws.String(org),
		},
	}

	for _, t := range tags {
		if aws.StringValue(t.Key) != "spinup:org" && aws.StringValue(t.Key) != "yale:org" {
			cleanTags = append(cleanTags, &Tag{Key: t.Key, Value: t.Value})
		}

		if aws.StringValue(t.Key) == "spinup:org" || aws.StringValue(t.Key) == "yale:org" {
			if aws.StringValue(t.Value) != org {
				msg := fmt.Sprintf("not a part of our org (%s)", org)
				return nil, errors.New(msg)
			}
		}
	}

	return cleanTags, nil
}
