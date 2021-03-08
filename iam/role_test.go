package iam

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/YaleSpinup/apierror"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/request"
	"github.com/aws/aws-sdk-go/service/iam"
	"github.com/aws/aws-sdk-go/service/iam/iamiface"
)

var testTime = time.Now()

var testRole = iam.Role{
	Arn:         aws.String("arn:aws:iam::12345678910:role/testrole"),
	CreateDate:  &testTime,
	Description: aws.String("role model"),
	Path:        aws.String("/"),
	RoleId:      aws.String("TESTROLEID123"),
	RoleName:    aws.String("testrole"),
}

var testPolicyDoc = PolicyDoc{
	Version: "2012-10-17",
	Statement: []PolicyStatement{
		{
			Effect: "Allow",
			Action: []string{
				"logs:CreateLogGroup",
				"logs:CreateLogStream",
				"logs:PutLogEvents",
			},
			Resource: []string{"*"},
		},
	},
}

func (m *mockIAMClient) CreateRoleWithContext(ctx context.Context, input *iam.CreateRoleInput, opts ...request.Option) (*iam.CreateRoleOutput, error) {
	var output = &iam.CreateRoleOutput{Role: &iam.Role{
		Arn:         aws.String(fmt.Sprintf("arn:aws:iam::12345678910:role/%s", *input.RoleName)),
		CreateDate:  &testTime,
		Description: input.Description,
		Path:        input.Path,
		RoleId:      aws.String(strings.ToUpper(fmt.Sprintf("%sID123", *input.RoleName))),
		RoleName:    input.RoleName,
	}}

	if m.err != nil {
		if aerr, ok := (m.err).(awserr.Error); ok {
			if aerr.Code() == "TestNoSuchEntity" {
				return output, nil
			}
		}
		return nil, m.err
	}

	return output, nil
}

func (m *mockIAMClient) DeleteRoleWithContext(ctx context.Context, input *iam.DeleteRoleInput, opts ...request.Option) (*iam.DeleteRoleOutput, error) {
	if m.err != nil {
		return nil, m.err
	}
	return &iam.DeleteRoleOutput{}, nil
}

func (m *mockIAMClient) GetRoleWithContext(ctx context.Context, input *iam.GetRoleInput, opts ...request.Option) (*iam.GetRoleOutput, error) {
	var output = &iam.GetRoleOutput{Role: &testRole}

	if m.err != nil {
		if aerr, ok := (m.err).(awserr.Error); ok {
			if aerr.Code() == "TestNoSuchEntity" {
				return nil, awserr.New(iam.ErrCodeNoSuchEntityException, "NoSuchEntity", nil)
			}
		}
		return nil, m.err
	}

	return output, nil
}

func (m *mockIAMClient) PutRolePolicyWithContext(ctx context.Context, input *iam.PutRolePolicyInput, opts ...request.Option) (*iam.PutRolePolicyOutput, error) {
	var output = &iam.PutRolePolicyOutput{}

	if m.err != nil {
		if aerr, ok := (m.err).(awserr.Error); ok {
			if aerr.Code() == "TestNoSuchEntity" {
				return output, nil
			}
		}
		return nil, m.err
	}

	return output, nil
}

func (m *mockIAMClient) GetRolePolicyWithContext(ctx context.Context, input *iam.GetRolePolicyInput, opts ...request.Option) (*iam.GetRolePolicyOutput, error) {
	if m.err != nil {
		return nil, m.err
	}

	if aws.StringValue(input.RoleName) != "testRole" {
		return nil, awserr.New(iam.ErrCodeNoSuchEntityException, "entity not found", nil)
	}

	if aws.StringValue(input.PolicyName) == "badPolicy" {
		return &iam.GetRolePolicyOutput{
			RoleName:       aws.String("testRole"),
			PolicyName:     aws.String("badPolcyDoc"),
			PolicyDocument: aws.String("%A"),
		}, nil
	}

	if aws.StringValue(input.PolicyName) != "testPolicy" {
		return nil, awserr.New(iam.ErrCodeNoSuchEntityException, "entity not found", nil)
	}

	d := url.QueryEscape(string(testPolicyDocument))

	return &iam.GetRolePolicyOutput{
		RoleName:       aws.String("testRole"),
		PolicyName:     aws.String("testPolicy"),
		PolicyDocument: aws.String(d),
	}, nil
}

