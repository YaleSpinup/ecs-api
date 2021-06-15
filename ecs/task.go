package ecs

import (
	"context"
	"fmt"
	"strings"

	"github.com/YaleSpinup/apierror"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/arn"
	"github.com/aws/aws-sdk-go/service/ecs"
	log "github.com/sirupsen/logrus"
)

// ListTasks lists the tasks with standard ECS input
func (e *ECS) ListTasks(ctx context.Context, input *ecs.ListTasksInput) ([]*string, error) {
	if input == nil {
		return nil, apierror.New(apierror.ErrBadRequest, "invalid input", nil)
	}

	log.Info("listing tasks")

	output, err := e.Service.ListTasksWithContext(ctx, input)
	if err != nil {
		return nil, ErrCode("failed listing tasks", err)
	}

	tasks := []*string{}
	for _, t := range output.TaskArns {
		a := aws.StringValue(t)

		log.Debugf("parsing arn %s", a)

		taskArn, err := arn.Parse(a)
		if err != nil {
			msg := fmt.Sprintf("failed to parse '%s'", a)
			return tasks, ErrCode(msg, err)
		}

		// task resource is the form task/xxxxxxxxxxxxx
		r := strings.SplitN(taskArn.Resource, "/", 2)
		tasks = append(tasks, aws.String(r[1]))
	}

	return tasks, nil
}

// GetTasks describes the given tasks in the give cluster
func (e *ECS) GetTasks(ctx context.Context, input *ecs.DescribeTasksInput) (*ecs.DescribeTasksOutput, error) {
	if input == nil {
		return nil, apierror.New(apierror.ErrBadRequest, "invalid input", nil)
	}

	log.Infof("getting cluster %s tasks  %s", aws.StringValue(input.Cluster), strings.Join(aws.StringValueSlice(input.Tasks), ","))

	out, err := e.Service.DescribeTasksWithContext(ctx, input)
	if err != nil {
		return nil, ErrCode("failed to describe tasks", err)
	}

	log.Debugf("output from describing task %s/%+v: %+v", aws.StringValue(input.Cluster), aws.StringValueSlice(input.Tasks), out)

	return out, nil
}

func (e *ECS) RunTask(ctx context.Context, input *ecs.RunTaskInput) (*ecs.RunTaskOutput, error) {
	if input == nil {
		return nil, apierror.New(apierror.ErrBadRequest, "invalid input", nil)
	}

	log.Infof("running %d task(s) from task definition %s/%s", aws.Int64Value(input.Count), aws.StringValue(input.Cluster), aws.StringValue(input.TaskDefinition))

	out, err := e.Service.RunTaskWithContext(ctx, input)
	if err != nil {
		return nil, ErrCode("failed to run tasks", err)
	}

	log.Debugf("output from running taskdef %s/%s: %+v", aws.StringValue(input.Cluster), aws.StringValue(input.TaskDefinition), out)

	return out, nil
}
