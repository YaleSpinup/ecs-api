package orchestration

import (
	"context"

	"github.com/aws/aws-sdk-go/aws"

	"github.com/aws/aws-sdk-go/service/ecs"
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
			Value: aws.String(o.Org),
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