func (m *mockIAMClient) ListRolePoliciesWithContext(ctx context.Context, input *iam.ListRolePoliciesInput, opts ...request.Option) (*iam.ListRolePoliciesOutput, error) {
	if m.err != nil {
		return nil, m.err
	}

	if aws.StringValue(input.RoleName) != "testRole" {
		return nil, awserr.New(iam.ErrCodeNoSuchEntityException, "entity not found", nil)
	}

	return &iam.ListRolePoliciesOutput{
		PolicyNames: aws.StringSlice([]string{"testPolicy"}),
	}, nil
}

func (m *mockIAMClient) DeleteRolePolicyWithContext(ctx context.Context, input *iam.DeleteRolePolicyInput, opts ...request.Option) (*iam.DeleteRolePolicyOutput, error) {
	if m.err != nil {
		return nil, m.err
	}

	if aws.StringValue(input.RoleName) != "testRole" || aws.StringValue(input.PolicyName) != "testPolicy" {
		return nil, awserr.New(iam.ErrCodeNoSuchEntityException, "entity not found", nil)
	}

	return &iam.DeleteRolePolicyOutput{}, nil
}

func TestCreateRole(t *testing.T) {
	i := IAM{
		Service:         newMockIAMClient(t, nil),
		DefaultKmsKeyID: "12345678-90ab-cdef-1234-567890abcdef",
	}

	defaultPolicy, err := json.Marshal(PolicyDoc{
		Version: "2012-10-17",
		Statement: []PolicyStatement{
			{
				Effect: "Allow",
				Action: []string{
					"sts:AssumeRole",
				},
				Principal: map[string][]string{
					"Service": {"ecs-tasks.amazonaws.com"},
				},
			},
		},
	})
	if err != nil {
		t.Errorf("expected nil error creating default policy doc, got %s", err)
	}

	// test success
	expected := &testRole
	out, err := i.CreateRole(context.TODO(), &iam.CreateRoleInput{
		AssumeRolePolicyDocument: aws.String(string(defaultPolicy)),
		Description:              aws.String("role model"),
		Path:                     aws.String("/"),
		RoleName:                 aws.String("testrole"),
	})
	if err != nil {
		t.Errorf("expected nil error, got: %s", err)
	}

	if !reflect.DeepEqual(out, expected) {
		t.Errorf("expected %+v, got %+v", expected, out)
	}

	// test nil input
	_, err = i.CreateRole(context.TODO(), nil)
	if aerr, ok := err.(apierror.Error); ok {
		if aerr.Code != apierror.ErrBadRequest {
			t.Errorf("expected error code %s, got: %s", apierror.ErrBadRequest, aerr.Code)
		}
	} else {
		t.Errorf("expected apierror.Error, got: %s", reflect.TypeOf(err).String())
	}

	// test ErrCodeInvalidInputException
	i.Service.(*mockIAMClient).err = awserr.New(iam.ErrCodeInvalidInputException, "InvalidInput", nil)
	_, err = i.CreateRole(context.TODO(), &iam.CreateRoleInput{
		AssumeRolePolicyDocument: aws.String(string(defaultPolicy)),
		Path:                     aws.String("/"),
		RoleName:                 aws.String("testrole"),
	})
	if aerr, ok := err.(apierror.Error); ok {
		if aerr.Code != apierror.ErrBadRequest {
			t.Errorf("expected error code %s, got: %s", apierror.ErrBadRequest, aerr.Code)
		}
	} else {
		t.Errorf("expected apierror.Error, got: %s", reflect.TypeOf(err).String())
	}

	// test ErrCodeMalformedPolicyDocumentException
	i.Service.(*mockIAMClient).err = awserr.New(iam.ErrCodeMalformedPolicyDocumentException, "MalformedPolicyDocument", nil)
	_, err = i.CreateRole(context.TODO(), &iam.CreateRoleInput{
		AssumeRolePolicyDocument: aws.String(string(defaultPolicy)),
		Path:                     aws.String("/"),
		RoleName:                 aws.String("testrole"),
	})
	if aerr, ok := err.(apierror.Error); ok {
		if aerr.Code != apierror.ErrBadRequest {
			t.Errorf("expected error code %s, got: %s", apierror.ErrBadRequest, aerr.Code)
		}
	} else {
		t.Errorf("expected apierror.Error, got: %s", reflect.TypeOf(err).String())
	}

	// test ErrCodeLimitExceededException
	i.Service.(*mockIAMClient).err = awserr.New(iam.ErrCodeLimitExceededException, "LimitExceeded", nil)
	_, err = i.CreateRole(context.TODO(), &iam.CreateRoleInput{
		AssumeRolePolicyDocument: aws.String(string(defaultPolicy)),
		Path:                     aws.String("/"),
		RoleName:                 aws.String("testrole"),
	})
	if aerr, ok := err.(apierror.Error); ok {
		if aerr.Code != apierror.ErrLimitExceeded {
			t.Errorf("expected error code %s, got: %s", apierror.ErrLimitExceeded, aerr.Code)
		}
	} else {
		t.Errorf("expected apierror.Error, got: %s", reflect.TypeOf(err).String())
	}

	// test ErrCodeConcurrentModificationException
	i.Service.(*mockIAMClient).err = awserr.New(iam.ErrCodeConcurrentModificationException, "ConcurrentModification", nil)
	_, err = i.CreateRole(context.TODO(), &iam.CreateRoleInput{
		AssumeRolePolicyDocument: aws.String(string(defaultPolicy)),
		Path:                     aws.String("/"),
		RoleName:                 aws.String("testrole"),
	})
	if aerr, ok := err.(apierror.Error); ok {
		if aerr.Code != apierror.ErrConflict {
			t.Errorf("expected error code %s, got: %s", apierror.ErrConflict, aerr.Code)
		}
	} else {
		t.Errorf("expected apierror.Error, got: %s", reflect.TypeOf(err).String())
	}

	// test ErrCodeEntityAlreadyExistsException
	i.Service.(*mockIAMClient).err = awserr.New(iam.ErrCodeEntityAlreadyExistsException, "EntityAlreadyExists", nil)
	_, err = i.CreateRole(context.TODO(), &iam.CreateRoleInput{
		AssumeRolePolicyDocument: aws.String(string(defaultPolicy)),
		Path:                     aws.String("/"),
		RoleName:                 aws.String("testrole"),
	})
	if aerr, ok := err.(apierror.Error); ok {
		if aerr.Code != apierror.ErrConflict {
			t.Errorf("expected error code %s, got: %s", apierror.ErrConflict, aerr.Code)
		}
	} else {
		t.Errorf("expected apierror.Error, got: %s", reflect.TypeOf(err).String())
	}

	// test ErrCodeServiceFailureException
	i.Service.(*mockIAMClient).err = awserr.New(iam.ErrCodeServiceFailureException, "ServiceFailure", nil)
	_, err = i.CreateRole(context.TODO(), &iam.CreateRoleInput{
		AssumeRolePolicyDocument: aws.String(string(defaultPolicy)),
		Path:                     aws.String("/"),
		RoleName:                 aws.String("testrole"),
	})
	if aerr, ok := err.(apierror.Error); ok {
		if aerr.Code != apierror.ErrServiceUnavailable {
			t.Errorf("expected error code %s, got: %s", apierror.ErrServiceUnavailable, aerr.Code)
		}
	} else {
		t.Errorf("expected apierror.Error, got: %s", reflect.TypeOf(err).String())
	}

	// test some other, unexpected AWS error
	i.Service.(*mockIAMClient).err = awserr.New("UnknownThingyBrokeYo", "ThingyBroke", nil)
	_, err = i.CreateRole(context.TODO(), &iam.CreateRoleInput{
		AssumeRolePolicyDocument: aws.String(string(defaultPolicy)),
		Path:                     aws.String("/"),
		RoleName:                 aws.String("testrole"),
	})
	if aerr, ok := err.(apierror.Error); ok {
		if aerr.Code != apierror.ErrBadRequest {
			t.Errorf("expected error code %s, got: %s", apierror.ErrBadRequest, aerr.Code)
		}
	} else {
		t.Errorf("expected apierror.Error, got: %s", reflect.TypeOf(err).String())
	}

	// test non-aws error
	i.Service.(*mockIAMClient).err = errors.New("things blowing up")
	_, err = i.CreateRole(context.TODO(), &iam.CreateRoleInput{
		AssumeRolePolicyDocument: aws.String(string(defaultPolicy)),
		Path:                     aws.String("/"),
		RoleName:                 aws.String("testrole"),
	})
	if aerr, ok := err.(apierror.Error); ok {
		if aerr.Code != apierror.ErrInternalError {
			t.Errorf("expected error code %s, got: %s", apierror.ErrInternalError, aerr.Code)
		}
	} else {
		t.Errorf("expected apierror.Error, got: %s", reflect.TypeOf(err).String())
	}
}

