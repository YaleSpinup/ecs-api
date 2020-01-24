package ecs

import (
	"context"

	"github.com/aws/aws-sdk-go/service/ecs"
)

// CreateTaskDefinition creates a task definition with context and input
func (e *ECS) CreateTaskDefinition(ctx context.Context, input *ecs.RegisterTaskDefinitionInput) (*ecs.TaskDefinition, error) {
	output, err := e.Service.RegisterTaskDefinitionWithContext(ctx, input)
	if err != nil {
		return nil, ErrCode("failed to create task definition", err)
	}

	return output.TaskDefinition, err
}

// GetTaskDefinition gets a task definition with context by name
func (e *ECS) GetTaskDefinition(ctx context.Context, name *string) (*ecs.TaskDefinition, error) {
	output, err := e.Service.DescribeTaskDefinitionWithContext(ctx, &ecs.DescribeTaskDefinitionInput{
		TaskDefinition: name,
	})

	if err != nil {
		return nil, ErrCode("failed to get task definition", err)
	}

	return output.TaskDefinition, err
}
