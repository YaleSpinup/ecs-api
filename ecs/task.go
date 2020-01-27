package ecs

import (
	"context"
	"fmt"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/arn"
	"github.com/aws/aws-sdk-go/service/ecs"
)

// ListTasks collects all of the task ids for a service in a cluster with the given status(s)ß
func (e *ECS) ListTasks(ctx context.Context, cluster, service string, status []string) ([]*string, error) {
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
