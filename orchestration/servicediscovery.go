package orchestration

import (
	"context"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/arn"

	"github.com/aws/aws-sdk-go/service/ecs"
	"github.com/aws/aws-sdk-go/service/servicediscovery"
	log "github.com/sirupsen/logrus"
)

// processServiceRegistry processes the service registry portion of the input.  If a service registry is provided as
// part of the service object, it will be used.  Alternatively, if a service registry definition is provided as input it
// will be created.  Otherwise the service will not be registered with service discovery.
func (o *Orchestrator) processServiceRegistry(ctx context.Context, input *ServiceOrchestrationInput) (*servicediscovery.Service, error) {
	if len(input.Service.ServiceRegistries) > 0 {
		log.Infof("using provided service registry %s", aws.StringValue(input.Service.ServiceRegistries[0].RegistryArn))
		arn, err := arn.Parse(aws.StringValue(input.Service.ServiceRegistries[0].RegistryArn))
		if err != nil {
			return nil, err
		}

		resource := strings.SplitN(arn.Resource, "/", 2)
		log.Debugf("split resource into type: %s and id: %s", resource[0], resource[1])

		sd, err := o.ServiceDiscovery.GetServiceDiscoveryService(ctx, aws.String(resource[1]))
		if err != nil {
			return nil, err
		}

		return sd, nil
	} else if input.ServiceRegistry != nil {
		log.Infof("creating service registry %+v", input.ServiceRegistry)
		sd, err := o.ServiceDiscovery.CreateServiceDiscoveryService(ctx, input.ServiceRegistry)
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
