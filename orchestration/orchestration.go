// Package orchestration brings together the other components of the API into a
// single orchestration interface for creating and deleting ecs services
package orchestration

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/YaleSpinup/ecs-api/iam"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/arn"

	"github.com/aws/aws-sdk-go/service/ecs"
	"github.com/aws/aws-sdk-go/service/ecs/ecsiface"
	"github.com/aws/aws-sdk-go/service/servicediscovery"
	"github.com/aws/aws-sdk-go/service/servicediscovery/servicediscoveryiface"
	log "github.com/sirupsen/logrus"
)

var (
	// DefaultCompatabilities sets the default task definition compatabilities to
	// Fargate.  By default, we won't support standard ECS.
	// https://docs.aws.amazon.com/AmazonECS/latest/APIReference/API_TaskDefinition.html
	DefaultCompatabilities = []*string{
		aws.String("FARGATE"),
	}
	// DefaultNetworkMode sets the default networking more for task definitions created
	// by the api.  Currently, Fargate only supports vpc networking.
	// https://docs.aws.amazon.com/AmazonECS/latest/developerguide/task-networking.html
	DefaultNetworkMode = aws.String("awsvpc")
	// DefaultLaunchType sets the default launch type to Fargate
	DefaultLaunchType = aws.String("FARGATE")
	// DefaultPublic disables the setting of public IPs on ENIs by default
	DefaultPublic = aws.String("DISABLED")
	// DefaultSubnets sets a list of default subnets to attach ENIs
	DefaultSubnets = []*string{}
	// DefaultSecurityGroups sets a list of default sgs to attach to ENIs
	DefaultSecurityGroups = []*string{}
	// Org is the organization where this orchestration runs
	Org = ""
)

// Orchestrator holds the service discovery client, iam client, ecs client, input, and output
type Orchestrator struct {
	// https://docs.aws.amazon.com/sdk-for-go/api/service/ecs/#ECS
	ECS *ecs.ECS
	// https://docs.aws.amazon.com/sdk-for-go/api/service/iam/#IAM
	IAM iam.IAM
	// https://docs.aws.amazon.com/sdk-for-go/api/service/servicediscovery/#ServiceDiscovery
	ServiceDiscovery *servicediscovery.ServiceDiscovery
	// Token is a uniqueness token for calls to AWS
	Token string
}

// ServiceOrchestrationInput encapsulates a single request for a service
type ServiceOrchestrationInput struct {
	// https://docs.aws.amazon.com/sdk-for-go/api/service/ecs/#CreateClusterInput
	Cluster *ecs.CreateClusterInput
	// https://docs.aws.amazon.com/sdk-for-go/api/service/ecs/#RegisterTaskDefinitionInput
	TaskDefinition *ecs.RegisterTaskDefinitionInput
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
	ServiceTags             *ecs.Tag
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
	service, err := getService(ctx, o.ECS, input.Cluster, input.Service)
	if err != nil {
		return nil, err
	}

	log.Infof("removing service\n%+v", service)
	err = deleteService(ctx, o.ECS, input)
	if err != nil {
		log.Errorf("error deleting service %s", err)
		return nil, err
	}

	// recursively remove the service registry and the cluster if it's empty
	if input.Recursive {
		if len(service.ServiceRegistries) > 0 {
			for _, r := range service.ServiceRegistries {
				srCtx, srCancel := context.WithTimeout(ctx, 10*time.Second)
				defer srCancel()

				srChan := deleteServiceRegistryWithRetry(srCtx, o.ServiceDiscovery, r.RegistryArn)

				// wait for a done context
				select {
				case <-srCtx.Done():
					log.Errorf("timeout waiting for successful service registry %s deletion", aws.StringValue(r.RegistryArn))
				case <-srChan:
					log.Infof("successfully deleted service registry %s", aws.StringValue(r.RegistryArn))
				}
			}
		}

		cluCtx, cluCancel := context.WithTimeout(ctx, 10*time.Second)
		defer cluCancel()

		cluChan := deleteClusterWithRetry(cluCtx, o.ECS, service.ClusterArn)

		// wait for a done context
		select {
		case <-cluCtx.Done():
			log.Errorf("timeout waiting for successful cluster %s deletion", aws.StringValue(service.ClusterArn))
		case <-cluChan:
			log.Infof("successfully deleted cluster %s", aws.StringValue(service.ClusterArn))
		}

	}

	return &ServiceOrchestrationOutput{Service: service}, nil
}

