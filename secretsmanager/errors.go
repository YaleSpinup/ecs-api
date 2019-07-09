package secretsmanager

import (
	"github.com/YaleSpinup/s3-api/apierror"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/secretsmanager"
	"github.com/pkg/errors"
)

func ErrCode(msg string, err error) error {
	if aerr, ok := errors.Cause(err).(awserr.Error); ok {
		switch aerr.Code() {
		case

			// ErrCodeInternalServiceError for service response error code
			// "InternalServiceError".
			//
			// An error occurred on the server side.
			secretsmanager.ErrCodeInternalServiceError:
			return apierror.New(apierror.ErrInternalError, msg, err)
		case
			// ErrCodeLimitExceededException for service response error code
			// "LimitExceededException".
			//
			// The request failed because it would exceed one of the Secrets Manager internal
			// limits.
			secretsmanager.ErrCodeLimitExceededException,

			// ErrCodeResourceExistsException for service response error code
			// "ResourceExistsException".
			//
			// A resource with the ID you requested already exists.
			secretsmanager.ErrCodeResourceExistsException:
			// return a conflict
			return apierror.New(apierror.ErrConflict, msg, aerr)
		case
			// ErrCodeDecryptionFailure for service response error code
			// "DecryptionFailure".
			//
			// Secrets Manager can't decrypt the protected secret text using the provided
			// KMS key.
			secretsmanager.ErrCodeDecryptionFailure,

			// ErrCodeEncryptionFailure for service response error code
			// "EncryptionFailure".
			//
			// Secrets Manager can't encrypt the protected secret text using the provided
			// KMS key. Check that the customer master key (CMK) is available, enabled,
			// and not in an invalid state. For more information, see How Key State Affects
			// Use of a Customer Master Key (http://docs.aws.amazon.com/kms/latest/developerguide/key-state.html).
			secretsmanager.ErrCodeEncryptionFailure,

			// ErrCodeInvalidNextTokenException for service response error code
			// "InvalidNextTokenException".
			//
			// You provided an invalid NextToken value.
			secretsmanager.ErrCodeInvalidNextTokenException,

			// ErrCodeInvalidParameterException for service response error code
			// "InvalidParameterException".
			//
			// You provided an invalid value for a parameter.
			secretsmanager.ErrCodeInvalidParameterException,

			// ErrCodeInvalidRequestException for service response error code
			// "InvalidRequestException".
			//
			// You provided a parameter value that is not valid for the current state of
			// the resource.
			//
			// Possible causes:
			//
			//    * You tried to perform the operation on a secret that's currently marked
			//    deleted.
			//
			//    * You tried to enable rotation on a secret that doesn't already have a
			//    Lambda function ARN configured and you didn't include such an ARN as a
			//    parameter in this call.
			secretsmanager.ErrCodeInvalidRequestException,

			// ErrCodeMalformedPolicyDocumentException for service response error code
			// "MalformedPolicyDocumentException".
			//
			// The policy document that you provided isn't valid.
			secretsmanager.ErrCodeMalformedPolicyDocumentException,

			// ErrCodePreconditionNotMetException for service response error code
			// "PreconditionNotMetException".
			//
			// The request failed because you did not complete all the prerequisite steps.
			secretsmanager.ErrCodePreconditionNotMetException:
			return apierror.New(apierror.ErrBadRequest, msg, aerr)
		case
			// ErrCodeResourceNotFoundException for service response error code
			// "ResourceNotFoundException".
			//
			// We can't find the resource that you asked for.
			secretsmanager.ErrCodeResourceNotFoundException:
			return apierror.New(apierror.ErrNotFound, msg, aerr)
		default:
			m := msg + ": " + aerr.Message()
			return apierror.New(apierror.ErrBadRequest, m, aerr)
		}
	}

	return apierror.New(apierror.ErrInternalError, msg, err)
}
