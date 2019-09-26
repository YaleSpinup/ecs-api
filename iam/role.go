package iam

import (
	"context"
	"fmt"

	"github.com/YaleSpinup/ecs-api/apierror"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/iam"
	log "github.com/sirupsen/logrus"
)

// CreateRole handles creating IAM role
func (i *IAM) CreateRole(ctx context.Context, input *iam.CreateRoleInput) (*iam.CreateRoleOutput, error) {
	if input == nil {
		return nil, apierror.New(apierror.ErrBadRequest, "invalid input", nil)
	}

	log.Infof("creating iam role: %s", *input.RoleName)

	output, err := i.Service.CreateRoleWithContext(ctx, input)
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			// * ErrCodeInvalidInputException "InvalidInput"
			// The request was rejected because an invalid or out-of-range value was supplied
			// for an input parameter.
			// * ErrCodeMalformedPolicyDocumentException "MalformedPolicyDocument"
			// The request was rejected because the policy document was malformed. The error
			// message describes the specific error.
			case iam.ErrCodeInvalidInputException, iam.ErrCodeMalformedPolicyDocumentException:
				msg := fmt.Sprintf("%s: %s", aerr.Code(), aerr.Error())
				return nil, apierror.New(apierror.ErrBadRequest, msg, err)
			// * ErrCodeLimitExceededException "LimitExceeded"
			// The request was rejected because it attempted to create resources beyond
			// the current AWS account limits. The error message describes the limit exceeded.
			case iam.ErrCodeLimitExceededException:
				msg := fmt.Sprintf("%s: %s", aerr.Code(), aerr.Error())
				return nil, apierror.New(apierror.ErrLimitExceeded, msg, err)
			// * ErrCodeEntityAlreadyExistsException "EntityAlreadyExists"
			// The request was rejected because it attempted to create a resource that already
			// exists.
			case iam.ErrCodeEntityAlreadyExistsException:
				msg := fmt.Sprintf("%s: %s", aerr.Code(), aerr.Error())
				return nil, apierror.New(apierror.ErrConflict, msg, err)
			// * ErrCodeServiceFailureException "ServiceFailure"
			// The request processing has failed because of an unknown error, exception
			// or failure.
			case iam.ErrCodeServiceFailureException:
				msg := fmt.Sprintf("%s: %s", aerr.Code(), aerr.Error())
				return nil, apierror.New(apierror.ErrServiceUnavailable, msg, err)
			default:
				return nil, apierror.New(apierror.ErrBadRequest, aerr.Message(), err)
			}
		}

		return nil, apierror.New(apierror.ErrInternalError, "unknown error occurred", err)
	}

	return output, nil
}

