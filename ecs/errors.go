package ecs

import (
	"github.com/YaleSpinup/ecs-api/apierror"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/ecs"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
)

func ErrCode(msg string, err error) error {
	log.Debugf("processing error code with message '%s' and error '%s'", msg, err)

	if aerr, ok := errors.Cause(err).(awserr.Error); ok {
		switch aerr.Code() {
		case
			// ErrCodeAccessDeniedException for service response error code
			// "AccessDeniedException".
			//
			// You do not have authorization to perform the requested action.
			ecs.ErrCodeAccessDeniedException,

			// ErrCodeBlockedException for service response error code
			// "BlockedException".
			//
			// Your AWS account has been blocked. For more information, contact AWS Support
			// (http://aws.amazon.com/contact-us/).
			ecs.ErrCodeBlockedException:

			return apierror.New(apierror.ErrForbidden, msg, aerr)
		case
			// ErrCodeServerException for service response error code
			// "ServerException".
			//
			// These errors are usually caused by a server issue.
			ecs.ErrCodeServerException:

			return apierror.New(apierror.ErrInternalError, msg, err)
		case
			// ErrCodeUpdateInProgressException for service response error code
			// "UpdateInProgressException".
			//
			// There is already a current Amazon ECS container agent update in progress
			// on the specified container instance. If the container agent becomes disconnected
			// while it is in a transitional stage, such as PENDING or STAGING, the update
			// process can get stuck in that state. However, when the agent reconnects,
			// it resumes where it stopped previously.
			ecs.ErrCodeUpdateInProgressException,

			// ErrCodeClusterContainsContainerInstancesException for service response error code
			// "ClusterContainsContainerInstancesException".
			//
			// You cannot delete a cluster that has registered container instances. First,
			// deregister the container instances before you can delete the cluster. For
			// more information, see DeregisterContainerInstance.
			ecs.ErrCodeClusterContainsContainerInstancesException,

			// ErrCodeClusterContainsServicesException for service response error code
			// "ClusterContainsServicesException".
			//
			// You cannot delete a cluster that contains services. First, update the service
			// to reduce its desired task count to 0 and then delete the service. For more
			// information, see UpdateService and DeleteService.
			ecs.ErrCodeClusterContainsServicesException,

			// ErrCodeClusterContainsTasksException for service response error code
			// "ClusterContainsTasksException".
			//
			// You cannot delete a cluster that has active tasks.
			ecs.ErrCodeClusterContainsTasksException,

			// ErrCodeResourceInUseException for service response error code
			// "ResourceInUseException".
			//
			// The specified resource is in-use and cannot be removed.
			ecs.ErrCodeResourceInUseException:

			return apierror.New(apierror.ErrConflict, msg, aerr)
		case
			// ErrCodeClientException for service response error code
			// "ClientException".
			//
			// These errors are usually caused by a client action, such as using an action
			// or resource on behalf of a user that doesn't have permissions to use the
			// action or resource, or specifying an identifier that is not valid.
			ecs.ErrCodeClientException,

			// ErrCodeInvalidParameterException for service response error code
			// "InvalidParameterException".
			//
			// The specified parameter is invalid. Review the available parameters for the
			// API request.
			ecs.ErrCodeInvalidParameterException,

			// ErrCodeMissingVersionException for service response error code
			// "MissingVersionException".
			//
			// Amazon ECS is unable to determine the current version of the Amazon ECS container
			// agent on the container instance and does not have enough information to proceed
			// with an update. This could be because the agent running on the container
			// instance is an older or custom version that does not use our version information.
			ecs.ErrCodeMissingVersionException,

			// ErrCodeNoUpdateAvailableException for service response error code
			// "NoUpdateAvailableException".
			//
			// There is no update available for this Amazon ECS container agent. This could
			// be because the agent is already running the latest version, or it is so old
			// that there is no update path to the current version.
			ecs.ErrCodeNoUpdateAvailableException,

			// ErrCodePlatformTaskDefinitionIncompatibilityException for service response error code
			// "PlatformTaskDefinitionIncompatibilityException".
			//
			// The specified platform version does not satisfy the task definition's required
			// capabilities.
			ecs.ErrCodePlatformTaskDefinitionIncompatibilityException,

			// ErrCodePlatformUnknownException for service response error code
			// "PlatformUnknownException".
			//
			// The specified platform version does not exist.
			ecs.ErrCodePlatformUnknownException,

			// ErrCodeServiceNotActiveException for service response error code
			// "ServiceNotActiveException".
			//
			// The specified service is not active. You can't update a service that is inactive.
			// If you have previously deleted a service, you can re-create it with CreateService.
			ecs.ErrCodeServiceNotActiveException,

			// ErrCodeUnsupportedFeatureException for service response error code
			// "UnsupportedFeatureException".
			//
			// The specified task is not supported in this Region.
			ecs.ErrCodeUnsupportedFeatureException:

			return apierror.New(apierror.ErrBadRequest, msg, aerr)
		case
			// ErrCodeClusterNotFoundException for service response error code
			// "ClusterNotFoundException".
			//
			// The specified cluster could not be found. You can view your available clusters
			// with ListClusters. Amazon ECS clusters are Region-specific.
			ecs.ErrCodeClusterNotFoundException,

			// ErrCodeResourceNotFoundException for service response error code
			// "ResourceNotFoundException".
			//
			// The specified resource could not be found.
			ecs.ErrCodeResourceNotFoundException,

			// ErrCodeServiceNotFoundException for service response error code
			// "ServiceNotFoundException".
			//
			// The specified service could not be found. You can view your available services
			// with ListServices. Amazon ECS services are cluster-specific and Region-specific.
			ecs.ErrCodeServiceNotFoundException,

			// ErrCodeTargetNotFoundException for service response error code
			// "TargetNotFoundException".
			//
			// The specified target could not be found. You can view your available container
			// instances with ListContainerInstances. Amazon ECS container instances are
			// cluster-specific and Region-specific.
			ecs.ErrCodeTargetNotFoundException,

			// ErrCodeTaskSetNotFoundException for service response error code
			// "TaskSetNotFoundException".
			//
			// The specified task set could not be found. You can view your available task
			// sets with DescribeTaskSets. Task sets are specific to each cluster, service
			// and Region.
			ecs.ErrCodeTaskSetNotFoundException:

			return apierror.New(apierror.ErrNotFound, msg, aerr)
		case
			// ErrCodeAttributeLimitExceededException for service response error code
			// "AttributeLimitExceededException".
			//
			// You can apply up to 10 custom attributes per resource. You can view the attributes
			// of a resource with ListAttributes. You can remove existing attributes on
			// a resource with DeleteAttributes.
			ecs.ErrCodeAttributeLimitExceededException,

			// ErrCodeLimitExceededException for service response error code
			// "LimitExceededException".
			//
			// The limit for the resource has been exceeded.
			ecs.ErrCodeLimitExceededException:

			return apierror.New(apierror.ErrLimitExceeded, msg, aerr)
		default:
			m := msg + ": " + aerr.Message()
			return apierror.New(apierror.ErrBadRequest, m, aerr)
		}
	}

	return apierror.New(apierror.ErrInternalError, msg, err)
}
