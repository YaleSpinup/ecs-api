package secretsmanager

import (
	"testing"

	"github.com/YaleSpinup/ecs-api/apierror"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/secretsmanager"
	"github.com/pkg/errors"
)

func TestErrCode(t *testing.T) {
	apiErrorTestCases := map[string]string{
		"": apierror.ErrBadRequest,

		secretsmanager.ErrCodeInternalServiceError: apierror.ErrInternalError,

		secretsmanager.ErrCodeLimitExceededException:  apierror.ErrConflict,
		secretsmanager.ErrCodeResourceExistsException: apierror.ErrConflict,

		secretsmanager.ErrCodeDecryptionFailure:                apierror.ErrBadRequest,
		secretsmanager.ErrCodeEncryptionFailure:                apierror.ErrBadRequest,
		secretsmanager.ErrCodeInvalidNextTokenException:        apierror.ErrBadRequest,
		secretsmanager.ErrCodeInvalidParameterException:        apierror.ErrBadRequest,
		secretsmanager.ErrCodeInvalidRequestException:          apierror.ErrBadRequest,
		secretsmanager.ErrCodeMalformedPolicyDocumentException: apierror.ErrBadRequest,
		secretsmanager.ErrCodePreconditionNotMetException:      apierror.ErrBadRequest,

		secretsmanager.ErrCodeResourceNotFoundException: apierror.ErrNotFound,
	}

	for awsErr, apiErr := range apiErrorTestCases {
		err := ErrCode("test error", awserr.New(awsErr, awsErr, nil))
		if aerr, ok := errors.Cause(err).(apierror.Error); ok {
			t.Logf("got apierror '%s'", aerr)
		} else {
			t.Errorf("expected cloudwatch error %s to be an apierror.Error %s, got %s", awsErr, apiErr, err)
		}
	}

	err := ErrCode("test error", errors.New("Unknown"))
	if aerr, ok := errors.Cause(err).(apierror.Error); ok {
		t.Logf("got apierror '%s'", aerr)
	} else {
		t.Errorf("expected unknown error to be an apierror.ErrInternalError, got %s", err)
	}
}
