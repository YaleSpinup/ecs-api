package orchestration

import (
	"context"
	"errors"
	"fmt"

	"github.com/aws/aws-sdk-go/aws"

	"github.com/aws/aws-sdk-go/service/ecs"
	"github.com/aws/aws-sdk-go/service/ecs/ecsiface"
	log "github.com/sirupsen/logrus"
)

// processTaskDefinition processes the task definition portion of the input.  If the task definition is provided with
// the service object, it is used.  Otherwise, if the task definition is defined as input, it will be created.  If neither
// is true, an error is returned.
func (o *Orchestrator) processTaskDefinition(ctx context.Context, input *ServiceOrchestrationInput) (*ecs.TaskDefinition, error) {
	client := o.ECS.Service

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
