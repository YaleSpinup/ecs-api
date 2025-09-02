package orchestration

import (
	"context"
	"strconv"
	"strings"

	"github.com/YaleSpinup/apierror"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/arn"
	"github.com/aws/aws-sdk-go/service/ecs"
	log "github.com/sirupsen/logrus"
)

type Task struct {
	*ecs.Task
	Revision int64
}

type TaskOutput struct {
	Tasks    []*Task
	Failures []*ecs.Failure
}

func (o *Orchestrator) GetTask(ctx context.Context, cluster, task string) (*TaskOutput, error) {
	if task == "" {
		return nil, apierror.New(apierror.ErrBadRequest, "task cannot be empty", nil)
	}

	out, err := o.ECS.GetTasks(ctx, &ecs.DescribeTasksInput{
		Cluster: aws.String(cluster),
		Tasks:   aws.StringSlice([]string{task}),
	})
	if err != nil {
		return nil, err
	}

	output, err := toTaskOutput(out.Tasks, out.Failures)
	if err != nil {
		return nil, err
	}

	return output, nil
}

// toTaskOutput adds the revision to the tasks
func toTaskOutput(tasks []*ecs.Task, failures []*ecs.Failure) (*TaskOutput, error) {
	output := &TaskOutput{Failures: failures}
	ts := make([]*Task, 0, len(tasks))
	for _, t := range tasks {
		revision := int64(0)
		tdArn, err := arn.Parse(aws.StringValue(t.TaskDefinitionArn))
		if err != nil {
			log.Debugf("failed to parse taskdefinition ARN: '%s': %s", aws.StringValue(t.TaskDefinitionArn), err)
		} else {
			log.Debugf("splitting task def arn resource %s", tdArn.Resource)
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

		ts = append(ts, &Task{
			Task:     t,
			Revision: revision,
		})
	}
	output.Tasks = ts

	log.Debugf("returning output from running tasks: %+v", output)

	return output, nil
}

func (o *Orchestrator) StopTask(ctx context.Context, cluster, task, reason string) error {
	if cluster == "" || task == "" {
		return apierror.New(apierror.ErrBadRequest, "cluster and task are required", nil)
	}

	input := ecs.StopTaskInput{
		Cluster: aws.String(cluster),
		Task:    aws.String(task),
	}

	if reason != "" {
		input.SetReason(reason)
	}

	if _, err := o.ECS.StopTask(ctx, &input); err != nil {
		return err
	}

	return nil
}