// processCluster processes the cluster portion of the input.  If the cluster is defined on ths service object
// it will be used, otherwise if the ClusterName is given, it will be created.  If neither is provided, an error
// will be returned.
func (o *Orchestrator) processCluster(ctx context.Context, input *ServiceOrchestrationInput) (*ecs.Cluster, error) {
	client := o.ECS
	if input.Service.Cluster != nil {
		log.Infof("Using provided cluster name %s", aws.StringValue(input.Cluster.ClusterName))

		cluster, err := getCluster(ctx, client, input.Service.Cluster)
		if err != nil {
			return nil, err
		}

		log.Debugf("Got cluster %+v", cluster)
		return cluster, nil
	} else if input.Cluster != nil {
		log.Infof("Creating cluster %s", aws.StringValue(input.Cluster.ClusterName))

		newTags := []*ecs.Tag{
			&ecs.Tag{
				Key:   aws.String("spinup:org"),
				Value: aws.String(Org),
			},
		}

		for _, t := range input.Cluster.Tags {
			if aws.StringValue(t.Key) != "spinup:org" && aws.StringValue(t.Key) != "yale:org" {
				newTags = append(newTags, t)
			}
		}
		input.Cluster.Tags = newTags

		cluster, err := createCluster(ctx, client, input.Cluster)
		if err != nil {
			return nil, err
		}
		log.Debugf("Created cluster %+v", cluster)
		input.Service.Cluster = cluster.ClusterName
		return cluster, nil
	}
	return nil, errors.New("A new or existing cluster is required")
}

// createCluster creates a cluster with context and name
func createCluster(ctx context.Context, client ecsiface.ECSAPI, cluster *ecs.CreateClusterInput) (*ecs.Cluster, error) {
	output, err := client.CreateClusterWithContext(ctx, cluster)
	if err != nil {
		return nil, err
	}
	return output.Cluster, err
}

// getCluster gets the details of a cluster with context by the cluster name
func getCluster(ctx context.Context, client ecsiface.ECSAPI, name *string) (*ecs.Cluster, error) {
	output, err := client.DescribeClustersWithContext(ctx, &ecs.DescribeClustersInput{
		Clusters: []*string{name},
	})

	if err != nil {
		return nil, err
	}

	if len(output.Failures) > 0 {
		log.Warnf("describe clusters %s returned failures %+v", aws.StringValue(name), output.Failures)
	}

	if len(output.Clusters) == 0 {
		msg := fmt.Sprintf("cluster %s not found", aws.StringValue(name))
		return nil, errors.New(msg)
	} else if len(output.Clusters) > 1 {
		return nil, errors.New("unexpected number of clusters returned")
	}

	return output.Clusters[0], err
}

// deleteCluster deletes a(n empty) cluster
func deleteCluster(ctx context.Context, client ecsiface.ECSAPI, name *string) error {
	_, err := client.DeleteClusterWithContext(ctx, &ecs.DeleteClusterInput{Cluster: name})
	if err != nil {
		log.Errorf("error deleting cluster %s: %s", aws.StringValue(name), err)
		return err
	}
	log.Infof("successfully deleted cluster %s", aws.StringValue(name))
	return nil
}

// deleteClusterWithRetry continues to retry deleting a cluster until the context is cancelled or it succeeds
func deleteClusterWithRetry(ctx context.Context, client ecsiface.ECSAPI, arn *string) chan string {
	cluChan := make(chan string, 1)
	go func() {
		t := 1 * time.Second
		for {
			if ctx.Err() != nil {
				log.Debug("cluster delete context is cancelled")
				return
			}

			cluster, err := getCluster(ctx, client, arn)
			if err != nil {
				log.Errorf("error finding cluster to delete %s: %s", aws.StringValue(arn), err)
				cluChan <- "unknown"
				return
			}
			log.Debugf("found cluster %+v", cluster)

			t *= 2
			c := aws.Int64Value(cluster.RegisteredContainerInstancesCount)
			if c > 0 {
				log.Infof("found cluster %s, but registered instance count is > 0 (%d)", aws.StringValue(cluster.ClusterName), c)
				time.Sleep(t)
				continue
			} else {
				log.Infof("found cluster %s with registered instance count of 0, attempting to delete", aws.StringValue(cluster.ClusterName))
				err := deleteCluster(ctx, client, arn)
				if err != nil {
					log.Errorf("error removing cluster %s: %s", aws.StringValue(arn), err)
					time.Sleep(t)
					continue
				}
			}

			cluChan <- "success"
			return
		}
	}()

	return cluChan
}

