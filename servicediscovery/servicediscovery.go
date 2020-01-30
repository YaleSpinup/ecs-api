package servicediscovery

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/YaleSpinup/ecs-api/common"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/arn"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/servicediscovery"
	"github.com/aws/aws-sdk-go/service/servicediscovery/servicediscoveryiface"

	log "github.com/sirupsen/logrus"
)

// ServiceDiscovery is the internal service discovery object which holds session
// and configuration information
type ServiceDiscovery struct {
	Service servicediscoveryiface.ServiceDiscoveryAPI
}

// NewSession builds a new aws servicediscovery session
func NewSession(account common.Account) ServiceDiscovery {
	s := ServiceDiscovery{}
	log.Infof("Creating new session with key id %s in region %s", account.Akid, account.Region)
	sess := session.Must(session.NewSession(&aws.Config{
		Credentials: credentials.NewStaticCredentials(account.Akid, account.Secret, ""),
		Region:      aws.String(account.Region),
	}))
	s.Service = servicediscovery.New(sess)
	return s
}

// GetServiceDiscoveryService gets the details of a service discovery service
func (s *ServiceDiscovery) GetServiceDiscoveryService(ctx context.Context, id *string) (*servicediscovery.Service, error) {
	output, err := s.Service.GetServiceWithContext(ctx, &servicediscovery.GetServiceInput{Id: id})
	if err != nil {
		return nil, err
	}
	return output.Service, err
}

// CreateServiceDiscoveryService creates a service discovery service
func (s *ServiceDiscovery) CreateServiceDiscoveryService(ctx context.Context, input *servicediscovery.CreateServiceInput) (*servicediscovery.Service, error) {
	input.SetHealthCheckCustomConfig(&servicediscovery.HealthCheckCustomConfig{FailureThreshold: aws.Int64(1)})
	output, err := s.Service.CreateServiceWithContext(ctx, input)
	if err != nil {
		return nil, err
	}
	return output.Service, err
}

// DeleteServiceRegistry removes a service discovery service by it's ID
func (s *ServiceDiscovery) DeleteServiceRegistry(ctx context.Context, serviceArn *string) error {
	// parse the ARN into it's component parts and split the resource/resource-id
	a, err := arn.Parse(aws.StringValue(serviceArn))
	if err != nil {
		return err
	}

	resource := strings.SplitN(a.Resource, "/", 2)
	output, err := s.Service.DeleteServiceWithContext(ctx, &servicediscovery.DeleteServiceInput{
		Id: aws.String(resource[1]),
	})

	if err != nil {
		return err
	}

	log.Debugf("output from service discovery service delete:\n%+v", output)
	return nil
}

// DeleteServiceRegistryWithRetry continues to retry deleting a service registration until the context is cancelled or it succeeds
func (s *ServiceDiscovery) DeleteServiceRegistryWithRetry(ctx context.Context, serviceArn *string) chan string {
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
			err := s.DeleteServiceRegistry(ctx, serviceArn)
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

// ServiceEndpoint takes the service discovery client and the ecs service registry configuration.  It first gets the
// details of the service registry given and from that determines the namespace ID.  The endpoint string is determined by
// combining the service registry service name(hostname) and the namespace name (domain name).
func (s *ServiceDiscovery) ServiceEndpoint(ctx context.Context, registryArn string) (*string, error) {
	serviceResistryArn, err := arn.Parse(registryArn)
	if err != nil {
		log.Errorf("error parsing servicediscovery service ARN %s", err)
		return nil, err
	}

	if serviceResistryArn.Resource != "" {
		log.Debugf("getting service registry service with id %s", serviceResistryArn.Resource)

		// serviceRegistryArn.Resource is of the format 'service/srv-xxxxxxxxxxxxx', but GetServiceInput needs just the ID
		serviceID := strings.SplitN(serviceResistryArn.Resource, "/", 2)
		sdOutput, err := s.Service.GetServiceWithContext(ctx, &servicediscovery.GetServiceInput{
			Id: aws.String(serviceID[1]),
		})

		if err != nil {
			log.Errorf("error getting service from ID %s: %s", serviceID[1], err)
			return nil, err
		}

		if nsID := aws.StringValue(sdOutput.Service.DnsConfig.NamespaceId); nsID != "" {
			namespaceOutput, err := s.Service.GetNamespaceWithContext(ctx, &servicediscovery.GetNamespaceInput{
				Id: aws.String(nsID),
			})
			if err != nil {
				log.Errorf("error getting namespace %s", err)
				return nil, err
			}
			endpoint := fmt.Sprintf("%s.%s", aws.StringValue(sdOutput.Service.Name), aws.StringValue(namespaceOutput.Namespace.Name))
			return &endpoint, nil
		}
	}

	log.Warnf("service discovery endpoint not found")
	return nil, nil
}
