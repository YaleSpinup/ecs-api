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
func (o *Orchestrator) processTaskDefinition(ctx context.Context, input *ServiceOrchestrationInput) (*ecs.TaskDefinition, rollbackFunc, error) {
	rbfunc := func(_ context.Context) error {
		log.Infof("processTaskDefinition rollback, nothing to do")
		return nil
	}

	if input.Service.TaskDefinition != nil {
		log.Infof("using provided task definition %s", aws.StringValue(input.Service.TaskDefinition))
		taskDefinition, err := o.ECS.GetTaskDefinition(ctx, input.Service.TaskDefinition)
		if err != nil {
			return nil, rbfunc, err
		}
		return taskDefinition, rbfunc, nil
	} else if input.TaskDefinition != nil {
		ecsTags := make([]*ecs.Tag, len(input.Tags))
		for i, t := range input.Tags {
			ecsTags[i] = &ecs.Tag{Key: t.Key, Value: t.Value}
		}
		input.TaskDefinition.Tags = ecsTags

		log.Infof("creating task definition %+v", input.TaskDefinition)

		if input.TaskDefinition.ExecutionRoleArn == nil {
			path := fmt.Sprintf("%s/%s", o.Org, *input.Cluster.ClusterName)
			roleARN, err := o.DefaultTaskExecutionRole(ctx, path)
			if err != nil {
				return nil, rbfunc, err
			}

			input.TaskDefinition.ExecutionRoleArn = &roleARN
		}

		if len(input.TaskDefinition.RequiresCompatibilities) == 0 {
			input.TaskDefinition.RequiresCompatibilities = DefaultCompatabilities
		}

		if input.TaskDefinition.NetworkMode == nil {
			input.TaskDefinition.NetworkMode = DefaultNetworkMode
		}

		logConfiguration, err := o.processLogConfiguration(ctx, aws.StringValue(input.Cluster.ClusterName), aws.StringValue(input.TaskDefinition.Family), input.Tags)
		if err != nil {
			return nil, rbfunc, err
		}

		for _, cd := range input.TaskDefinition.ContainerDefinitions {
			cd.SetLogConfiguration(logConfiguration)
		}

		taskDefinition, err := o.ECS.CreateTaskDefinition(ctx, input.TaskDefinition)
		if err != nil {
			return nil, rbfunc, err
		}

		rbfunc = func(ctx context.Context) error {
			id := aws.StringValue(taskDefinition.TaskDefinitionArn)
			log.Debugf("rolling back task definition %s", id)

			_, err := o.ECS.DeleteTaskDefinition(ctx, taskDefinition.TaskDefinitionArn)
			if err != nil {
				return fmt.Errorf("failed to delete task definition %s: %s", id, err)
			}

			log.Infof("successfully rolled back task definition %s", id)
			return nil
		}

		td := fmt.Sprintf("%s:%d", aws.StringValue(taskDefinition.Family), aws.Int64Value(taskDefinition.Revision))
		input.Service.TaskDefinition = aws.String(td)
		return taskDefinition, rbfunc, nil
	}

	return nil, rbfunc, errors.New("taskDefinition or service task definition name is required")
}

// processTaskDefinitionUpdate processes the task definition portion of the input
func (o *Orchestrator) processTaskDefinitionUpdate(ctx context.Context, input *ServiceOrchestrationUpdateInput, active *ServiceOrchestrationUpdateOutput) error {
	if input.TaskDefinition == nil {
		return errors.New("taskDefinition or service task definition name is required")
	}

	if input.TaskDefinition.ExecutionRoleArn == nil {
		path := fmt.Sprintf("%s/%s", o.Org, input.ClusterName)
		roleARN, err := o.DefaultTaskExecutionRole(ctx, path)
		if err != nil {
			return err
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

	logConfiguration, err := o.processLogConfiguration(ctx, input.ClusterName, aws.StringValue(input.TaskDefinition.Family), input.Tags)
	if err != nil {
		return err
	}

	for _, cd := range input.TaskDefinition.ContainerDefinitions {
		cd.SetLogConfiguration(logConfiguration)
	}

	log.Infof("creating task definition %+v", input.TaskDefinition)

	out, err := o.ECS.CreateTaskDefinition(ctx, input.TaskDefinition)
	if err != nil {
		return err
	}
	active.TaskDefinition = out

	if input.Service == nil {
		input.Service = &ecs.UpdateServiceInput{}
	}

	// apply new task definition ARN to the service update
	input.Service.TaskDefinition = active.TaskDefinition.TaskDefinitionArn

	return nil
}

func (o *Orchestrator) processLogConfiguration(ctx context.Context, logGroup, streamPrefix string, tags []*Tag) (*ecs.LogConfiguration, error) {
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
	}

	if err := o.CloudWatchLogs.UpdateRetention(ctx, &cloudwatchlogs.PutRetentionPolicyInput{
		LogGroupName:    aws.String(logGroup),
		RetentionInDays: DefaultCloudwatchLogsRetention,
	}); err != nil {
		return nil, err
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
