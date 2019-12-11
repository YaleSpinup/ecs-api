package orchestration

import (
	"context"
	"errors"

	"github.com/aws/aws-sdk-go/aws"

	"github.com/aws/aws-sdk-go/service/ecs"
	"github.com/aws/aws-sdk-go/service/ecs/ecsiface"
	log "github.com/sirupsen/logrus"
)

// processService processes the service input.  It normalizes inputs and creates the ECS service.
func (o *Orchestrator) processService(ctx context.Context, input *ServiceOrchestrationInput) (*ecs.Service, error) {
	client := o.ECS.Service
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
