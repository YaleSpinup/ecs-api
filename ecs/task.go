package ecs

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/YaleSpinup/apierror"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/arn"
	"github.com/aws/aws-sdk-go/service/ecs"
	log "github.com/sirupsen/logrus"
)

// ListTasks collects all of the task ids for a service in a cluster with the given status(s)ß
func (e *ECS) ListTasks(ctx context.Context, cluster, service string, status []string) ([]*string, error) {
	if cluster == "" || service == "" {
		return nil, apierror.New(apierror.ErrBadRequest, "invalid input", nil)
	}

	// default to "RUNNING" status
	if status == nil {
		status = []string{"RUNNING"}
	}

	log.Infof("listing tasks in %s/%s with status %s", cluster, service, strings.Join(status, ","))

	tasks := []*string{}
	for _, s := range status {
		output, err := e.Service.ListTasksWithContext(ctx, &ecs.ListTasksInput{
			Cluster:       aws.String(cluster),
			ServiceName:   aws.String(service),
			LaunchType:    aws.String("FARGATE"),
			DesiredStatus: aws.String(s),
		})

		if err != nil {
			msg := fmt.Sprintf("failed listing tasks for cluster %s, service %s with status %s", cluster, service, s)
			return tasks, ErrCode(msg, err)
		}

		for _, t := range output.TaskArns {
			taskArn, err := arn.Parse(aws.StringValue(t))
			if err != nil {
				msg := fmt.Sprintf("failed to parse '%s'", aws.StringValue(t))
				return tasks, ErrCode(msg, err)
			}

			// task resource is the form task/xxxxxxxxxxxxx
			r := strings.SplitN(taskArn.Resource, "/", 2)
			tasks = append(tasks, aws.String(r[1]))
		}

	}

	return tasks, nil
}

type Task struct {
	*ecs.Task
	Revision int64
}

type DescribeTasksOutput struct {
	Failures []*ecs.Failure
	Tasks    []*Task
}

// GetTasks describes the given tasks in the give cluster
func (e *ECS) GetTasks(ctx context.Context, input *ecs.DescribeTasksInput) (*DescribeTasksOutput, error) {
	if input == nil {
		return nil, apierror.New(apierror.ErrBadRequest, "invalid input", nil)
	}

	log.Infof("getting cluster %s tasks  %s", aws.StringValue(input.Cluster), strings.Join(aws.StringValueSlice(input.Tasks), ","))

	ecsOutput, err := e.Service.DescribeTasksWithContext(ctx, input)
	if err != nil {
		return nil, ErrCode("failed to describe tasks", err)
	}

	output := &DescribeTasksOutput{Failures: ecsOutput.Failures}
	tasks := make([]*Task, 0, len(ecsOutput.Tasks))
	for _, t := range ecsOutput.Tasks {
		revision := int64(0)
		tdArn, err := arn.Parse(aws.StringValue(t.TaskDefinitionArn))
		if err != nil {
			log.Errorf("failed to parse taskdefinition ARN: '%s': %s", aws.StringValue(t.TaskDefinitionArn), err)
		} else {
			ss := strings.Split(tdArn.Resource, ":")
			if len(ss) > 1 {
				s := ss[len(ss)-1]
				si, err := strconv.Atoi(s)
				if err != nil {
					log.Errorf("failed to parse revision '%s' as number for arn resource '%s': %s", s, tdArn.Resource, err)
				}
				revision = int64(si)
			}
		}

		tasks = append(tasks, &Task{
			Task:     t,
			Revision: revision,
		})
	}
	output.Tasks = tasks

	return output, nil
}
