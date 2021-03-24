package iam

import (
	"context"
	"net/url"

	"github.com/YaleSpinup/apierror"

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
func (i *IAM) GetRole(ctx context.Context, roleName string) (*iam.Role, error) {
	if roleName == "" {
		return nil, apierror.New(apierror.ErrBadRequest, "invalid input", nil)
	}

	log.Infof("getting iam role %s", roleName)

	output, err := i.Service.GetRoleWithContext(ctx, &iam.GetRoleInput{
		RoleName: aws.String(roleName),
	})
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

	out, err := i.Service.PutRolePolicyWithContext(ctx, input)
	if err != nil {
		return ErrCode("failed to attach policy to role", err)
	}

	log.Debugf("got output from put role policy %+v", out)

	return nil
}

// GetRolePolicy gets the inline policy attached to an IAM role
func (i *IAM) GetRolePolicy(ctx context.Context, role, policy string) (string, error) {
	if role == "" || policy == "" {
		return "", apierror.New(apierror.ErrBadRequest, "invalid input", nil)
	}

	log.Infof("getting policy %s for role %s", policy, role)

	out, err := i.Service.GetRolePolicyWithContext(ctx, &iam.GetRolePolicyInput{
		PolicyName: aws.String(policy),
		RoleName:   aws.String(role),
	})
	if err != nil {
		return "", ErrCode("failed to get role policy", err)
	}

	log.Debugf("got output from getting role policy %+v", out)

	// Document is returned url encoded, we must decode it to unmarshal and compare
	d, err := url.QueryUnescape(aws.StringValue(out.PolicyDocument))
	if err != nil {
		return "", err
	}

	log.Debugf("decoded policy document %s", d)

	return d, nil
}

// ListRolePolicies lists the inline policies for a role
func (i *IAM) ListRolePolicies(ctx context.Context, role string) ([]string, error) {
	if role == "" {
		return nil, apierror.New(apierror.ErrBadRequest, "invalid input", nil)
	}

	log.Infof("listing polcies for role %s", role)

	out, err := i.Service.ListRolePoliciesWithContext(ctx, &iam.ListRolePoliciesInput{
		RoleName: aws.String(role),
	})
	if err != nil {
		return nil, ErrCode("failed to list role polcies", err)
	}

	log.Debugf("got output listing role policies for '%s': %+v", role, out)

	return aws.StringValueSlice(out.PolicyNames), nil
}

// DeleteRolePolicy deletes an inline policy for a role
func (i *IAM) DeleteRolePolicy(ctx context.Context, role, policy string) error {
	if role == "" || policy == "" {
		return apierror.New(apierror.ErrBadRequest, "invalid input", nil)
	}

	log.Infof("deleting policy %s for role %s", policy, role)

	if _, err := i.Service.DeleteRolePolicyWithContext(ctx, &iam.DeleteRolePolicyInput{
		RoleName:   aws.String(role),
		PolicyName: aws.String(policy),
	}); err != nil {
		return ErrCode("failed to delete role policy", err)
	}

	return nil
}

// TagRole adds tags to an IAM role
func (i *IAM) TagRole(ctx context.Context, role string, tags []*iam.Tag) error {
	if role == "" || tags == nil {
		return apierror.New(apierror.ErrBadRequest, "invalid input", nil)
	}

	log.Infof("tagging role %s", role)

	if _, err := i.Service.TagRoleWithContext(ctx, &iam.TagRoleInput{
		RoleName: aws.String(role),
		Tags:     tags,
	}); err != nil {
		return ErrCode("failed to tag role", err)
	}

	return nil
}
