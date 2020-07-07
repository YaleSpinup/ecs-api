package orchestration

import (
	"context"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/arn"

	"github.com/aws/aws-sdk-go/service/ecs"
	"github.com/aws/aws-sdk-go/service/servicediscovery"
	log "github.com/sirupsen/logrus"
)

// processServiceRegistry processes the service registry portion of the input.  If a service registry is provided as
// part of the service object, it will be used.  Alternatively, if a service registry definition is provided as input it
// will be created.  Otherwise the service will not be registered with service discovery.
func (o *Orchestrator) processServiceRegistry(ctx context.Context, input *ServiceOrchestrationInput) (*servicediscovery.Service, rollbackFunc, error) {
	rbfunc := func(_ context.Context) error {
		log.Infof("processServiceRegistry rollback, nothing to do")
		return nil
	}

	if len(input.Service.ServiceRegistries) > 0 {
		log.Infof("using provided service registry %s", aws.StringValue(input.Service.ServiceRegistries[0].RegistryArn))
		arn, err := arn.Parse(aws.StringValue(input.Service.ServiceRegistries[0].RegistryArn))
		if err != nil {
			return nil, rbfunc, err
		}

		resource := strings.SplitN(arn.Resource, "/", 2)
		log.Debugf("split resource into type: %s and id: %s", resource[0], resource[1])

		sd, err := o.ServiceDiscovery.GetServiceDiscoveryService(ctx, aws.String(resource[1]))
		if err != nil {
			return nil, rbfunc, err
		}

		return sd, rbfunc, nil
	} else if input.ServiceRegistry != nil {
		log.Infof("creating service registry %+v", input.ServiceRegistry)
		sd, err := o.ServiceDiscovery.CreateServiceDiscoveryService(ctx, input.ServiceRegistry)
		if err != nil {
			return nil, rbfunc, err
		}

		input.Service.ServiceRegistries = append(input.Service.ServiceRegistries, &ecs.ServiceRegistry{
			RegistryArn: sd.Arn,
		})

		rbfunc = func(ctx context.Context) error {
			srCtx, cancel := context.WithTimeout(ctx, 60*time.Second)
			defer cancel()

			srChan := o.ServiceDiscovery.DeleteServiceRegistryWithRetry(srCtx, sd.Arn)
			select {
			case <-srCtx.Done():
				log.Errorf("timeout waiting for successful service registry %s rollback", aws.StringValue(sd.Arn))
			case out := <-srChan:
				if out == "success" {
					log.Infof("successfully rolled back service registry %s", aws.StringValue(sd.Arn))
				}
			}

			return nil
		}

		return sd, rbfunc, nil
	}

	log.Warn("service discovery registry was not provided, not registering")
	return nil, rbfunc, nil
}
