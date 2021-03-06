package ecs

import (
	"context"

	"github.com/YaleSpinup/apierror"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ecs"
	log "github.com/sirupsen/logrus"
)

// CreateTaskDefinition creates a task definition with context and input
func (e *ECS) CreateTaskDefinition(ctx context.Context, input *ecs.RegisterTaskDefinitionInput) (*ecs.TaskDefinition, error) {
	if input == nil {
		return nil, apierror.New(apierror.ErrBadRequest, "invalid input", nil)
	}

	log.Infof("creating task definition '%s'", aws.StringValue(input.Family))

	output, err := e.Service.RegisterTaskDefinitionWithContext(ctx, input)
	if err != nil {
		return nil, ErrCode("failed to create task definition", err)
	}

	log.Debugf("got output from RegisterTaskDefinition: %+v", output)

	return output.TaskDefinition, err
}

// DeleteTaskDefinition deleted a task definition
func (e *ECS) DeleteTaskDefinition(ctx context.Context, taskdefinition *string) (*ecs.TaskDefinition, error) {
	if aws.StringValue(taskdefinition) == "" {
		return nil, apierror.New(apierror.ErrBadRequest, "invalid input", nil)
	}

	log.Infof("deregistering task definition '%s'", aws.StringValue(taskdefinition))

	output, err := e.Service.DeregisterTaskDefinitionWithContext(ctx, &ecs.DeregisterTaskDefinitionInput{TaskDefinition: taskdefinition})
	if err != nil {
		return nil, ErrCode("failed to delete task definition", err)
	}

	log.Debugf("got output from DeregisterTaskDefinition: %+v", output)

	return output.TaskDefinition, err
}

// GetTaskDefinition gets a task definition with context by name
func (e *ECS) GetTaskDefinition(ctx context.Context, taskdefinition *string, tags bool) (*ecs.TaskDefinition, []*ecs.Tag, error) {
	if aws.StringValue(taskdefinition) == "" {
		return nil, nil, apierror.New(apierror.ErrBadRequest, "invalid input", nil)
	}

	log.Infof("getting details about task definition '%s'", aws.StringValue(taskdefinition))

	input := ecs.DescribeTaskDefinitionInput{
		TaskDefinition: taskdefinition,
	}

	if tags {
		input.Include = aws.StringSlice([]string{"TAGS"})
	}

	output, err := e.Service.DescribeTaskDefinitionWithContext(ctx, &input)
	if err != nil {
		return nil, nil, ErrCode("failed to get task definition", err)
	}

	log.Debugf("got output from DescribeTaskDefinition: %+v", output)

	return output.TaskDefinition, output.Tags, err
}

// ListTaskDefinitionRevisions lists all of the task definition [revisions] in a family
func (e *ECS) ListTaskDefinitionRevisions(ctx context.Context, family *string) ([]string, error) {
	if aws.StringValue(family) == "" {
		return nil, apierror.New(apierror.ErrBadRequest, "invalid input", nil)
	}

	log.Infof("listing task definition revisions with family '%s'", aws.StringValue(family))

	input := ecs.ListTaskDefinitionsInput{
		FamilyPrefix: family,
	}

	output := []string{}
	for {
		out, err := e.Service.ListTaskDefinitionsWithContext(ctx, &input)
		if err != nil {
			return output, ErrCode("failed to list taskdefinitions in family"+aws.StringValue(family), err)
		}

		for _, t := range out.TaskDefinitionArns {
			output = append(output, aws.StringValue(t))
		}

		if out.NextToken == nil {
			break
		}
		input.NextToken = out.NextToken
	}

	log.Debugf("got list of task definitions in family '%s': %+v", aws.StringValue(family), output)

	return output, nil
}