func TestDeleteRole(t *testing.T) {
	i := IAM{
		Service:         newMockIAMClient(t, nil),
		DefaultKmsKeyID: "12345678-90ab-cdef-1234-567890abcdef",
	}

	// test success
	err := i.DeleteRole(context.TODO(), &iam.DeleteRoleInput{RoleName: aws.String("testrole")})
	if err != nil {
		t.Errorf("expected nil error, got: %s", err)
	}

	// test nil input
	err = i.DeleteRole(context.TODO(), nil)
	if aerr, ok := err.(apierror.Error); ok {
		if aerr.Code != apierror.ErrBadRequest {
			t.Errorf("expected error code %s, got: %s", apierror.ErrBadRequest, aerr.Code)
		}
	} else {
		t.Errorf("expected apierror.Error, got: %s", reflect.TypeOf(err).String())
	}

	// test empty policy arn
	err = i.DeleteRole(context.TODO(), &iam.DeleteRoleInput{})
	if aerr, ok := err.(apierror.Error); ok {
		if aerr.Code != apierror.ErrBadRequest {
			t.Errorf("expected error code %s, got: %s", apierror.ErrBadRequest, aerr.Code)
		}
	} else {
		t.Errorf("expected apierror.Error, got: %s", reflect.TypeOf(err).String())
	}

	// test ErrCodeNoSuchEntityException
	i.Service.(*mockIAMClient).err = awserr.New(iam.ErrCodeNoSuchEntityException, "NoSuchEntity", nil)
	err = i.DeleteRole(context.TODO(), &iam.DeleteRoleInput{RoleName: aws.String("rolenotfound")})
	if aerr, ok := err.(apierror.Error); ok {
		if aerr.Code != apierror.ErrNotFound {
			t.Errorf("expected error code %s, got: %s", apierror.ErrNotFound, aerr.Code)
		}
	} else {
		t.Errorf("expected apierror.Error, got: %s", reflect.TypeOf(err).String())
	}

	// test ErrCodeLimitExceededException
	i.Service.(*mockIAMClient).err = awserr.New(iam.ErrCodeLimitExceededException, "LimitExceeded", nil)
	err = i.DeleteRole(context.TODO(), &iam.DeleteRoleInput{RoleName: aws.String("testrole")})
	if aerr, ok := err.(apierror.Error); ok {
		if aerr.Code != apierror.ErrLimitExceeded {
			t.Errorf("expected error code %s, got: %s", apierror.ErrLimitExceeded, aerr.Code)
		}
	} else {
		t.Errorf("expected apierror.Error, got: %s", reflect.TypeOf(err).String())
	}

	// test ErrCodeUnmodifiableEntityException
	i.Service.(*mockIAMClient).err = awserr.New(iam.ErrCodeUnmodifiableEntityException, "UnmodifiableEntity", nil)
	err = i.DeleteRole(context.TODO(), &iam.DeleteRoleInput{RoleName: aws.String("testrole")})
	if aerr, ok := err.(apierror.Error); ok {
		if aerr.Code != apierror.ErrInternalError {
			t.Errorf("expected error code %s, got: %s", apierror.ErrInternalError, aerr.Code)
		}
	} else {
		t.Errorf("expected apierror.Error, got: %s", reflect.TypeOf(err).String())
	}

	// test ErrCodeConcurrentModificationException
	i.Service.(*mockIAMClient).err = awserr.New(iam.ErrCodeConcurrentModificationException, "ConcurrentModification", nil)
	err = i.DeleteRole(context.TODO(), &iam.DeleteRoleInput{RoleName: aws.String("testrole")})
	if aerr, ok := err.(apierror.Error); ok {
		if aerr.Code != apierror.ErrConflict {
			t.Errorf("expected error code %s, got: %s", apierror.ErrConflict, aerr.Code)
		}
	} else {
		t.Errorf("expected apierror.Error, got: %s", reflect.TypeOf(err).String())
	}

	// test ErrCodeDeleteConflictException
	i.Service.(*mockIAMClient).err = awserr.New(iam.ErrCodeDeleteConflictException, "DeleteConflict", nil)
	err = i.DeleteRole(context.TODO(), &iam.DeleteRoleInput{RoleName: aws.String("testrole")})
	if aerr, ok := err.(apierror.Error); ok {
		if aerr.Code != apierror.ErrConflict {
			t.Errorf("expected error code %s, got: %s", apierror.ErrConflict, aerr.Code)
		}
	} else {
		t.Errorf("expected apierror.Error, got: %s", reflect.TypeOf(err).String())
	}

	// test ErrCodeServiceFailureException
	i.Service.(*mockIAMClient).err = awserr.New(iam.ErrCodeServiceFailureException, "ServiceFailure", nil)
	err = i.DeleteRole(context.TODO(), &iam.DeleteRoleInput{RoleName: aws.String("testrole")})
	if aerr, ok := err.(apierror.Error); ok {
		if aerr.Code != apierror.ErrServiceUnavailable {
			t.Errorf("expected error code %s, got: %s", apierror.ErrServiceUnavailable, aerr.Code)
		}
	} else {
		t.Errorf("expected apierror.Error, got: %s", reflect.TypeOf(err).String())
	}

	// test some other, unexpected AWS error
	i.Service.(*mockIAMClient).err = awserr.New("UnknownThingyBrokeYo", "ThingyBroke", nil)
	err = i.DeleteRole(context.TODO(), &iam.DeleteRoleInput{RoleName: aws.String("testrole")})
	if aerr, ok := err.(apierror.Error); ok {
		if aerr.Code != apierror.ErrBadRequest {
			t.Errorf("expected error code %s, got: %s", apierror.ErrBadRequest, aerr.Code)
		}
	} else {
		t.Errorf("expected apierror.Error, got: %s", reflect.TypeOf(err).String())
	}

	// test non-aws error
	i.Service.(*mockIAMClient).err = errors.New("things blowing up")
	err = i.DeleteRole(context.TODO(), &iam.DeleteRoleInput{RoleName: aws.String("testrole")})
	if aerr, ok := err.(apierror.Error); ok {
		if aerr.Code != apierror.ErrInternalError {
			t.Errorf("expected error code %s, got: %s", apierror.ErrInternalError, aerr.Code)
		}
	} else {
		t.Errorf("expected apierror.Error, got: %s", reflect.TypeOf(err).String())
	}
}

