package ssm

import (
	"testing"

	"github.com/YaleSpinup/apierror"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/ssm"
	"github.com/pkg/errors"
)

func TestErrCode(t *testing.T) {
	apiErrorTestCases := map[string]string{
		"": apierror.ErrBadRequest,

		ssm.ErrCodeInternalServerError: apierror.ErrInternalError,

		ssm.ErrCodeAlreadyExistsException:                 apierror.ErrConflict,
		ssm.ErrCodeAssociationAlreadyExists:               apierror.ErrConflict,
		ssm.ErrCodeDocumentAlreadyExists:                  apierror.ErrConflict,
		ssm.ErrCodeDuplicateDocumentContent:               apierror.ErrConflict,
		ssm.ErrCodeDuplicateDocumentVersionName:           apierror.ErrConflict,
		ssm.ErrCodeDuplicateInstanceId:                    apierror.ErrConflict,
		ssm.ErrCodeOpsItemAlreadyExistsException:          apierror.ErrConflict,
		ssm.ErrCodeParameterAlreadyExists:                 apierror.ErrConflict,
		ssm.ErrCodeResourceDataSyncAlreadyExistsException: apierror.ErrConflict,
		ssm.ErrCodeResourceInUseException:                 apierror.ErrConflict,
		ssm.ErrCodeTargetInUseException:                   apierror.ErrConflict,

		ssm.ErrCodeAssociatedInstances:                           apierror.ErrBadRequest,
		ssm.ErrCodeFeatureNotAvailableException:                  apierror.ErrBadRequest,
		ssm.ErrCodeHierarchyTypeMismatchException:                apierror.ErrBadRequest,
		ssm.ErrCodeIdempotentParameterMismatch:                   apierror.ErrBadRequest,
		ssm.ErrCodeIncompatiblePolicyException:                   apierror.ErrBadRequest,
		ssm.ErrCodeInvalidActivation:                             apierror.ErrBadRequest,
		ssm.ErrCodeInvalidActivationId:                           apierror.ErrBadRequest,
		ssm.ErrCodeInvalidAggregatorException:                    apierror.ErrBadRequest,
		ssm.ErrCodeInvalidAllowedPatternException:                apierror.ErrBadRequest,
		ssm.ErrCodeInvalidAssociation:                            apierror.ErrBadRequest,
		ssm.ErrCodeInvalidAssociationVersion:                     apierror.ErrBadRequest,
		ssm.ErrCodeInvalidAutomationExecutionParametersException: apierror.ErrBadRequest,
		ssm.ErrCodeInvalidAutomationSignalException:              apierror.ErrBadRequest,
		ssm.ErrCodeInvalidAutomationStatusUpdateException:        apierror.ErrBadRequest,
		ssm.ErrCodeInvalidCommandId:                              apierror.ErrBadRequest,
		ssm.ErrCodeInvalidDeleteInventoryParametersException:     apierror.ErrBadRequest,
		ssm.ErrCodeInvalidDeletionIdException:                    apierror.ErrBadRequest,
		ssm.ErrCodeInvalidDocument:                               apierror.ErrBadRequest,
		ssm.ErrCodeInvalidDocumentContent:                        apierror.ErrBadRequest,
		ssm.ErrCodeInvalidDocumentOperation:                      apierror.ErrBadRequest,
		ssm.ErrCodeInvalidDocumentSchemaVersion:                  apierror.ErrBadRequest,
		ssm.ErrCodeInvalidDocumentVersion:                        apierror.ErrBadRequest,
		ssm.ErrCodeInvalidFilter:                                 apierror.ErrBadRequest,
		ssm.ErrCodeInvalidFilterKey:                              apierror.ErrBadRequest,
		ssm.ErrCodeInvalidFilterOption:                           apierror.ErrBadRequest,
		ssm.ErrCodeInvalidFilterValue:                            apierror.ErrBadRequest,
		ssm.ErrCodeInvalidInstanceId:                             apierror.ErrBadRequest,
		ssm.ErrCodeInvalidInstanceInformationFilterValue:         apierror.ErrBadRequest,
		ssm.ErrCodeInvalidInventoryGroupException:                apierror.ErrBadRequest,
		ssm.ErrCodeInvalidInventoryItemContextException:          apierror.ErrBadRequest,
		ssm.ErrCodeInvalidInventoryRequestException:              apierror.ErrBadRequest,
		ssm.ErrCodeInvalidItemContentException:                   apierror.ErrBadRequest,
		ssm.ErrCodeInvalidKeyId:                                  apierror.ErrBadRequest,
		ssm.ErrCodeInvalidNextToken:                              apierror.ErrBadRequest,
		ssm.ErrCodeInvalidNotificationConfig:                     apierror.ErrBadRequest,
		ssm.ErrCodeInvalidOptionException:                        apierror.ErrBadRequest,
		ssm.ErrCodeInvalidOutputFolder:                           apierror.ErrBadRequest,
		ssm.ErrCodeInvalidOutputLocation:                         apierror.ErrBadRequest,
		ssm.ErrCodeInvalidParameters:                             apierror.ErrBadRequest,
		ssm.ErrCodeInvalidPermissionType:                         apierror.ErrBadRequest,
		ssm.ErrCodeInvalidPluginName:                             apierror.ErrBadRequest,
		ssm.ErrCodeInvalidPolicyAttributeException:               apierror.ErrBadRequest,
		ssm.ErrCodeInvalidPolicyTypeException:                    apierror.ErrBadRequest,
		ssm.ErrCodeInvalidResourceId:                             apierror.ErrBadRequest,
		ssm.ErrCodeInvalidResourceType:                           apierror.ErrBadRequest,
		ssm.ErrCodeInvalidResultAttributeException:               apierror.ErrBadRequest,
		ssm.ErrCodeInvalidRole:                                   apierror.ErrBadRequest,
		ssm.ErrCodeInvalidSchedule:                               apierror.ErrBadRequest,
		ssm.ErrCodeInvalidTarget:                                 apierror.ErrBadRequest,
		ssm.ErrCodeInvalidTypeNameException:                      apierror.ErrBadRequest,
		ssm.ErrCodeInvalidUpdate:                                 apierror.ErrBadRequest,
		ssm.ErrCodeItemContentMismatchException:                  apierror.ErrBadRequest,
		ssm.ErrCodeOpsItemInvalidParameterException:              apierror.ErrBadRequest,
		ssm.ErrCodeParameterMaxVersionLimitExceeded:              apierror.ErrBadRequest,
		ssm.ErrCodeParameterPatternMismatchException:             apierror.ErrBadRequest,
		ssm.ErrCodeResourceDataSyncInvalidConfigurationException: apierror.ErrBadRequest,
		ssm.ErrCodeStatusUnchanged:                               apierror.ErrBadRequest,
		ssm.ErrCodeTargetNotConnected:                            apierror.ErrBadRequest,
		ssm.ErrCodeUnsupportedFeatureRequiredException:           apierror.ErrBadRequest,
		ssm.ErrCodeUnsupportedInventoryItemContextException:      apierror.ErrBadRequest,
		ssm.ErrCodeUnsupportedInventorySchemaVersionException:    apierror.ErrBadRequest,
		ssm.ErrCodeUnsupportedOperatingSystem:                    apierror.ErrBadRequest,
		ssm.ErrCodeUnsupportedParameterType:                      apierror.ErrBadRequest,
		ssm.ErrCodeUnsupportedPlatformType:                       apierror.ErrBadRequest,

		ssm.ErrCodeInvocationDoesNotExist:                       apierror.ErrNotFound,
		ssm.ErrCodeOpsItemNotFoundException:                     apierror.ErrNotFound,
		ssm.ErrCodeParameterNotFound:                            apierror.ErrNotFound,
		ssm.ErrCodeParameterVersionNotFound:                     apierror.ErrNotFound,
		ssm.ErrCodeResourceDataSyncNotFoundException:            apierror.ErrNotFound,
		ssm.ErrCodeServiceSettingNotFound:                       apierror.ErrNotFound,
		ssm.ErrCodeAssociationDoesNotExist:                      apierror.ErrNotFound,
		ssm.ErrCodeAssociationExecutionDoesNotExist:             apierror.ErrNotFound,
		ssm.ErrCodeAutomationDefinitionNotFoundException:        apierror.ErrNotFound,
		ssm.ErrCodeAutomationDefinitionVersionNotFoundException: apierror.ErrNotFound,
		ssm.ErrCodeAutomationExecutionNotFoundException:         apierror.ErrNotFound,
		ssm.ErrCodeAutomationStepNotFoundException:              apierror.ErrNotFound,
		ssm.ErrCodeDoesNotExistException:                        apierror.ErrNotFound,

		ssm.ErrCodeAutomationExecutionLimitExceededException: apierror.ErrLimitExceeded,
		ssm.ErrCodeComplianceTypeCountLimitExceededException: apierror.ErrLimitExceeded,
		ssm.ErrCodeCustomSchemaCountLimitExceededException:   apierror.ErrLimitExceeded,
		ssm.ErrCodeDocumentLimitExceeded:                     apierror.ErrLimitExceeded,
		ssm.ErrCodeDocumentPermissionLimit:                   apierror.ErrLimitExceeded,
		ssm.ErrCodeDocumentVersionLimitExceeded:              apierror.ErrLimitExceeded,
		ssm.ErrCodeHierarchyLevelLimitExceededException:      apierror.ErrLimitExceeded,
		ssm.ErrCodeItemSizeLimitExceededException:            apierror.ErrLimitExceeded,
		ssm.ErrCodeMaxDocumentSizeExceeded:                   apierror.ErrLimitExceeded,
		ssm.ErrCodeOpsItemLimitExceededException:             apierror.ErrLimitExceeded,
		ssm.ErrCodeParameterLimitExceeded:                    apierror.ErrLimitExceeded,
		ssm.ErrCodeParameterVersionLabelLimitExceeded:        apierror.ErrLimitExceeded,
		ssm.ErrCodePoliciesLimitExceededException:            apierror.ErrLimitExceeded,
		ssm.ErrCodeResourceDataSyncCountExceededException:    apierror.ErrLimitExceeded,
		ssm.ErrCodeResourceLimitExceededException:            apierror.ErrLimitExceeded,
		ssm.ErrCodeSubTypeCountLimitExceededException:        apierror.ErrLimitExceeded,
		ssm.ErrCodeAssociationLimitExceeded:                  apierror.ErrLimitExceeded,
		ssm.ErrCodeAssociationVersionLimitExceeded:           apierror.ErrLimitExceeded,
		ssm.ErrCodeTooManyTagsError:                          apierror.ErrLimitExceeded,
		ssm.ErrCodeTooManyUpdates:                            apierror.ErrLimitExceeded,
		ssm.ErrCodeTotalSizeLimitExceededException:           apierror.ErrLimitExceeded,
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
