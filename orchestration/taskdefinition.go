package orchestration

import (
	"context"
	"errors"
	"fmt"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"

	"github.com/aws/aws-sdk-go/service/cloudwatchlogs"
	"github.com/aws/aws-sdk-go/service/ecs"
	log "github.com/sirupsen/logrus"
)

// processTaskDefinition processes the task definition portion of the input.  If the task definition is provided with
// the service object, it is used.  Otherwise, if the task definition is defined as input, it will be created.  If neither
// is true, an error is returned.
func (o *Orchestrator) processTaskDefinition(ctx context.Context, input *ServiceOrchestrationInput) (*ecs.TaskDefinition, error) {
	if input.Service.TaskDefinition != nil {
		log.Infof("using provided task definition %s", aws.StringValue(input.Service.TaskDefinition))
		taskDefinition, err := o.ECS.GetTaskDefinition(ctx, input.Service.TaskDefinition)
		if err != nil {
			return nil, err
		}
		return taskDefinition, nil
	} else if input.TaskDefinition != nil {
		newTags := []*ecs.Tag{
			&ecs.Tag{
				Key:   aws.String("spinup:org"),
				Value: aws.String(o.Org),
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
			path := fmt.Sprintf("%s/%s", o.Org, *input.Cluster.ClusterName)
			roleARN, err := o.IAM.DefaultTaskExecutionRole(ctx, path)
			if err != nil {
				return nil, err
			}

			input.TaskDefinition.ExecutionRoleArn = &roleARN
		}

		if len(input.TaskDefinition.RequiresCompatibilities) == 0 {
			input.TaskDefinition.RequiresCompatibilities = DefaultCompatabilities
		}

		if input.TaskDefinition.NetworkMode == nil {
			input.TaskDefinition.NetworkMode = DefaultNetworkMode
		}

		logConfiguration, err := o.processLogConfiguration(ctx, aws.StringValue(input.Cluster.ClusterName), aws.StringValue(input.TaskDefinition.Family), input.TaskDefinition.Tags)
		if err != nil {
			return nil, err
		}

		for _, cd := range input.TaskDefinition.ContainerDefinitions {
			cd.SetLogConfiguration(logConfiguration)
		}

		taskDefinition, err := o.ECS.CreateTaskDefinition(ctx, input.TaskDefinition)
		if err != nil {
			return nil, err
		}

		td := fmt.Sprintf("%s:%d", aws.StringValue(taskDefinition.Family), aws.Int64Value(taskDefinition.Revision))
		input.Service.TaskDefinition = aws.String(td)
		return taskDefinition, nil
	}

	return nil, errors.New("taskDefinition or service task definition name is required")
}

// processTaskDefinitionUpdate processes the task definition portion of the input
func (o *Orchestrator) processTaskDefinitionUpdate(ctx context.Context, input *ServiceOrchestrationUpdateInput) (*ecs.TaskDefinition, error) {
	if input.TaskDefinition == nil {
		return nil, errors.New("taskDefinition or service task definition name is required")
	}
	newTags := []*ecs.Tag{
		&ecs.Tag{
			Key:   aws.String("spinup:org"),
			Value: aws.String(o.Org),
		},
	}

	for _, t := range input.TaskDefinition.Tags {
		if aws.StringValue(t.Key) != "spinup:org" && aws.StringValue(t.Key) != "yale:org" {
			newTags = append(newTags, t)
		}
	}
	input.TaskDefinition.Tags = newTags

	if input.TaskDefinition.ExecutionRoleArn == nil {
		path := fmt.Sprintf("%s/%s", o.Org, input.ClusterName)
		roleARN, err := o.IAM.DefaultTaskExecutionRole(ctx, path)
		if err != nil {
			return nil, err
		}

		log.Debugf("setting roleARN: %s", roleARN)
		input.TaskDefinition.ExecutionRoleArn = aws.String(roleARN)
	}

	if len(input.TaskDefinition.RequiresCompatibilities) == 0 {
		log.Debugf("setting default compatabilities: %+v", DefaultCompatabilities)
		input.TaskDefinition.RequiresCompatibilities = DefaultCompatabilities
	}

	if input.TaskDefinition.NetworkMode == nil {
		log.Debugf("setting default network mode: %s", aws.StringValue(DefaultNetworkMode))
		input.TaskDefinition.NetworkMode = DefaultNetworkMode
	}

	logConfiguration, err := o.processLogConfiguration(ctx, input.ClusterName, aws.StringValue(input.TaskDefinition.Family), input.TaskDefinition.Tags)
	if err != nil {
		return nil, err
	}

	for _, cd := range input.TaskDefinition.ContainerDefinitions {
		cd.SetLogConfiguration(logConfiguration)
	}

	log.Infof("creating task definition %+v", input.TaskDefinition)

	return o.ECS.CreateTaskDefinition(ctx, input.TaskDefinition)
}

func (o *Orchestrator) processLogConfiguration(ctx context.Context, logGroup, streamPrefix string, tags []*ecs.Tag) (*ecs.LogConfiguration, error) {
	if logGroup == "" {
		return nil, errors.New("cloudwatch logs group name cannot be empty")
	}

	var tagsMap = make(map[string]*string)
	for _, tag := range tags {
		tagsMap[aws.StringValue(tag.Key)] = tag.Value
	}

	err := o.CloudWatchLogs.CreateLogGroup(ctx, &cloudwatchlogs.CreateLogGroupInput{
		LogGroupName: aws.String(logGroup),
		Tags:         tagsMap,
	})
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			case cloudwatchlogs.ErrCodeResourceAlreadyExistsException:
				log.Warnf("cloudwatch log group already exists, continuing: (%s)", err)
			default:
				return nil, err
			}
		}
	} else {
		if err := o.CloudWatchLogs.UpdateRetention(ctx, &cloudwatchlogs.PutRetentionPolicyInput{
			LogGroupName:    aws.String(logGroup),
			RetentionInDays: DefaultCloudwatchLogsRetention,
		}); err != nil {
			return nil, err
		}
	}

	return &ecs.LogConfiguration{
		LogDriver: aws.String("awslogs"),
		Options: map[string]*string{
			"awslogs-region":        aws.String("us-east-1"),
			"awslogs-create-group":  aws.String("true"),
			"awslogs-group":         aws.String(logGroup),
			"awslogs-stream-prefix": aws.String(streamPrefix),
		},
	}, nil
}
