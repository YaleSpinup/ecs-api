package ecs

import (
	"context"
	"reflect"
	"testing"

	"github.com/YaleSpinup/apierror"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/request"
	"github.com/aws/aws-sdk-go/service/ecs"

	"github.com/pkg/errors"
)

var (
	goodContainerDefs = []*ecs.ContainerDefinition{
		{
			Name:  aws.String("webserver"),
			Image: aws.String("nginx:alpine"),
		},
		{
			Name:  aws.String("testDef1"),
			Image: aws.String("secretImage1"),
		},
		{
			Name:  aws.String("testDef2"),
			Image: aws.String("secretImage2"),
		},
	}

	goodTd1 = &ecs.TaskDefinition{
		Compatibilities:      aws.StringSlice([]string{"EC2", "FARGATE"}),
		ContainerDefinitions: goodContainerDefs,
		Cpu:                  aws.String("256"),
		Family:               aws.String("goodtd"),
		Memory:               aws.String("512"),
		Revision:             aws.Int64(666),
		Status:               aws.String("ACTIVE"),
		TaskDefinitionArn:    aws.String("arn:aws:ecs:us-east-1:1234567890:task-definition/goodtd:666"),
	}

	goodTd2 = &ecs.TaskDefinition{
		Compatibilities:      aws.StringSlice([]string{"EC2", "FARGATE"}),
		ContainerDefinitions: goodContainerDefs,
		Cpu:                  aws.String("256"),
		Family:               aws.String("goodtd"),
		Memory:               aws.String("512"),
		Revision:             aws.Int64(667),
		Status:               aws.String("ACTIVE"),
		TaskDefinitionArn:    aws.String("arn:aws:ecs:us-east-1:1234567890:task-definition/goodtd:667"),
	}

	goodTd3 = &ecs.TaskDefinition{
		Compatibilities:      aws.StringSlice([]string{"EC2", "FARGATE"}),
		ContainerDefinitions: goodContainerDefs,
		Cpu:                  aws.String("256"),
		Family:               aws.String("goodtd"),
		Memory:               aws.String("512"),
		Revision:             aws.Int64(668),
		Status:               aws.String("ACTIVE"),
		TaskDefinitionArn:    aws.String("arn:aws:ecs:us-east-1:1234567890:task-definition/goodtd:668"),
	}

	testTaskDefs = []*ecs.TaskDefinition{goodTd1, goodTd2, goodTd3}
)

func (m *mockECSClient) RegisterTaskDefinitionWithContext(ctx aws.Context, input *ecs.RegisterTaskDefinitionInput, opts ...request.Option) (*ecs.RegisterTaskDefinitionOutput, error) {
	if aws.StringValue(input.Family) == "goodtd" {
		goodTd1.Compatibilities = input.RequiresCompatibilities
		goodTd1.NetworkMode = input.NetworkMode
		return &ecs.RegisterTaskDefinitionOutput{
			TaskDefinition: goodTd1,
		}, nil
	}
	return nil, errors.New("Failed to create mock task definition")
}

func (m *mockECSClient) DescribeTaskDefinitionWithContext(ctx aws.Context, input *ecs.DescribeTaskDefinitionInput, opts ...request.Option) (*ecs.DescribeTaskDefinitionOutput, error) {
	if m.err != nil {
		return nil, m.err
	}

	for _, td := range testTaskDefs {
		if aws.StringValue(td.TaskDefinitionArn) == aws.StringValue(input.TaskDefinition) {
			return &ecs.DescribeTaskDefinitionOutput{
				TaskDefinition: td,
			}, nil
		}
	}

	return nil, awserr.New("404", ecs.ErrCodeResourceNotFoundException, errors.New("Task definition not found: "+aws.StringValue(input.TaskDefinition)))
}

func (m *mockECSClient) DeregisterTaskDefinitionWithContext(ctx aws.Context, input *ecs.DeregisterTaskDefinitionInput, opts ...request.Option) (*ecs.DeregisterTaskDefinitionOutput, error) {
	if m.err != nil {
		return nil, m.err
	}

	for _, td := range testTaskDefs {
		if aws.StringValue(td.TaskDefinitionArn) == aws.StringValue(input.TaskDefinition) {
			return &ecs.DeregisterTaskDefinitionOutput{
				TaskDefinition: td,
			}, nil
		}
	}

	return nil, awserr.New("404", ecs.ErrCodeResourceNotFoundException, errors.New("Task definition not found: "+aws.StringValue(input.TaskDefinition)))
}

func (m *mockECSClient) ListTaskDefinitionsWithContext(ctx aws.Context, input *ecs.ListTaskDefinitionsInput, opts ...request.Option) (*ecs.ListTaskDefinitionsOutput, error) {
	if m.err != nil {
		return nil, m.err
	}

	return &ecs.ListTaskDefinitionsOutput{
		TaskDefinitionArns: []*string{
			goodTd1.TaskDefinitionArn,
			goodTd2.TaskDefinitionArn,
			goodTd3.TaskDefinitionArn,
		},
	}, nil
}

