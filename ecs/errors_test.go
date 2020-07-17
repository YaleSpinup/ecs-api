package ecs

import (
	"testing"

	"github.com/YaleSpinup/apierror"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/ecs"
	"github.com/pkg/errors"
)

func TestErrCode(t *testing.T) {
	apiErrorTestCases := map[string]string{
		"": apierror.ErrBadRequest,

		ecs.ErrCodeAccessDeniedException: apierror.ErrForbidden,
		ecs.ErrCodeBlockedException:      apierror.ErrForbidden,

		ecs.ErrCodeServerException: apierror.ErrInternalError,

		ecs.ErrCodeUpdateInProgressException:                  apierror.ErrConflict,
		ecs.ErrCodeClusterContainsContainerInstancesException: apierror.ErrConflict,
		ecs.ErrCodeClusterContainsServicesException:           apierror.ErrConflict,
		ecs.ErrCodeClusterContainsTasksException:              apierror.ErrConflict,
		ecs.ErrCodeResourceInUseException:                     apierror.ErrConflict,

		ecs.ErrCodeClientException:                                apierror.ErrBadRequest,
		ecs.ErrCodeInvalidParameterException:                      apierror.ErrBadRequest,
		ecs.ErrCodeMissingVersionException:                        apierror.ErrBadRequest,
		ecs.ErrCodeNoUpdateAvailableException:                     apierror.ErrBadRequest,
		ecs.ErrCodePlatformTaskDefinitionIncompatibilityException: apierror.ErrBadRequest,
		ecs.ErrCodePlatformUnknownException:                       apierror.ErrBadRequest,
		ecs.ErrCodeServiceNotActiveException:                      apierror.ErrBadRequest,
		ecs.ErrCodeUnsupportedFeatureException:                    apierror.ErrBadRequest,

		ecs.ErrCodeClusterNotFoundException:  apierror.ErrNotFound,
		ecs.ErrCodeResourceNotFoundException: apierror.ErrNotFound,
		ecs.ErrCodeServiceNotFoundException:  apierror.ErrNotFound,
		ecs.ErrCodeTargetNotFoundException:   apierror.ErrNotFound,
		ecs.ErrCodeTaskSetNotFoundException:  apierror.ErrNotFound,

		ecs.ErrCodeAttributeLimitExceededException: apierror.ErrLimitExceeded,
		ecs.ErrCodeLimitExceededException:          apierror.ErrLimitExceeded,
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
