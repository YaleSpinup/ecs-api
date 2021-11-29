package orchestration

import (
	"context"
	"errors"
	"fmt"

	"github.com/YaleSpinup/apierror"
	"github.com/aws/aws-sdk-go/aws"

	"github.com/aws/aws-sdk-go/service/cloudwatchlogs"
	"github.com/aws/aws-sdk-go/service/ecs"
	log "github.com/sirupsen/logrus"
)

// processTaskDefinitionCreate processes the task definition portion of the input.  If the task definition is defined as input,
// it will be created otherwiuse an error is returned.
func (o *Orchestrator) processTaskDefinitionCreate(ctx context.Context, input *ServiceOrchestrationInput) (*ecs.TaskDefinition, rollbackFunc, error) {
	rbfunc := defaultRbfunc("processTaskDefinitionCreate")

	if input == nil || input.TaskDefinition == nil {
		return nil, rbfunc, apierror.New(apierror.ErrBadRequest, "task definition cannot be nil", nil)
	}

	if input.Cluster == nil || input.Cluster.ClusterName == nil {
		return nil, rbfunc, apierror.New(apierror.ErrBadRequest, "cluster cannot be nil", nil)
	}

	if input.Service == nil {
		return nil, rbfunc, apierror.New(apierror.ErrBadRequest, "service cannot be nil", nil)
	}

	log.Debugf("processing task definition create for a service %+v", input.TaskDefinition)

	input.TaskDefinition.Tags = ecsTags(input.Tags)

	// path is org/clustername
	path := fmt.Sprintf("%s/%s", o.Org, aws.StringValue(input.Cluster.ClusterName))

	// role name is clustername-ecsTaskExecution
	roleName := fmt.Sprintf("%s-ecsTaskExecution", aws.StringValue(input.Cluster.ClusterName))

	roleARN, err := o.DefaultTaskExecutionRole(ctx, path, roleName, input.Tags)
	if err != nil {
		return nil, rbfunc, err
	}

	input.TaskDefinition.ExecutionRoleArn = aws.String(roleARN)
	input.TaskDefinition.TaskRoleArn = aws.String(roleARN)
	input.TaskDefinition.RequiresCompatibilities = DefaultCompatabilities
	input.TaskDefinition.NetworkMode = DefaultNetworkMode

	logConfiguration, err := o.defaultLogConfiguration(ctx, aws.StringValue(input.Cluster.ClusterName), aws.StringValue(input.TaskDefinition.Family), input.Tags)
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

func (o *Orchestrator) processTaskDefTaskDefinitionCreate(ctx context.Context, input *TaskDefCreateOrchestrationInput) (*ecs.TaskDefinition, rollbackFunc, error) {
	rbfunc := defaultRbfunc("processTaskTaskDefinitionCreate")

	if input == nil || input.TaskDefinition == nil {
		return nil, rbfunc, apierror.New(apierror.ErrBadRequest, "task definition cannot be nil", nil)
	}

	if input.Cluster == nil || input.Cluster.ClusterName == nil {
		return nil, rbfunc, apierror.New(apierror.ErrBadRequest, "cluster cannot be nil", nil)
	}

	log.Debugf("processing task definition create for a task %+v", input.TaskDefinition)

	input.TaskDefinition.Tags = ecsTags(input.Tags)

	// path is org/clustername
	path := fmt.Sprintf("%s/%s", o.Org, aws.StringValue(input.Cluster.ClusterName))

	// role name is clustername-ecsTaskExecution
	roleName := fmt.Sprintf("%s-ecsTaskExecution", aws.StringValue(input.Cluster.ClusterName))

	roleARN, err := o.DefaultTaskExecutionRole(ctx, path, roleName, input.Tags)
	if err != nil {
		return nil, rbfunc, err
	}

	input.TaskDefinition.ExecutionRoleArn = aws.String(roleARN)
	input.TaskDefinition.TaskRoleArn = aws.String(roleARN)
	input.TaskDefinition.RequiresCompatibilities = DefaultCompatabilities
	input.TaskDefinition.NetworkMode = DefaultNetworkMode

	logConfiguration, err := o.defaultLogConfiguration(ctx, aws.StringValue(input.Cluster.ClusterName), aws.StringValue(input.TaskDefinition.Family), input.Tags)
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

	return taskDefinition, rbfunc, nil
}

// processTaskDefinitionUpdate processes the task definition portion of the input
func (o *Orchestrator) processTaskDefinitionUpdate(ctx context.Context, input *ServiceOrchestrationUpdateInput, active *ServiceOrchestrationUpdateOutput) error {
	if input == nil || input.TaskDefinition == nil {
		return apierror.New(apierror.ErrBadRequest, "task definition cannot be nil", nil)
	}

	if input.Service == nil {
		return apierror.New(apierror.ErrBadRequest, "service cannot be nil", nil)
	}

	log.Debugf("processing task definition update for a task %+v", input.TaskDefinition)

	// path is org/clustername
	path := fmt.Sprintf("%s/%s", o.Org, input.ClusterName)

	// role name is clustername-ecsTaskExecution
	roleName := fmt.Sprintf("%s-ecsTaskExecution", input.ClusterName)

	roleARN, err := o.DefaultTaskExecutionRole(ctx, path, roleName, input.Tags)
	if err != nil {
		return err
	}

	log.Debugf("setting roleARN: %s", roleARN)
	input.TaskDefinition.ExecutionRoleArn = aws.String(roleARN)
	input.TaskDefinition.TaskRoleArn = aws.String(roleARN)

	if len(input.TaskDefinition.RequiresCompatibilities) == 0 {
		log.Debugf("setting default compatabilities: %+v", DefaultCompatabilities)
		input.TaskDefinition.RequiresCompatibilities = DefaultCompatabilities
	}

	if input.TaskDefinition.NetworkMode == nil {
		log.Debugf("setting default network mode: %s", aws.StringValue(DefaultNetworkMode))
		input.TaskDefinition.NetworkMode = DefaultNetworkMode
	}

	tags := input.Tags
	if tags == nil {
		et := make([]*Tag, len(input.TaskDefinition.Tags))
		for i, t := range input.TaskDefinition.Tags {
			et[i] = &Tag{Key: t.Key, Value: t.Value}
		}
		tags = et
	}

	logConfiguration, err := o.defaultLogConfiguration(ctx, input.ClusterName, aws.StringValue(input.TaskDefinition.Family), tags)
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

func (o *Orchestrator) processTaskDefTaskDefinitionUpdate(ctx context.Context, input *TaskDefUpdateOrchestrationInput, active *TaskDefUpdateOrchestrationOutput) error {
	if input == nil || input.TaskDefinition == nil {
		return apierror.New(apierror.ErrBadRequest, "task definition cannot be nil", nil)
	}

	log.Debugf("processing task definition update for a task %+v", input.TaskDefinition)

	// path is org/clustername
	path := fmt.Sprintf("%s/%s", o.Org, input.ClusterName)

	// role name is clustername-ecsTaskExecution
	roleName := fmt.Sprintf("%s-ecsTaskExecution", input.ClusterName)

	roleARN, err := o.DefaultTaskExecutionRole(ctx, path, roleName, input.Tags)
	if err != nil {
		return err
	}

	input.TaskDefinition.ExecutionRoleArn = aws.String(roleARN)
	input.TaskDefinition.TaskRoleArn = aws.String(roleARN)
	input.TaskDefinition.RequiresCompatibilities = DefaultCompatabilities
	input.TaskDefinition.NetworkMode = DefaultNetworkMode

	tags := input.Tags
	if tags == nil {
		et := make([]*Tag, len(input.TaskDefinition.Tags))
		for i, t := range input.TaskDefinition.Tags {
			et[i] = &Tag{Key: t.Key, Value: t.Value}
		}
		tags = et
	}

	logConfiguration, err := o.defaultLogConfiguration(ctx, input.ClusterName, aws.StringValue(input.TaskDefinition.Family), tags)
	if err != nil {
		return err
	}

	for _, cd := range input.TaskDefinition.ContainerDefinitions {
		cd.SetLogConfiguration(logConfiguration)
	}

	taskDefinition, err := o.ECS.CreateTaskDefinition(ctx, input.TaskDefinition)
	if err != nil {
		return err
	}

	active.TaskDefinition = taskDefinition

	return nil
}

// defaultLogConfiguration generates a log group and sets retention on the log group.  It returns the default log configuration.
func (o *Orchestrator) defaultLogConfiguration(ctx context.Context, logGroup, streamPrefix string, tags []*Tag) (*ecs.LogConfiguration, error) {
	if logGroup == "" {
		return nil, errors.New("cloudwatch logs group name cannot be empty")
	}

	var tagsMap = make(map[string]*string)
	for _, tag := range tags {
		tagsMap[aws.StringValue(tag.Key)] = tag.Value
	}

	if err := o.CloudWatchLogs.CreateLogGroup(ctx, &cloudwatchlogs.CreateLogGroupInput{
		LogGroupName: aws.String(logGroup),
		Tags:         tagsMap,
	}); err != nil {
		if aerr, ok := err.(apierror.Error); ok {
			switch aerr.Code {
			case apierror.ErrConflict:
				log.Warnf("cloudwatch log group already exists, continuing: (%s)", err)
			default:
				return nil, err
			}
		} else {
			return nil, err
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