// processTaskDefinition processes the task definition portion of the input.  If the task definition is provided with
// the service object, it is used.  Otherwise, if the task definition is defined as input, it will be created.  If neither
// is true, an error is returned.
func (o *Orchestrator) processTaskDefinition(ctx context.Context, input *ServiceOrchestrationInput) (*ecs.TaskDefinition, error) {
	client := o.ECS

	if input.Service.TaskDefinition != nil {
		log.Infof("using provided task definition %s", aws.StringValue(input.Service.TaskDefinition))
		taskDefinition, err := getTaskDefinition(ctx, client, input.Service.TaskDefinition)
		if err != nil {
			return nil, err
		}
		return taskDefinition, nil
	} else if input.TaskDefinition != nil {

		newTags := []*ecs.Tag{
			&ecs.Tag{
				Key:   aws.String("spinup:org"),
				Value: aws.String(Org),
			},
		}

		for _, t := range input.TaskDefinition.Tags {
			if aws.StringValue(t.Key) != "spinup:org" && aws.StringValue(t.Key) != "yale:org" {
				newTags = append(newTags, t)
			}
		}
		input.TaskDefinition.Tags = newTags

		log.Infof("creating task definition %+v", input.TaskDefinition)

		if input.TaskDefinition.ExecutionRoleArn == nil {
			path := fmt.Sprintf("%s/%s", Org, *input.Cluster.ClusterName)
			roleARN, err := o.IAM.DefaultTaskExecutionRole(ctx, path)
			if err != nil {
				return nil, err
			}

			input.TaskDefinition.ExecutionRoleArn = &roleARN
		}

		taskDefinition, err := createTaskDefinition(ctx, client, input.TaskDefinition)
		if err != nil {
			return nil, err
		}

		td := fmt.Sprintf("%s:%d", aws.StringValue(taskDefinition.Family), aws.Int64Value(taskDefinition.Revision))
		input.Service.TaskDefinition = aws.String(td)
		return taskDefinition, nil
	}

	return nil, errors.New("taskDefinition or service task definition name is required")
}

// createTaskDefinition creates a task definition with context and input
func createTaskDefinition(ctx context.Context, client ecsiface.ECSAPI, input *ecs.RegisterTaskDefinitionInput) (*ecs.TaskDefinition, error) {
	if len(input.RequiresCompatibilities) == 0 {
		input.RequiresCompatibilities = DefaultCompatabilities
	}

	if input.NetworkMode == nil {
		input.NetworkMode = DefaultNetworkMode
	}

	output, err := client.RegisterTaskDefinitionWithContext(ctx, input)
	if err != nil {
		return nil, err
	}

	return output.TaskDefinition, err
}

// getTaskDefinition gets a task definition with context by name
func getTaskDefinition(ctx context.Context, client ecsiface.ECSAPI, name *string) (*ecs.TaskDefinition, error) {
	output, err := client.DescribeTaskDefinitionWithContext(ctx, &ecs.DescribeTaskDefinitionInput{
		TaskDefinition: name,
	})

	if err != nil {
		return nil, err
	}

	return output.TaskDefinition, err
}

// processServiceRegistry processes the service registry portion of the input.  If a service registry is provided as
// part of the service object, it will be used.  Alternatively, if a service registry definition is provided as input it
// will be created.  Otherwise the service will not be registered with service discovery.
func (o *Orchestrator) processServiceRegistry(ctx context.Context, input *ServiceOrchestrationInput) (*servicediscovery.Service, error) {
	client := o.ServiceDiscovery
	if len(input.Service.ServiceRegistries) > 0 {
		log.Infof("using provided service registry %s", aws.StringValue(input.Service.ServiceRegistries[0].RegistryArn))
		arn, err := arn.Parse(aws.StringValue(input.Service.ServiceRegistries[0].RegistryArn))
		if err != nil {
			return nil, err
		}

		resource := strings.SplitN(arn.Resource, "/", 2)
		log.Debugf("split resource into type: %s and id: %s", resource[0], resource[1])

		sd, err := getServiceDiscoveryService(ctx, client, aws.String(resource[1]))
		if err != nil {
			return nil, err
		}

		return sd, nil
	} else if input.ServiceRegistry != nil {
		log.Infof("creating service registry %+v", input.ServiceRegistry)
		sd, err := createServiceDiscoveryService(ctx, client, input.ServiceRegistry)
		if err != nil {
			return nil, err
		}

		input.Service.ServiceRegistries = append(input.Service.ServiceRegistries, &ecs.ServiceRegistry{
			RegistryArn: sd.Arn,
		})

		return sd, nil
	}

	log.Warn("service discovery registry was not provided, not registering")
	return nil, nil
}

// getServiceDiscoveryService gets the details of a service discovery service
func getServiceDiscoveryService(ctx context.Context, client servicediscoveryiface.ServiceDiscoveryAPI, id *string) (*servicediscovery.Service, error) {
	output, err := client.GetServiceWithContext(ctx, &servicediscovery.GetServiceInput{Id: id})
	if err != nil {
		return nil, err
	}
	return output.Service, err
}