// DeleteRole handles deleting IAM role
func (i *IAM) DeleteRole(ctx context.Context, input *iam.DeleteRoleInput) (*iam.DeleteRoleOutput, error) {
	if input == nil || aws.StringValue(input.RoleName) == "" {
		return nil, apierror.New(apierror.ErrBadRequest, "invalid input", nil)
	}

	log.Infof("deleting iam role %s", aws.StringValue(input.RoleName))

	output, err := i.Service.DeleteRoleWithContext(ctx, input)
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			// * ErrCodeNoSuchEntityException "NoSuchEntity"
			// The request was rejected because it referenced a resource entity that does
			// not exist. The error message describes the resource.
			case iam.ErrCodeNoSuchEntityException:
				msg := fmt.Sprintf("%s: %s", aerr.Code(), aerr.Error())
				return nil, apierror.New(apierror.ErrNotFound, msg, err)
			// * ErrCodeLimitExceededException "LimitExceeded"
			// The request was rejected because it attempted to create resources beyond
			// the current AWS account limits. The error message describes the limit exceeded.
			case iam.ErrCodeLimitExceededException:
				msg := fmt.Sprintf("%s: %s", aerr.Code(), aerr.Error())
				return nil, apierror.New(apierror.ErrLimitExceeded, msg, err)
			// * ErrCodeInvalidInputException "InvalidInput"
			// The request was rejected because an invalid or out-of-range value was supplied
			// for an input parameter.
			case iam.ErrCodeInvalidInputException:
				msg := fmt.Sprintf("%s: %s", aerr.Code(), aerr.Error())
				return nil, apierror.New(apierror.ErrBadRequest, msg, err)
			// * ErrCodeDeleteConflictException "DeleteConflict"
			// The request was rejected because it attempted to delete a resource that has
			// attached subordinate entities. The error message describes these entities.
			case iam.ErrCodeDeleteConflictException:
				msg := fmt.Sprintf("%s: %s", aerr.Code(), aerr.Error())
				return nil, apierror.New(apierror.ErrConflict, msg, err)
			// * ErrCodeServiceFailureException "ServiceFailure"
			// The request processing has failed because of an unknown error, exception
			// or failure.
			case iam.ErrCodeServiceFailureException:
				msg := fmt.Sprintf("%s: %s", aerr.Code(), aerr.Error())
				return nil, apierror.New(apierror.ErrServiceUnavailable, msg, err)
			default:
				return nil, apierror.New(apierror.ErrBadRequest, aerr.Message(), err)
			}
		}
		return nil, apierror.New(apierror.ErrInternalError, "unknown error occurred", err)
	}

	return output, nil
}

// PutRolePolicy handles attaching an inline policy to IAM role
func (i *IAM) PutRolePolicy(ctx context.Context, input *iam.PutRolePolicyInput) (*iam.PutRolePolicyOutput, error) {
	if input == nil {
		return nil, apierror.New(apierror.ErrBadRequest, "invalid input", nil)
	}

	log.Infof("attaching inline policy to iam role: %s", *input.RoleName)

	output, err := i.Service.PutRolePolicyWithContext(ctx, input)
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			// * ErrCodeInvalidInputException "InvalidInput"
			// The request was rejected because an invalid or out-of-range value was supplied
			// for an input parameter.
			// * ErrCodeMalformedPolicyDocumentException "MalformedPolicyDocument"
			// The request was rejected because the policy document was malformed. The error
			// message describes the specific error.
			case iam.ErrCodeInvalidInputException, iam.ErrCodeMalformedPolicyDocumentException:
				msg := fmt.Sprintf("%s: %s", aerr.Code(), aerr.Error())
				return nil, apierror.New(apierror.ErrBadRequest, msg, err)
			// * ErrCodeLimitExceededException "LimitExceeded"
			// The request was rejected because it attempted to create resources beyond
			// the current AWS account limits. The error message describes the limit exceeded.
			case iam.ErrCodeLimitExceededException:
				msg := fmt.Sprintf("%s: %s", aerr.Code(), aerr.Error())
				return nil, apierror.New(apierror.ErrLimitExceeded, msg, err)
			// * ErrCodeNoSuchEntityException "NoSuchEntityException"
			// The request was rejected because it referenced an entity that does not exist.
			// The error code describes the entity.
			case iam.ErrCodeNoSuchEntityException:
				msg := fmt.Sprintf("%s: %s", aerr.Code(), aerr.Error())
				return nil, apierror.New(apierror.ErrNotFound, msg, err)
			// * ErrCodeServiceFailureException "ServiceFailure"
			// The request processing has failed because of an unknown error, exception
			// or failure.
			case iam.ErrCodeServiceFailureException:
				msg := fmt.Sprintf("%s: %s", aerr.Code(), aerr.Error())
				return nil, apierror.New(apierror.ErrServiceUnavailable, msg, err)
			default:
				return nil, apierror.New(apierror.ErrBadRequest, aerr.Message(), err)
			}
		}

		return nil, apierror.New(apierror.ErrInternalError, "unknown error occurred", err)
	}

	return output, nil
}