func TestGetRole(t *testing.T) {
	i := IAM{
		Service:         newMockIAMClient(t, nil),
		DefaultKmsKeyID: "12345678-90ab-cdef-1234-567890abcdef",
	}

	// test success
	expected := &testRole
	out, err := i.GetRole(context.TODO(), "testRole")
	if err != nil {
		t.Errorf("expected nil error, got: %s", err)
	}

	if !reflect.DeepEqual(out, expected) {
		t.Errorf("expected %+v, got %+v", expected, out)
	}

	// test empty role name
	_, err = i.GetRole(context.TODO(), "")
	if aerr, ok := err.(apierror.Error); ok {
		if aerr.Code != apierror.ErrBadRequest {
			t.Errorf("expected error code %s, got: %s", apierror.ErrBadRequest, aerr.Code)
		}
	} else {
		t.Errorf("expected apierror.Error, got: %s", reflect.TypeOf(err).String())
	}

	// test ErrCodeNoSuchEntityException
	i.Service.(*mockIAMClient).err = awserr.New(iam.ErrCodeNoSuchEntityException, "NoSuchEntity", nil)
	_, err = i.GetRole(context.TODO(), "testrole")
	if aerr, ok := err.(apierror.Error); ok {
		if aerr.Code != apierror.ErrNotFound {
			t.Errorf("expected error code %s, got: %s", apierror.ErrNotFound, aerr.Code)
		}
	} else {
		t.Errorf("expected apierror.Error, got: %s", reflect.TypeOf(err).String())
	}

	// test ErrCodeServiceFailureException
	i.Service.(*mockIAMClient).err = awserr.New(iam.ErrCodeServiceFailureException, "ServiceFailure", nil)
	_, err = i.GetRole(context.TODO(), "testrole")
	if aerr, ok := err.(apierror.Error); ok {
		if aerr.Code != apierror.ErrServiceUnavailable {
			t.Errorf("expected error code %s, got: %s", apierror.ErrServiceUnavailable, aerr.Code)
		}
	} else {
		t.Errorf("expected apierror.Error, got: %s", reflect.TypeOf(err).String())
	}

	// test some other, unexpected AWS error
	i.Service.(*mockIAMClient).err = awserr.New("UnknownThingyBrokeYo", "ThingyBroke", nil)
	_, err = i.GetRole(context.TODO(), "testrole")
	if aerr, ok := err.(apierror.Error); ok {
		if aerr.Code != apierror.ErrBadRequest {
			t.Errorf("expected error code %s, got: %s", apierror.ErrBadRequest, aerr.Code)
		}
	} else {
		t.Errorf("expected apierror.Error, got: %s", reflect.TypeOf(err).String())
	}

	// test non-aws error
	i.Service.(*mockIAMClient).err = errors.New("things blowing up")
	_, err = i.GetRole(context.TODO(), "testrole")
	if aerr, ok := err.(apierror.Error); ok {
		if aerr.Code != apierror.ErrInternalError {
			t.Errorf("expected error code %s, got: %s", apierror.ErrInternalError, aerr.Code)
		}
	} else {
		t.Errorf("expected apierror.Error, got: %s", reflect.TypeOf(err).String())
	}
}

