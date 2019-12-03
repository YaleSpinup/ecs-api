package orchestration

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/request"
	"github.com/aws/aws-sdk-go/service/ecs"
)

func (m *mockECSClient) RegisterTaskDefinitionWithContext(ctx aws.Context, input *ecs.RegisterTaskDefinitionInput, opts ...request.Option) (*ecs.RegisterTaskDefinitionOutput, error) {
	if aws.StringValue(input.Family) == "goodtd" {
		goodTd.Compatibilities = input.RequiresCompatibilities
		goodTd.NetworkMode = input.NetworkMode
		return &ecs.RegisterTaskDefinitionOutput{
			TaskDefinition: goodTd,
		}, nil
	}
	return nil, errors.New("Failed to create mock task definition")
}

func (m *mockECSClient) DescribeTaskDefinitionWithContext(ctx aws.Context, input *ecs.DescribeTaskDefinitionInput, opts ...request.Option) (*ecs.DescribeTaskDefinitionOutput, error) {
	if aws.StringValue(input.TaskDefinition) == "goodtd" {
		return &ecs.DescribeTaskDefinitionOutput{
			TaskDefinition: goodTd,
		}, nil
	}
	msg := fmt.Sprintf("Failed to get mock task definition %s", aws.StringValue(input.TaskDefinition))
	return nil, errors.New(msg)
}

func TestCreateTaskDefinition(t *testing.T) {
	client := &mockECSClient{}

	// test a boring task definition
	td, err := createTaskDefinition(context.TODO(), client, &ecs.RegisterTaskDefinitionInput{Family: aws.String("goodtd")})
	if err != nil {
		t.Fatal("expected no error from create task definition, got:", err)
	}
	t.Log("got task definition response for good task definition", td)
	if !reflect.DeepEqual(goodTd, td) {
		t.Fatalf("Expected %+v\nGot %+v", goodTd, td)
	}

	// test an error task definition
	td, err = createTaskDefinition(context.TODO(), client, &ecs.RegisterTaskDefinitionInput{})
	if err == nil {
		t.Fatal("expected error from create task definition, got", err, td)
	}
	t.Log("got expected error response for bad task definition", err)

	// test a task definition with custom compatabilities
	td, err = createTaskDefinition(context.TODO(), client, &ecs.RegisterTaskDefinitionInput{
		Family:                  aws.String("goodtd"),
		RequiresCompatibilities: aws.StringSlice([]string{"FOOBAR"}),
	})
	if err != nil {
		t.Fatal("expected no error from create task definition with custom compatabilities, got:", err)
	}
	if !reflect.DeepEqual([]string{"FOOBAR"}, aws.StringValueSlice(td.Compatibilities)) {
		t.Fatal("Expected compatabilitieis to be custom:", []string{"FOOBAR"}, "got:", aws.StringValueSlice(td.Compatibilities))
	}
	t.Log("got task definition response for good task definition with custom compatablilities", td)
}

func TestDescribeTaskDefinition(t *testing.T) {
	client := &mockECSClient{}
	td, err := getTaskDefinition(context.TODO(), client, aws.String("goodtd"))
	if err != nil {
		t.Fatal("expected no error from describe task definition, got:", err)
	}
	t.Log("got task definition response for good task definition", td)
	if !reflect.DeepEqual(goodTd, td) {
		t.Fatalf("Expected %+v\nGot %+v", goodTd, td)
	}

	// test an error task definition
	td, err = getTaskDefinition(context.TODO(), client, aws.String("badtd"))
	if err == nil {
		t.Fatal("expected error from create task definition, got", err, td)
	}
	t.Log("got expected error response for bad task definition", err)
}
