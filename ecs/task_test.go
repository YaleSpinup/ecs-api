package ecs

import (
	"context"
	"reflect"
	"testing"

	"github.com/YaleSpinup/ecs-api/apierror"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/request"
	"github.com/aws/aws-sdk-go/service/ecs"

	"github.com/pkg/errors"
)

type taskListTest struct {
	// input
	cluster, service string
	status           []string
	// output
	expected []*string
	err      error
	// mock data
	tasks  map[string]string
	awsErr awserr.Error
}

var taskListTests = []*taskListTest{
	&taskListTest{
		cluster: "clu1",
		service: "svc1",
		status:  []string{"RUNNING"},
		expected: []*string{
			aws.String("task2:2"),
		},
		tasks: map[string]string{
			"arn:aws:ecs:us-east-1:1234567890:task/task1:1": "STOPPED",
			"arn:aws:ecs:us-east-1:1234567890:task/task2:2": "RUNNING",
			"arn:aws:ecs:us-east-1:1234567890:task/task3:3": "STOPPING",
			"arn:aws:ecs:us-east-1:1234567890:task/task4:4": "PENDING",
			"arn:aws:ecs:us-east-1:1234567890:task/task5:5": "FAILED",
		},
	},
	&taskListTest{
		cluster: "clu1",
		service: "svc2",
		status:  []string{"STOPPED"},
		expected: []*string{
			aws.String("task1:1"),
		},
		tasks: map[string]string{
			"arn:aws:ecs:us-east-1:1234567890:task/task1:1": "STOPPED",
			"arn:aws:ecs:us-east-1:1234567890:task/task2:2": "RUNNING",
			"arn:aws:ecs:us-east-1:1234567890:task/task3:3": "STOPPING",
			"arn:aws:ecs:us-east-1:1234567890:task/task4:4": "PENDING",
			"arn:aws:ecs:us-east-1:1234567890:task/task5:5": "FAILED",
		},
	},
	&taskListTest{
		cluster: "clu2",
		service: "svc1",
		status:  []string{"STOPPING", "PENDING", "FAILED"},
		expected: []*string{
			aws.String("task3:3"),
			aws.String("task4:4"),
			aws.String("task5:5"),
		},
		tasks: map[string]string{
			"arn:aws:ecs:us-east-1:1234567890:task/task1:1": "STOPPED",
			"arn:aws:ecs:us-east-1:1234567890:task/task2:2": "RUNNING",
			"arn:aws:ecs:us-east-1:1234567890:task/task3:3": "STOPPING",
			"arn:aws:ecs:us-east-1:1234567890:task/task4:4": "PENDING",
			"arn:aws:ecs:us-east-1:1234567890:task/task5:5": "FAILED",
		},
	},
	&taskListTest{
		cluster:  "clu2",
		service:  "svc2",
		status:   []string{"STOPPING", "PENDING", "FAILED"},
		expected: []*string{},
		tasks: map[string]string{
			"arn:aws:ecs:us-east-1:1234567890:task/task1:1": "RUNNING",
			"arn:aws:ecs:us-east-1:1234567890:task/task2:2": "RUNNING",
			"arn:aws:ecs:us-east-1:1234567890:task/task3:3": "RUNNING",
		},
	},
	&taskListTest{
		cluster: "clu2",
		service: "svc2",
		status:  []string{"RUNNING"},
		err:     apierror.New("400", "Bad Request", errors.New("Bad Request")),
		awsErr:  awserr.New("400", "Bad Request", errors.New("Bad Request")),
	},
}

func (m *mockECSClient) ListTasksWithContext(ctx aws.Context, input *ecs.ListTasksInput, opts ...request.Option) (*ecs.ListTasksOutput, error) {
	if m.err != nil {
		m.t.Logf("returning error, %s", m.err)
		return nil, m.err
	}

	cluster := aws.StringValue(input.Cluster)
	service := aws.StringValue(input.ServiceName)
	status := aws.StringValue(input.DesiredStatus)

	output := []*string{}
	for _, taskTest := range taskListTests {
		if cluster == taskTest.cluster && service == taskTest.service {
			for taskArn, taskStatus := range taskTest.tasks {
				if status == taskStatus {
					output = append(output, aws.String(taskArn))
				}
			}
		}
	}

	return &ecs.ListTasksOutput{TaskArns: output}, nil
}

func TestListTasks(t *testing.T) {
	for _, test := range taskListTests {
		client := ECS{Service: &mockECSClient{t: t, err: test.awsErr}}
		output, err := client.ListTasks(context.TODO(), test.cluster, test.service, test.status)
		if test.err == nil && err == nil {
			if !reflect.DeepEqual(output, test.expected) {
				t.Errorf("expected output %+v, got %+v", aws.StringValueSlice(test.expected), aws.StringValueSlice(output))
			}
		} else if test.err != nil && err == nil {
			t.Errorf("expected error %s, got nil", test.err)
		} else if err != nil && test.err == nil {
			t.Errorf("expected nil error, got %s", err)
		} else {
			if aerr, ok := errors.Cause(err).(apierror.Error); ok {
				t.Logf("got apierror '%s'", aerr)
			} else {
				t.Errorf("expected error to be an apierror.Error, got %s", err)
			}
		}
	}
}