func TestPutRolePolicy(t *testing.T) {
	i := IAM{
		Service:         newMockIAMClient(t, nil),
		DefaultKmsKeyID: "12345678-90ab-cdef-1234-567890abcdef",
	}

	testPolicy, err := json.Marshal(testPolicyDoc)
	if err != nil {
		t.Errorf("expected nil error, got: %s", err)
	}

	// test success
	err = i.PutRolePolicy(context.TODO(), &iam.PutRolePolicyInput{
		PolicyDocument: aws.String(string(testPolicy)),
		PolicyName:     aws.String("testpolicy"),
		RoleName:       aws.String("testrole"),
	})
	if err != nil {
		t.Errorf("expected nil error, got: %s", err)
	}

	// test nil input
	err = i.PutRolePolicy(context.TODO(), nil)
	if aerr, ok := err.(apierror.Error); ok {
		if aerr.Code != apierror.ErrBadRequest {
			t.Errorf("expected error code %s, got: %s", apierror.ErrBadRequest, aerr.Code)
		}
	} else {
		t.Errorf("expected apierror.Error, got: %s", reflect.TypeOf(err).String())
	}

	// test empty role name and empty policy doc
	err = i.PutRolePolicy(context.TODO(), &iam.PutRolePolicyInput{})
	if aerr, ok := err.(apierror.Error); ok {
		if aerr.Code != apierror.ErrBadRequest {
			t.Errorf("expected error code %s, got: %s", apierror.ErrBadRequest, aerr.Code)
		}
	} else {
		t.Errorf("expected apierror.Error, got: %s", reflect.TypeOf(err).String())
	}

	// test ErrCodeNoSuchEntityException
	i.Service.(*mockIAMClient).err = awserr.New(iam.ErrCodeNoSuchEntityException, "NoSuchEntity", nil)
	err = i.PutRolePolicy(context.TODO(), &iam.PutRolePolicyInput{
		PolicyDocument: aws.String(string(testPolicy)),
		PolicyName:     aws.String("testpolicy"),
		RoleName:       aws.String("testrole"),
	})
	if aerr, ok := err.(apierror.Error); ok {
		if aerr.Code != apierror.ErrNotFound {
			t.Errorf("expected error code %s, got: %s", apierror.ErrNotFound, aerr.Code)
		}
	} else {
		t.Errorf("expected apierror.Error, got: %s", reflect.TypeOf(err).String())
	}

	// test ErrCodeLimitExceededException
	i.Service.(*mockIAMClient).err = awserr.New(iam.ErrCodeLimitExceededException, "LimitExceeded", nil)
	err = i.PutRolePolicy(context.TODO(), &iam.PutRolePolicyInput{
		PolicyDocument: aws.String(string(testPolicy)),
		PolicyName:     aws.String("testpolicy"),
		RoleName:       aws.String("testrole"),
	})
	if aerr, ok := err.(apierror.Error); ok {
		if aerr.Code != apierror.ErrLimitExceeded {
			t.Errorf("expected error code %s, got: %s", apierror.ErrLimitExceeded, aerr.Code)
		}
	} else {
		t.Errorf("expected apierror.Error, got: %s", reflect.TypeOf(err).String())
	}

	// test ErrCodeMalformedPolicyDocumentException
	i.Service.(*mockIAMClient).err = awserr.New(iam.ErrCodeMalformedPolicyDocumentException, "MalformedPolicyDocument", nil)
	err = i.PutRolePolicy(context.TODO(), &iam.PutRolePolicyInput{
		PolicyDocument: aws.String(string(testPolicy)),
		PolicyName:     aws.String("testpolicy"),
		RoleName:       aws.String("testrole"),
	})
	if aerr, ok := err.(apierror.Error); ok {
		if aerr.Code != apierror.ErrBadRequest {
			t.Errorf("expected error code %s, got: %s", apierror.ErrBadRequest, aerr.Code)
		}
	} else {
		t.Errorf("expected apierror.Error, got: %s", reflect.TypeOf(err).String())
	}

	// test ErrCodeUnmodifiableEntityException
	i.Service.(*mockIAMClient).err = awserr.New(iam.ErrCodeUnmodifiableEntityException, "UnmodifiableEntity", nil)
	err = i.PutRolePolicy(context.TODO(), &iam.PutRolePolicyInput{
		PolicyDocument: aws.String(string(testPolicy)),
		PolicyName:     aws.String("testpolicy"),
		RoleName:       aws.String("testrole"),
	})
	if aerr, ok := err.(apierror.Error); ok {
		if aerr.Code != apierror.ErrInternalError {
			t.Errorf("expected error code %s, got: %s", apierror.ErrInternalError, aerr.Code)
		}
	} else {
		t.Errorf("expected apierror.Error, got: %s", reflect.TypeOf(err).String())
	}

	// test ErrCodeServiceFailureException
	i.Service.(*mockIAMClient).err = awserr.New(iam.ErrCodeServiceFailureException, "ServiceFailure", nil)
	err = i.PutRolePolicy(context.TODO(), &iam.PutRolePolicyInput{
		PolicyDocument: aws.String(string(testPolicy)),
		PolicyName:     aws.String("testpolicy"),
		RoleName:       aws.String("testrole"),
	})
	if aerr, ok := err.(apierror.Error); ok {
		if aerr.Code != apierror.ErrServiceUnavailable {
			t.Errorf("expected error code %s, got: %s", apierror.ErrServiceUnavailable, aerr.Code)
		}
	} else {
		t.Errorf("expected apierror.Error, got: %s", reflect.TypeOf(err).String())
	}

	// test some other, unexpected AWS error
	i.Service.(*mockIAMClient).err = awserr.New("UnknownThingyBrokeYo", "ThingyBroke", nil)
	err = i.PutRolePolicy(context.TODO(), &iam.PutRolePolicyInput{
		PolicyDocument: aws.String(string(testPolicy)),
		PolicyName:     aws.String("testpolicy"),
		RoleName:       aws.String("testrole"),
	})
	if aerr, ok := err.(apierror.Error); ok {
		if aerr.Code != apierror.ErrBadRequest {
			t.Errorf("expected error code %s, got: %s", apierror.ErrBadRequest, aerr.Code)
		}
	} else {
		t.Errorf("expected apierror.Error, got: %s", reflect.TypeOf(err).String())
	}

	// test non-aws error
	i.Service.(*mockIAMClient).err = errors.New("things blowing up")
	err = i.PutRolePolicy(context.TODO(), &iam.PutRolePolicyInput{
		PolicyDocument: aws.String(string(testPolicy)),
		PolicyName:     aws.String("testpolicy"),
		RoleName:       aws.String("testrole"),
	})
	if aerr, ok := err.(apierror.Error); ok {
		if aerr.Code != apierror.ErrInternalError {
			t.Errorf("expected error code %s, got: %s", apierror.ErrInternalError, aerr.Code)
		}
	} else {
		t.Errorf("expected apierror.Error, got: %s", reflect.TypeOf(err).String())
	}
}