// createServiceDiscoveryService creates a service discovery service
func createServiceDiscoveryService(ctx context.Context, client servicediscoveryiface.ServiceDiscoveryAPI, input *servicediscovery.CreateServiceInput) (*servicediscovery.Service, error) {
	input.SetHealthCheckCustomConfig(&servicediscovery.HealthCheckCustomConfig{FailureThreshold: aws.Int64(1)})
	output, err := client.CreateServiceWithContext(ctx, input)
	if err != nil {
		return nil, err
	}
	return output.Service, err
}

// deleteServiceRegistry removes a service discovery service by it's ID
func deleteServiceRegistry(ctx context.Context, client *servicediscovery.ServiceDiscovery, serviceArn *string) error {
	// parse the ARN into it's component parts and split the resource/resource-id
	a, err := arn.Parse(aws.StringValue(serviceArn))
	if err != nil {
		return err
	}

	resource := strings.SplitN(a.Resource, "/", 2)
	output, err := client.DeleteServiceWithContext(ctx, &servicediscovery.DeleteServiceInput{
		Id: aws.String(resource[1]),
	})

	if err != nil {
		return err
	}

	log.Debugf("output from service discovery service delete:\n%+v", output)
	return nil
}

// deleteServiceRegistryWithRetry continues to retry deleting a service registration until the context is cancelled or it succeeds
func deleteServiceRegistryWithRetry(ctx context.Context, client *servicediscovery.ServiceDiscovery, serviceArn *string) chan string {
	srChan := make(chan string, 1)
	go func() {
		t := 1 * time.Second
		for {
			if ctx.Err() != nil {
				log.Debug("service registration delete context is cancelled")
				return
			}

			t *= 2
			log.Debugf("attempting to remove service registry: %s", aws.StringValue(serviceArn))
			err := deleteServiceRegistry(ctx, client, serviceArn)
			if err != nil {
				log.Warnf("failed removing service registry %s: %s", aws.StringValue(serviceArn), err)
				time.Sleep(t)
				continue
			}

			srChan <- "success"
			return
		}
	}()
	return srChan
}

// processService processes the service input.  It normalizes inputs and creates the ECS service.
func (o *Orchestrator) processService(ctx context.Context, input *ServiceOrchestrationInput) (*ecs.Service, error) {
	client := o.ECS
	if input.Service.ClientToken == nil {
		input.Service.ClientToken = aws.String(o.Token)
	}

	if input.Service.NetworkConfiguration == nil {
		input.Service.NetworkConfiguration = &ecs.NetworkConfiguration{
			AwsvpcConfiguration: &ecs.AwsVpcConfiguration{
				AssignPublicIp: DefaultPublic,
				SecurityGroups: DefaultSecurityGroups,
				Subnets:        DefaultSubnets,
			},
		}
	}

	newTags := []*ecs.Tag{
		&ecs.Tag{
			Key:   aws.String("spinup:org"),
			Value: aws.String(Org),
		},
	}

	for _, t := range input.Service.Tags {
		if aws.StringValue(t.Key) != "spinup:org" && aws.StringValue(t.Key) != "yale:org" {
			newTags = append(newTags, t)
		}
	}
	input.Service.Tags = newTags

	if input.Service.LaunchType == nil {
		input.Service.LaunchType = DefaultLaunchType
	}
	log.Debugf("processing service with input:\n%+v", input.Service)
	output, err := client.CreateServiceWithContext(ctx, input.Service)
	if err != nil {
		return nil, err
	}

	return output.Service, nil
}

// getService describes an ECS service in a cluster by the service name
func getService(ctx context.Context, client ecsiface.ECSAPI, cluster, service *string) (*ecs.Service, error) {
	output, err := client.DescribeServicesWithContext(ctx, &ecs.DescribeServicesInput{
		Cluster:  cluster,
		Services: []*string{service},
	})

	if err != nil {
		return nil, err
	}

	if len(output.Services) != 1 {
		return nil, errors.New("unexpected service length in describe services")
	}

	return output.Services[0], nil
}

// deleteService removes an ECS service in a cluster by the service name (forcefully)
func deleteService(ctx context.Context, client ecsiface.ECSAPI, input *ServiceDeleteInput) error {
	output, err := client.DeleteServiceWithContext(ctx, &ecs.DeleteServiceInput{
		Cluster: input.Cluster,
		Service: input.Service,
		Force:   aws.Bool(true),
	})

	log.Debugf("output from delete service:\n%+v", output)
	return err
}
