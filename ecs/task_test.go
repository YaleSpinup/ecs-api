package ecs

import (
	"context"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"testing"

	"github.com/YaleSpinup/apierror"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/awsutil"
	"github.com/aws/aws-sdk-go/aws/request"
	"github.com/aws/aws-sdk-go/service/ecs"
	"github.com/aws/aws-sdk-go/service/ecs/ecsiface"
	"github.com/pkg/errors"
)

type taskListTest struct {
	// input
	cluster, service, family string
	status                   []string
	// output
	expected []*string
	err      error
	// mock data
	tasks  map[string]string
	awsErr awserr.Error
}

var taskListTests = []*taskListTest{
	// test empty cluster and service
	{
		err: apierror.New(apierror.ErrBadRequest, "invalid input", nil),
	},
	// test empty service
	{
		cluster: "clu0",
		err:     apierror.New(apierror.ErrBadRequest, "invalid input", nil),
	},
	// test empty cluster
	{
		service: "svc0",
		err:     apierror.New(apierror.ErrBadRequest, "invalid input", nil),
	},
	// test empty status list
	{
		cluster: "clu0",
		service: "svc0",
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
	// test single RUNNING status
	{
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
	// test single STOPPED status
	{
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
	// test multiple matching status'
	{
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
	// test no matching statuses
	{
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
	// test Bad Request from AWS
	{
		cluster: "clu2",
		service: "svc2",
		status:  []string{"RUNNING"},
		err:     apierror.New("400", "Bad Request", errors.New("Bad Request")),
		awsErr:  awserr.New("400", "Bad Request", errors.New("Bad Request")),
	},
}

var testTasks = []*ecs.Task{
	{
		ClusterArn:        aws.String("arn:aws:ecs:us-east-1:1234567890:cluster/clu0"),
		Cpu:               aws.String("2048"),
		Memory:            aws.String("4096"),
		TaskArn:           aws.String("arn:aws:ecs:us-east-1:1234567890:task/task1"),
		TaskDefinitionArn: aws.String("arn:aws:ecs:us-east-1:1234567890:task-definition/task1:10"),
	},
	{
		ClusterArn:        aws.String("arn:aws:ecs:us-east-1:1234567890:cluster/clu0"),
		Cpu:               aws.String("1024"),
		Memory:            aws.String("4096"),
		TaskArn:           aws.String("arn:aws:ecs:us-east-1:1234567890:task/task2"),
		TaskDefinitionArn: aws.String("arn:aws:ecs:us-east-1:1234567890:task-definition/task2:10"),
	},
}

func (m *mockECSClient) ListTasksWithContext(ctx aws.Context, input *ecs.ListTasksInput, opts ...request.Option) (*ecs.ListTasksOutput, error) {
	if m.err != nil {
		return nil, m.err
	}

	cluster := aws.StringValue(input.Cluster)
	service := aws.StringValue(input.ServiceName)
	status := aws.StringValue(input.DesiredStatus)
	family := aws.StringValue(input.Family)

	output := []*string{}
	for _, taskTest := range taskListTests {
		if cluster != "" && cluster != taskTest.cluster {
			m.t.Logf("input cluster %s doesn't match %s", cluster, taskTest.cluster)
			continue
		}
		m.t.Logf("input cluster %s matches %s", cluster, taskTest.cluster)

		if service != "" && service != taskTest.service {
			m.t.Logf("input service %s doesn't match %s", service, taskTest.service)
			continue
		}
		m.t.Logf("input service %s matches %s", service, taskTest.service)

		if family != "" && family != taskTest.family {
			m.t.Logf("input family %s doesn't match %s", family, taskTest.family)
			continue
		}
		m.t.Logf("input family %s matches %s", family, taskTest.family)

		for taskArn, taskStatus := range taskTest.tasks {
			if status != "" && status != taskStatus {
				continue
			}

			m.t.Logf("appending task arn %s", taskArn)

			output = append(output, aws.String(taskArn))
		}
	}

	// map order isn't guarenteed so sort the resulting slice to be sure out tests are consistent
	sort.Slice(output, func(i, j int) bool {
		a1s := strings.Split(aws.StringValue(output[i]), ":")
		a1v, _ := strconv.Atoi(a1s[len(a1s)-1])

		a2s := strings.Split(aws.StringValue(output[j]), ":")
		a2v, _ := strconv.Atoi(a2s[len(a2s)-1])

		return a1v < a2v
	})

	m.t.Logf("returning task arns %+v", output)

	return &ecs.ListTasksOutput{TaskArns: output}, nil
}

func (m *mockECSClient) DescribeTasksWithContext(ctx aws.Context, input *ecs.DescribeTasksInput, opts ...request.Option) (*ecs.DescribeTasksOutput, error) {
	if m.err != nil {
		return nil, m.err
	}

	return &ecs.DescribeTasksOutput{
		Tasks: testTasks,
	}, nil
}

func (m *mockECSClient) RunTaskWithContext(ctx context.Context, input *ecs.RunTaskInput, opts ...request.Option) (*ecs.RunTaskOutput, error) {
	if m.err != nil {
		return nil, m.err
	}

	return &ecs.RunTaskOutput{
		Tasks: testTasks,
	}, nil
}

func TestGetTasks(t *testing.T) {
	client := ECS{Service: &mockECSClient{t: t}}

	if _, err := client.GetTasks(context.TODO(), nil); err == nil {
		t.Error("expected error for nil input, got nil")
	}

	out, err := client.GetTasks(context.TODO(), &ecs.DescribeTasksInput{
		Cluster: aws.String("clu0"),
		Tasks: []*string{
			aws.String("task1"),
			aws.String("task2"),
		},
	})

	if err != nil {
		t.Errorf("expected nil error, got %s", err)
	}

	if !awsutil.DeepEqual(out.Tasks, testTasks) {
		t.Errorf("expected %s, got %s", awsutil.Prettify(testTasks), awsutil.Prettify(out.Tasks))
	}
}

func TestECS_RunTask(t *testing.T) {
	type fields struct {
		Service        ecsiface.ECSAPI
		DefaultSgs     []string
		DefaultSubnets []string
	}
	type args struct {
		ctx   context.Context
		input *ecs.RunTaskInput
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    *ecs.RunTaskOutput
		wantErr bool
	}{
		{
			name: "nil input",
			fields: fields{
				Service: &mockECSClient{t: t},
			},
			args: args{
				ctx: context.TODO(),
			},
			wantErr: true,
		},
		{
			name: "error from aws",
			fields: fields{
				Service: &mockECSClient{
					t:   t,
					err: awserr.New(ecs.ErrCodePlatformUnknownException, "bad platform", nil),
				},
			},
			args: args{
				ctx:   context.TODO(),
				input: &ecs.RunTaskInput{},
			},
			wantErr: true,
		},
		{
			name: "good input",
			fields: fields{
				Service: &mockECSClient{t: t},
			},
			args: args{
				ctx:   context.TODO(),
				input: &ecs.RunTaskInput{},
			},
			want: &ecs.RunTaskOutput{
				Tasks: testTasks,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := &ECS{
				Service:        tt.fields.Service,
				DefaultSgs:     tt.fields.DefaultSgs,
				DefaultSubnets: tt.fields.DefaultSubnets,
			}
			got, err := e.RunTask(tt.args.ctx, tt.args.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ECS.RunTask() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ECS.RunTask() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestECS_ListTasks(t *testing.T) {
	type fields struct {
		Service        ecsiface.ECSAPI
		DefaultSgs     []string
		DefaultSubnets []string
	}
	type args struct {
		ctx   context.Context
		input *ecs.ListTasksInput
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    []*string
		wantErr bool
	}{
		{
			name: "nil input",
			fields: fields{
				Service: &mockECSClient{t: t},
			},
			args: args{
				ctx: context.TODO(),
			},
			wantErr: true,
		},
		{
			name: "error from aws",
			fields: fields{
				Service: &mockECSClient{
					t:   t,
					err: awserr.New(ecs.ErrCodePlatformUnknownException, "bad platform", nil),
				},
			},
			args: args{
				ctx:   context.TODO(),
				input: &ecs.ListTasksInput{},
			},
			wantErr: true,
		},
		{
			name: "match service",
			fields: fields{
				Service: &mockECSClient{t: t},
			},
			args: args{
				ctx: context.TODO(),
				input: &ecs.ListTasksInput{
					Cluster:     aws.String("clu0"),
					ServiceName: aws.String("svc0"),
				},
			},
			want: []*string{
				aws.String("task1:1"),
				aws.String("task2:2"),
				aws.String("task3:3"),
				aws.String("task4:4"),
				aws.String("task5:5"),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := &ECS{
				Service:        tt.fields.Service,
				DefaultSgs:     tt.fields.DefaultSgs,
				DefaultSubnets: tt.fields.DefaultSubnets,
			}
			got, err := e.ListTasks(tt.args.ctx, tt.args.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ECS.ListTasks() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ECS.ListTasks() = %v, want %v", awsutil.Prettify(got), awsutil.Prettify(tt.want))
			}
		})
	}
}
