package elbv2

import (
	"testing"

	"github.com/YaleSpinup/ecs-api/apierror"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/elbv2"
	"github.com/pkg/errors"
)

func TestErrCode(t *testing.T) {
	apiErrorTestCases := map[string]string{
		"": apierror.ErrBadRequest,
		elbv2.ErrCodeOperationNotPermittedException:                    apierror.ErrForbidden,
		elbv2.ErrCodeTooManyActionsException:                           apierror.ErrLimitExceeded,
		elbv2.ErrCodeTooManyCertificatesException:                      apierror.ErrLimitExceeded,
		elbv2.ErrCodeTooManyListenersException:                         apierror.ErrLimitExceeded,
		elbv2.ErrCodeTooManyLoadBalancersException:                     apierror.ErrLimitExceeded,
		elbv2.ErrCodeTooManyRegistrationsForTargetIdException:          apierror.ErrLimitExceeded,
		elbv2.ErrCodeTooManyRulesException:                             apierror.ErrLimitExceeded,
		elbv2.ErrCodeTooManyTagsException:                              apierror.ErrLimitExceeded,
		elbv2.ErrCodeTooManyTargetGroupsException:                      apierror.ErrLimitExceeded,
		elbv2.ErrCodeTooManyTargetsException:                           apierror.ErrLimitExceeded,
		elbv2.ErrCodeTooManyUniqueTargetGroupsPerLoadBalancerException: apierror.ErrLimitExceeded,
		elbv2.ErrCodeDuplicateListenerException:                        apierror.ErrConflict,
		elbv2.ErrCodeDuplicateLoadBalancerNameException:                apierror.ErrConflict,
		elbv2.ErrCodeDuplicateTagKeysException:                         apierror.ErrConflict,
		elbv2.ErrCodeDuplicateTargetGroupNameException:                 apierror.ErrConflict,
		elbv2.ErrCodePriorityInUseException:                            apierror.ErrConflict,
		elbv2.ErrCodeResourceInUseException:                            apierror.ErrConflict,
		elbv2.ErrCodeTargetGroupAssociationLimitException:              apierror.ErrConflict,
		elbv2.ErrCodeAvailabilityZoneNotSupportedException:             apierror.ErrBadRequest,
		elbv2.ErrCodeCertificateNotFoundException:                      apierror.ErrBadRequest,
		elbv2.ErrCodeHealthUnavailableException:                        apierror.ErrBadRequest,
		elbv2.ErrCodeIncompatibleProtocolsException:                    apierror.ErrBadRequest,
		elbv2.ErrCodeInvalidConfigurationRequestException:              apierror.ErrBadRequest,
		elbv2.ErrCodeInvalidLoadBalancerActionException:                apierror.ErrBadRequest,
		elbv2.ErrCodeInvalidSchemeException:                            apierror.ErrBadRequest,
		elbv2.ErrCodeInvalidSecurityGroupException:                     apierror.ErrBadRequest,
		elbv2.ErrCodeInvalidSubnetException:                            apierror.ErrBadRequest,
		elbv2.ErrCodeInvalidTargetException:                            apierror.ErrBadRequest,
		elbv2.ErrCodeUnsupportedProtocolException:                      apierror.ErrBadRequest,
		elbv2.ErrCodeListenerNotFoundException:                         apierror.ErrNotFound,
		elbv2.ErrCodeLoadBalancerNotFoundException:                     apierror.ErrNotFound,
		elbv2.ErrCodeAllocationIdNotFoundException:                     apierror.ErrNotFound,
		elbv2.ErrCodeRuleNotFoundException:                             apierror.ErrNotFound,
		elbv2.ErrCodeSSLPolicyNotFoundException:                        apierror.ErrNotFound,
		elbv2.ErrCodeSubnetNotFoundException:                           apierror.ErrNotFound,
		elbv2.ErrCodeTargetGroupNotFoundException:                      apierror.ErrNotFound,
	}

	for awsErr, apiErr := range apiErrorTestCases {
		err := ErrCode("test error", awserr.New(awsErr, awsErr, nil))
		if aerr, ok := errors.Cause(err).(apierror.Error); ok {
			t.Logf("got apierror '%s'", aerr)
		} else {
			t.Errorf("expected elbv2 error %s to be an apierror.Error %s, got %s", awsErr, apiErr, err)
		}
	}

	err := ErrCode("test error", errors.New("Unknown"))
	if aerr, ok := errors.Cause(err).(apierror.Error); ok {
		t.Logf("got apierror '%s'", aerr)
	} else {
		t.Errorf("expected unknown error to be an apierror.ErrInternalError, got %s", err)
	}
}
