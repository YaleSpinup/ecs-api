package servicediscovery

import (
	"testing"

	"github.com/YaleSpinup/ecs-api/apierror"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/servicediscovery"
	"github.com/pkg/errors"
)

func TestErrCode(t *testing.T) {
	apiErrorTestCases := map[string]string{
		"": apierror.ErrBadRequest,

		servicediscovery.ErrCodeResourceLimitExceeded: apierror.ErrLimitExceeded,

		servicediscovery.ErrCodeNamespaceAlreadyExists: apierror.ErrConflict,
		servicediscovery.ErrCodeResourceInUse:          apierror.ErrConflict,
		servicediscovery.ErrCodeServiceAlreadyExists:   apierror.ErrConflict,

		servicediscovery.ErrCodeDuplicateRequest: apierror.ErrBadRequest,
		servicediscovery.ErrCodeInvalidInput:     apierror.ErrBadRequest,

		servicediscovery.ErrCodeCustomHealthNotFound: apierror.ErrNotFound,
		servicediscovery.ErrCodeInstanceNotFound:     apierror.ErrNotFound,
		servicediscovery.ErrCodeNamespaceNotFound:    apierror.ErrNotFound,
		servicediscovery.ErrCodeOperationNotFound:    apierror.ErrNotFound,
		servicediscovery.ErrCodeServiceNotFound:      apierror.ErrNotFound,
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
