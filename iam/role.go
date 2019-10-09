package iam

import (
	"context"

	"github.com/YaleSpinup/ecs-api/apierror"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/iam"
	log "github.com/sirupsen/logrus"
)

// CreateRole handles creating an IAM role
func (i *IAM) CreateRole(ctx context.Context, input *iam.CreateRoleInput) (*iam.Role, error) {
	if input == nil {
		return nil, apierror.New(apierror.ErrBadRequest, "invalid input", nil)
	}

	log.Infof("creating iam role: %s", *input.RoleName)

	output, err := i.Service.CreateRoleWithContext(ctx, input)
	if err != nil {
		return nil, ErrCode("failed to create role", err)
	}

	return output.Role, nil
}

// DeleteRole handles deleting an IAM role
func (i *IAM) DeleteRole(ctx context.Context, input *iam.DeleteRoleInput) error {
	if input == nil || aws.StringValue(input.RoleName) == "" {
		return apierror.New(apierror.ErrBadRequest, "invalid input", nil)
	}

	log.Infof("deleting iam role %s", aws.StringValue(input.RoleName))

	_, err := i.Service.DeleteRoleWithContext(ctx, input)
	if err != nil {
		return ErrCode("failed to delete role", err)
	}

	return nil
}

// GetRole handles getting information about an IAM role
func (i *IAM) GetRole(ctx context.Context, input *iam.GetRoleInput) (*iam.Role, error) {
	if input == nil || aws.StringValue(input.RoleName) == "" {
		return nil, apierror.New(apierror.ErrBadRequest, "invalid input", nil)
	}

	log.Infof("getting iam role %s", aws.StringValue(input.RoleName))

	output, err := i.Service.GetRoleWithContext(ctx, input)
	if err != nil {
		return nil, ErrCode("failed to get role", err)
	}

	return output.Role, nil
}

// PutRolePolicy handles attaching an inline policy to IAM role
func (i *IAM) PutRolePolicy(ctx context.Context, input *iam.PutRolePolicyInput) error {
	if input == nil || aws.StringValue(input.RoleName) == "" || aws.StringValue(input.PolicyDocument) == "" || aws.StringValue(input.PolicyName) == "" {
		return apierror.New(apierror.ErrBadRequest, "invalid input", nil)
	}

	log.Infof("attaching inline policy to iam role: %s", *input.RoleName)

	_, err := i.Service.PutRolePolicyWithContext(ctx, input)
	if err != nil {
		return ErrCode("failed to attach policy to role", err)
	}

	return nil
}