func TestIAM_GetRolePolicy(t *testing.T) {
	type fields struct {
		Service         iamiface.IAMAPI
		DefaultKmsKeyID string
	}
	type args struct {
		ctx    context.Context
		role   string
		policy string
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    string
		wantErr bool
	}{
		{
			name: "empty role and policy",
			fields: fields{
				Service:         newMockIAMClient(t, nil),
				DefaultKmsKeyID: "123",
			},
			args: args{
				ctx:    context.TODO(),
				role:   "",
				policy: "",
			},
			wantErr: true,
		},
		{
			name: "empty role",
			fields: fields{
				Service:         newMockIAMClient(t, nil),
				DefaultKmsKeyID: "123",
			},
			args: args{
				ctx:    context.TODO(),
				role:   "",
				policy: "testPolicy",
			},
			wantErr: true,
		},
		{
			name: "empty policy",
			fields: fields{
				Service:         newMockIAMClient(t, nil),
				DefaultKmsKeyID: "123",
			},
			args: args{
				ctx:    context.TODO(),
				role:   "testRole",
				policy: "",
			},
			wantErr: true,
		},
		{
			name: "example policy and role",
			fields: fields{
				Service:         newMockIAMClient(t, nil),
				DefaultKmsKeyID: "123",
			},
			args: args{
				ctx:    context.TODO(),
				role:   "testRole",
				policy: "testPolicy",
			},
			want: testPolicyDocument,
		},
		{
			name: "invalid url escaping in policy",
			fields: fields{
				Service:         newMockIAMClient(t, nil),
				DefaultKmsKeyID: "123",
			},
			args: args{
				ctx:    context.TODO(),
				role:   "testRole",
				policy: "badPolicy",
			},
			wantErr: true,
		},
		{
			name: "aws errror",
			fields: fields{
				Service:         newMockIAMClient(t, awserr.New(iam.ErrCodeInvalidInputException, "bad input", nil)),
				DefaultKmsKeyID: "123",
			},
			args: args{
				ctx:    context.TODO(),
				role:   "testRole",
				policy: "badPolicy",
			},
			wantErr: true,
		},
		{
			name: "non-aws errror",
			fields: fields{
				Service:         newMockIAMClient(t, errors.New("boom")),
				DefaultKmsKeyID: "123",
			},
			args: args{
				ctx:    context.TODO(),
				role:   "testRole",
				policy: "badPolicy",
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			i := &IAM{
				Service:         tt.fields.Service,
				DefaultKmsKeyID: tt.fields.DefaultKmsKeyID,
			}
			got, err := i.GetRolePolicy(tt.args.ctx, tt.args.role, tt.args.policy)
			if (err != nil) != tt.wantErr {
				t.Errorf("IAM.GetRolePolicy() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("IAM.GetRolePolicy() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIAM_ListRolePolicies(t *testing.T) {
	type fields struct {
		Service         iamiface.IAMAPI
		DefaultKmsKeyID string
	}
	type args struct {
		ctx  context.Context
		role string
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    []string
		wantErr bool
	}{
		{
			name: "empty role",
			fields: fields{
				Service:         newMockIAMClient(t, nil),
				DefaultKmsKeyID: "123",
			},
			args: args{
				ctx:  context.TODO(),
				role: "",
			},
			wantErr: true,
		},
		{
			name: "example role",
			fields: fields{
				Service:         newMockIAMClient(t, nil),
				DefaultKmsKeyID: "123",
			},
			args: args{
				ctx:  context.TODO(),
				role: "testRole",
			},
			want: []string{"testPolicy"},
		},
		{
			name: "aws errror",
			fields: fields{
				Service:         newMockIAMClient(t, awserr.New(iam.ErrCodeInvalidInputException, "bad input", nil)),
				DefaultKmsKeyID: "123",
			},
			args: args{
				ctx:  context.TODO(),
				role: "testRole",
			},
			wantErr: true,
		},
		{
			name: "non-aws errror",
			fields: fields{
				Service:         newMockIAMClient(t, errors.New("boom")),
				DefaultKmsKeyID: "123",
			},
			args: args{
				ctx:  context.TODO(),
				role: "testRole",
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			i := &IAM{
				Service:         tt.fields.Service,
				DefaultKmsKeyID: tt.fields.DefaultKmsKeyID,
			}
			got, err := i.ListRolePolicies(tt.args.ctx, tt.args.role)
			if (err != nil) != tt.wantErr {
				t.Errorf("IAM.ListRolePolicies() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("IAM.ListRolePolicies() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIAM_DeleteRolePolicy(t *testing.T) {
	type fields struct {
		Service         iamiface.IAMAPI
		DefaultKmsKeyID string
	}
	type args struct {
		ctx    context.Context
		role   string
		policy string
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			name: "empty role and policy",
			fields: fields{
				Service:         newMockIAMClient(t, nil),
				DefaultKmsKeyID: "123",
			},
			args: args{
				ctx:    context.TODO(),
				role:   "",
				policy: "",
			},
			wantErr: true,
		},
		{
			name: "empty role",
			fields: fields{
				Service:         newMockIAMClient(t, nil),
				DefaultKmsKeyID: "123",
			},
			args: args{
				ctx:    context.TODO(),
				role:   "",
				policy: "testPolicy",
			},
			wantErr: true,
		},
		{
			name: "empty policy",
			fields: fields{
				Service:         newMockIAMClient(t, nil),
				DefaultKmsKeyID: "123",
			},
			args: args{
				ctx:    context.TODO(),
				role:   "testRole",
				policy: "",
			},
			wantErr: true,
		},
		{
			name: "example policy and role",
			fields: fields{
				Service:         newMockIAMClient(t, nil),
				DefaultKmsKeyID: "123",
			},
			args: args{
				ctx:    context.TODO(),
				role:   "testRole",
				policy: "testPolicy",
			},
		},
		{
			name: "aws errror",
			fields: fields{
				Service:         newMockIAMClient(t, awserr.New(iam.ErrCodeInvalidInputException, "bad input", nil)),
				DefaultKmsKeyID: "123",
			},
			args: args{
				ctx:    context.TODO(),
				role:   "testRole",
				policy: "badPolicy",
			},
			wantErr: true,
		},
		{
			name: "non-aws errror",
			fields: fields{
				Service:         newMockIAMClient(t, errors.New("boom")),
				DefaultKmsKeyID: "123",
			},
			args: args{
				ctx:    context.TODO(),
				role:   "testRole",
				policy: "badPolicy",
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			i := &IAM{
				Service:         tt.fields.Service,
				DefaultKmsKeyID: tt.fields.DefaultKmsKeyID,
			}
			if err := i.DeleteRolePolicy(tt.args.ctx, tt.args.role, tt.args.policy); (err != nil) != tt.wantErr {
				t.Errorf("IAM.DeleteRolePolicy() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
