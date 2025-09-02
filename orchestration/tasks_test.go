package orchestration

import (
	"reflect"
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ecs"
)

func Test_toTaskOutput(t *testing.T) {
	type args struct {
		tasks    []*ecs.Task
		failures []*ecs.Failure
	}
	tests := []struct {
		name    string
		args    args
		want    *TaskOutput
		wantErr bool
	}{
		{
			name: "nil input",
			args: args{},
			want: &TaskOutput{
				Tasks: []*Task{},
			},
		},
		{
			name: "tasks input",
			args: args{
				tasks: []*ecs.Task{
					{TaskDefinitionArn: aws.String("arn:aws:ecs:us-east-1:1234567890:task-definition/td1:1")},
					{TaskDefinitionArn: aws.String("arn:aws:ecs:us-east-1:1234567890:task-definition/td2:2")},
					{TaskDefinitionArn: aws.String("arn:aws:ecs:us-east-1:1234567890:task-definition/td3:3")},
				},
			},
			want: &TaskOutput{
				Tasks: []*Task{
					{&ecs.Task{TaskDefinitionArn: aws.String("arn:aws:ecs:us-east-1:1234567890:task-definition/td1:1")}, 1},
					{&ecs.Task{TaskDefinitionArn: aws.String("arn:aws:ecs:us-east-1:1234567890:task-definition/td2:2")}, 2},
					{&ecs.Task{TaskDefinitionArn: aws.String("arn:aws:ecs:us-east-1:1234567890:task-definition/td3:3")}, 3},
				},
			},
		},
		{
			name: "bad arn input",
			args: args{
				tasks: []*ecs.Task{
					{TaskDefinitionArn: aws.String("i-am-not-an-arn")},
				},
			},
			want: &TaskOutput{
				Tasks: []*Task{
					{&ecs.Task{TaskDefinitionArn: aws.String("i-am-not-an-arn")}, 0},
				},
			},
		},
		{
			name: "multiple bad arn inputs should not fail",
			args: args{
				tasks: []*ecs.Task{
					{TaskDefinitionArn: aws.String("invalid-arn-1")},
					{TaskDefinitionArn: aws.String("not-an-arn-either")},
					{TaskDefinitionArn: aws.String("arn:aws:ecs:us-east-1:1234567890:task-definition/valid:1")},
				},
			},
			want: &TaskOutput{
				Tasks: []*Task{
					{&ecs.Task{TaskDefinitionArn: aws.String("invalid-arn-1")}, 0},
					{&ecs.Task{TaskDefinitionArn: aws.String("not-an-arn-either")}, 0},
					{&ecs.Task{TaskDefinitionArn: aws.String("arn:aws:ecs:us-east-1:1234567890:task-definition/valid:1")}, 1},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := toTaskOutput(tt.args.tasks, tt.args.failures)
			if (err != nil) != tt.wantErr {
				t.Errorf("toTaskOutput() error = %+v, wantErr %+v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("toTaskOutput() = %+v, want %+v", got, tt.want)
			}
		})
	}
}
