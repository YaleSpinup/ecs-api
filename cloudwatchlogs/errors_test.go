package cloudwatchlogs

import (
	"testing"

	"github.com/YaleSpinup/ecs-api/apierror"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/cloudwatchlogs"
	"github.com/pkg/errors"
)

func TestErrCode(t *testing.T) {
	apiErrorTestCases := map[string]string{
		"": apierror.ErrBadRequest,

		cloudwatchlogs.ErrCodeUnrecognizedClientException: apierror.ErrForbidden,
		cloudwatchlogs.ErrCodeServiceUnavailableException: apierror.ErrInternalError,

		cloudwatchlogs.ErrCodeDataAlreadyAcceptedException:   apierror.ErrConflict,
		cloudwatchlogs.ErrCodeOperationAbortedException:      apierror.ErrConflict,
		cloudwatchlogs.ErrCodeResourceAlreadyExistsException: apierror.ErrConflict,

		cloudwatchlogs.ErrCodeInvalidOperationException:     apierror.ErrBadRequest,
		cloudwatchlogs.ErrCodeInvalidParameterException:     apierror.ErrBadRequest,
		cloudwatchlogs.ErrCodeInvalidSequenceTokenException: apierror.ErrBadRequest,
		cloudwatchlogs.ErrCodeMalformedQueryException:       apierror.ErrBadRequest,

		cloudwatchlogs.ErrCodeResourceNotFoundException: apierror.ErrNotFound,

		cloudwatchlogs.ErrCodeLimitExceededException: apierror.ErrLimitExceeded,
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
