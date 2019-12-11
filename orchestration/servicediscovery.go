package orchestration

import (
	"context"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/arn"
	"github.com/aws/aws-sdk-go/aws/awserr"

	"github.com/aws/aws-sdk-go/service/ecs"
	"github.com/aws/aws-sdk-go/service/servicediscovery"
	"github.com/aws/aws-sdk-go/service/servicediscovery/servicediscoveryiface"
	log "github.com/sirupsen/logrus"
)

// processServiceRegistry processes the service registry portion of the input.  If a service registry is provided as
// part of the service object, it will be used.  Alternatively, if a service registry definition is provided as input it
// will be created.  Otherwise the service will not be registered with service discovery.
func (o *Orchestrator) processServiceRegistry(ctx context.Context, input *ServiceOrchestrationInput) (*servicediscovery.Service, error) {
	client := o.ServiceDiscovery.Service
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
				if awsErr, ok := err.(awserr.Error); ok {
					switch aerr := awsErr.Code(); aerr {
					case servicediscovery.ErrCodeResourceInUse,
						servicediscovery.ErrCodeResourceLimitExceeded:
						log.Warnf("unable to remove service registry %s: %s", aws.StringValue(serviceArn), err)
						time.Sleep(t)
						continue
					default:
						log.Errorf("failed removing service registry %s: %s", aws.StringValue(serviceArn), err)
						srChan <- "failure"
						return
					}
				}
			}

			srChan <- "success"
			return
		}
	}()
	return srChan
}