func TestCreateTaskDefinition(t *testing.T) {
	client := ECS{Service: &mockECSClient{t: t}}

	// test a boring task definition
	td, err := client.CreateTaskDefinition(context.TODO(), &ecs.RegisterTaskDefinitionInput{Family: aws.String("goodtd")})
	if err != nil {
		t.Fatal("expected no error from create task definition, got:", err)
	}
	t.Log("got task definition response for good task definition", td)
	if !reflect.DeepEqual(goodTd1, td) {
		t.Fatalf("Expected %+v\nGot %+v", goodTd1, td)
	}

	// test a nil task definition
	td, err = client.CreateTaskDefinition(context.TODO(), nil)
	if err == nil {
		t.Fatal("expected error from create task definition, got", err, td)
	}

	// test an error task definition
	td, err = client.CreateTaskDefinition(context.TODO(), &ecs.RegisterTaskDefinitionInput{})
	if err == nil {
		t.Fatal("expected error from create task definition, got", err, td)
	}
	t.Log("got expected error response for bad task definition", err)

	// test a task definition with custom compatabilities
	td, err = client.CreateTaskDefinition(context.TODO(), &ecs.RegisterTaskDefinitionInput{
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

func TestGetTaskDefinition(t *testing.T) {
	client := ECS{Service: &mockECSClient{t: t}}
	td, _, err := client.GetTaskDefinition(context.TODO(), goodTd1.TaskDefinitionArn, false)
	if err != nil {
		t.Fatal("expected no error from describe task definition, got:", err)
	}
	t.Log("got task definition response for good task definition", td)
	if !reflect.DeepEqual(goodTd1, td) {
		t.Fatalf("Expected %+v\nGot %+v", goodTd1, td)
	}

	// test nil task definition
	_, _, err = client.GetTaskDefinition(context.TODO(), nil, false)
	if err == nil {
		t.Fatal("expected error from create task definition, got nil")
	}

	// test empty task definition
	_, _, err = client.GetTaskDefinition(context.TODO(), aws.String(""), false)
	if err == nil {
		t.Fatal("expected error from create task definition, got nil")
	}

	_, _, err = client.GetTaskDefinition(context.TODO(), aws.String("somenotfoundtd"), false)
	if err == nil {
		t.Fatal("expected error from GetTaskDefinition, got:", err)
	}

	// test an error task definition
	td, _, err = client.GetTaskDefinition(context.TODO(), aws.String("badtd"), false)
	if err == nil {
		t.Fatal("expected error from create task definition, got", err, td)
	}
	t.Log("got expected error response for bad task definition", err)
}

func TestListTaskDefinitionRevisions(t *testing.T) {
	expected := aws.StringValueSlice([]*string{
		goodTd1.TaskDefinitionArn,
		goodTd2.TaskDefinitionArn,
		goodTd3.TaskDefinitionArn,
	})

	client := ECS{Service: &mockECSClient{t: t}}

	_, err := client.ListTaskDefinitionRevisions(context.TODO(), aws.String(""))
	if err == nil {
		t.Fatal("expected error from ListTaskDefinitionRevisions with empty family, got nil")
	}

	_, err = client.ListTaskDefinitionRevisions(context.TODO(), nil)
	if err == nil {
		t.Fatal("expected error from ListTaskDefinitionRevisions with nil family, got nil")
	}

	tds, err := client.ListTaskDefinitionRevisions(context.TODO(), aws.String("goodtd"))
	if err != nil {
		t.Fatal("expected no error from ListTaskDefinitionRevisions, got:", err)
	}

	t.Log("got task definition list", tds)

	if !reflect.DeepEqual(expected, tds) {
		t.Fatalf("Expected %+v\nGot %+v", expected, tds)
	}

	client = ECS{Service: &mockECSClient{t: t, err: awserr.New("400", "Bad Request", errors.New("Bad Request"))}}
	_, err = client.ListTaskDefinitionRevisions(context.TODO(), aws.String("goodtd"))
	if err == nil {
		t.Fatal("expected no error from ListTaskDefinitionRevisions, got:", err)
	} else {
		if aerr, ok := errors.Cause(err).(apierror.Error); ok {
			t.Logf("got apierror '%s'", aerr)
		} else {
			t.Errorf("expected error to be an apierror.Error, got %s", err)
		}
	}
}

func TestDeleteTaskDefinition(t *testing.T) {
	client := ECS{Service: &mockECSClient{t: t}}
	_, err := client.DeleteTaskDefinition(context.TODO(), aws.String(""))
	if err == nil {
		t.Fatal("expected error from DeleteTaskDefinition with empty td, got nil")
	}

	_, err = client.DeleteTaskDefinition(context.TODO(), nil)
	if err == nil {
		t.Fatal("expected error from DeleteTaskDefinition with nil td, got nil")
	}

	_, err = client.DeleteTaskDefinition(context.TODO(), aws.String("somenotfoundtd"))
	if err == nil {
		t.Fatal("expected error from DeleteTaskDefinition, got:", err)
	}

	td, err := client.DeleteTaskDefinition(context.TODO(), goodTd1.TaskDefinitionArn)
	if err != nil {
		t.Fatal("expected no error from DeleteTaskDefinition, got:", err)
	}

	t.Log("got task definition list", td)

	if !reflect.DeepEqual(goodTd1, td) {
		t.Fatalf("Expected %+v\nGot %+v", goodTd1, td)
	}

	client = ECS{Service: &mockECSClient{t: t, err: awserr.New("400", "Bad Request", errors.New("Bad Request"))}}
	_, err = client.DeleteTaskDefinition(context.TODO(), aws.String("goodtd"))
	if err == nil {
		t.Fatal("expected error from DeleteTaskDefinition, got:", err)
	} else {
		if aerr, ok := errors.Cause(err).(apierror.Error); ok {
			t.Logf("got apierror '%s'", aerr)
		} else {
			t.Errorf("expected error to be an apierror.Error, got %s", err)
		}
	}
}
