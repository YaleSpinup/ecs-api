package iam

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"testing"

	"github.com/YaleSpinup/ecs-api/apierror"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/request"
	"github.com/aws/aws-sdk-go/service/iam"
)

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
		PolicyStatement{
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
	if m.err != nil {
		return nil, m.err
	}
	return &iam.CreateRoleOutput{Role: &iam.Role{
		Arn:         aws.String(fmt.Sprintf("arn:aws:iam::12345678910:role/%s", *input.RoleName)),
		CreateDate:  &testTime,
		Description: input.Description,
		Path:        input.Path,
		RoleId:      aws.String(strings.ToUpper(fmt.Sprintf("%sID123", *input.RoleName))),
		RoleName:    input.RoleName,
	}}, nil
}

func (m *mockIAMClient) DeleteRoleWithContext(ctx context.Context, input *iam.DeleteRoleInput, opts ...request.Option) (*iam.DeleteRoleOutput, error) {
	if m.err != nil {
		return nil, m.err
	}
	return &iam.DeleteRoleOutput{}, nil
}

func (m *mockIAMClient) GetRoleWithContext(ctx context.Context, input *iam.GetRoleInput, opts ...request.Option) (*iam.GetRoleOutput, error) {
	if m.err != nil {
		return nil, m.err
	}
	return &iam.GetRoleOutput{Role: &testRole}, nil
}

func (m *mockIAMClient) PutRolePolicyWithContext(ctx context.Context, input *iam.PutRolePolicyInput, opts ...request.Option) (*iam.PutRolePolicyOutput, error) {
	if m.err != nil {
		return nil, m.err
	}
	return &iam.PutRolePolicyOutput{}, nil
}

func TestCreateRole(t *testing.T) {
	i := IAM{
		Service:         newMockIAMClient(t, nil),
		DefaultKmsKeyID: "12345678-90ab-cdef-1234-567890abcdef",
	}

	// build the default IAM task execution policy (from the config and known inputs)
	defaultPolicy, err := i.DefaultTaskExecutionPolicy("org/testCluster")
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
	out, err := i.GetRole(context.TODO(), &iam.GetRoleInput{RoleName: aws.String("testrole")})
	if err != nil {
		t.Errorf("expected nil error, got: %s", err)
	}

	if !reflect.DeepEqual(out, expected) {
		t.Errorf("expected %+v, got %+v", expected, out)
	}

	// test nil input
	_, err = i.GetRole(context.TODO(), nil)
	if aerr, ok := err.(apierror.Error); ok {
		if aerr.Code != apierror.ErrBadRequest {
			t.Errorf("expected error code %s, got: %s", apierror.ErrBadRequest, aerr.Code)
		}
	} else {
		t.Errorf("expected apierror.Error, got: %s", reflect.TypeOf(err).String())
	}

	// test empty role name
	_, err = i.GetRole(context.TODO(), &iam.GetRoleInput{})
	if aerr, ok := err.(apierror.Error); ok {
		if aerr.Code != apierror.ErrBadRequest {
			t.Errorf("expected error code %s, got: %s", apierror.ErrBadRequest, aerr.Code)
		}
	} else {
		t.Errorf("expected apierror.Error, got: %s", reflect.TypeOf(err).String())
	}

	// test ErrCodeNoSuchEntityException
	i.Service.(*mockIAMClient).err = awserr.New(iam.ErrCodeNoSuchEntityException, "NoSuchEntity", nil)
	_, err = i.GetRole(context.TODO(), &iam.GetRoleInput{RoleName: aws.String("testrole")})
	if aerr, ok := err.(apierror.Error); ok {
		if aerr.Code != apierror.ErrNotFound {
			t.Errorf("expected error code %s, got: %s", apierror.ErrNotFound, aerr.Code)
		}
	} else {
		t.Errorf("expected apierror.Error, got: %s", reflect.TypeOf(err).String())
	}

	// test ErrCodeServiceFailureException
	i.Service.(*mockIAMClient).err = awserr.New(iam.ErrCodeServiceFailureException, "ServiceFailure", nil)
	_, err = i.GetRole(context.TODO(), &iam.GetRoleInput{RoleName: aws.String("testrole")})
	if aerr, ok := err.(apierror.Error); ok {
		if aerr.Code != apierror.ErrServiceUnavailable {
			t.Errorf("expected error code %s, got: %s", apierror.ErrServiceUnavailable, aerr.Code)
		}
	} else {
		t.Errorf("expected apierror.Error, got: %s", reflect.TypeOf(err).String())
	}

	// test some other, unexpected AWS error
	i.Service.(*mockIAMClient).err = awserr.New("UnknownThingyBrokeYo", "ThingyBroke", nil)
	_, err = i.GetRole(context.TODO(), &iam.GetRoleInput{RoleName: aws.String("testrole")})
	if aerr, ok := err.(apierror.Error); ok {
		if aerr.Code != apierror.ErrBadRequest {
			t.Errorf("expected error code %s, got: %s", apierror.ErrBadRequest, aerr.Code)
		}
	} else {
		t.Errorf("expected apierror.Error, got: %s", reflect.TypeOf(err).String())
	}

	// test non-aws error
	i.Service.(*mockIAMClient).err = errors.New("things blowing up")
	_, err = i.GetRole(context.TODO(), &iam.GetRoleInput{RoleName: aws.String("testrole")})
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
