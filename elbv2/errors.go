package elbv2

import (
	"github.com/YaleSpinup/ecs-api/apierror"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/elbv2"
	"github.com/pkg/errors"
)

func ErrCode(msg string, err error) error {
	if aerr, ok := errors.Cause(err).(awserr.Error); ok {
		switch aerr.Code() {
		case

			// ErrCodeOperationNotPermittedException for service response error code
			// "OperationNotPermitted".
			//
			// This operation is not allowed.
			elbv2.ErrCodeOperationNotPermittedException:

			return apierror.New(apierror.ErrForbidden, msg, err)

		case

			// ErrCodeTooManyActionsException for service response error code
			// "TooManyActions".
			//
			// You've reached the limit on the number of actions per rule.
			elbv2.ErrCodeTooManyActionsException,

			// ErrCodeTooManyCertificatesException for service response error code
			// "TooManyCertificates".
			//
			// You've reached the limit on the number of certificates per load balancer.
			elbv2.ErrCodeTooManyCertificatesException,

			// ErrCodeTooManyListenersException for service response error code
			// "TooManyListeners".
			//
			// You've reached the limit on the number of listeners per load balancer.
			elbv2.ErrCodeTooManyListenersException,

			// ErrCodeTooManyLoadBalancersException for service response error code
			// "TooManyLoadBalancers".
			//
			// You've reached the limit on the number of load balancers for your AWS account.
			elbv2.ErrCodeTooManyLoadBalancersException,

			// ErrCodeTooManyRegistrationsForTargetIdException for service response error code
			// "TooManyRegistrationsForTargetId".
			//
			// You've reached the limit on the number of times a target can be registered
			// with a load balancer.
			elbv2.ErrCodeTooManyRegistrationsForTargetIdException,

			// ErrCodeTooManyRulesException for service response error code
			// "TooManyRules".
			//
			// You've reached the limit on the number of rules per load balancer.
			elbv2.ErrCodeTooManyRulesException,

			// ErrCodeTooManyTagsException for service response error code
			// "TooManyTags".
			//
			// You've reached the limit on the number of tags per load balancer.
			elbv2.ErrCodeTooManyTagsException,

			// ErrCodeTooManyTargetGroupsException for service response error code
			// "TooManyTargetGroups".
			//
			// You've reached the limit on the number of target groups for your AWS account.
			elbv2.ErrCodeTooManyTargetGroupsException,

			// ErrCodeTooManyTargetsException for service response error code
			// "TooManyTargets".
			//
			// You've reached the limit on the number of targets.
			elbv2.ErrCodeTooManyTargetsException,

			// ErrCodeTooManyUniqueTargetGroupsPerLoadBalancerException for service response error code
			// "TooManyUniqueTargetGroupsPerLoadBalancer".
			//
			// You've reached the limit on the number of unique target groups per load balancer
			// across all listeners. If a target group is used by multiple actions for a
			// load balancer, it is counted as only one use.
			elbv2.ErrCodeTooManyUniqueTargetGroupsPerLoadBalancerException:

			return apierror.New(apierror.ErrLimitExceeded, msg, aerr)
		case
			"Internal Server Error":
			return apierror.New(apierror.ErrInternalError, msg, err)
		case
			// ErrCodeDuplicateListenerException for service response error code
			// "DuplicateListener".
			//
			// A listener with the specified port already exists.
			elbv2.ErrCodeDuplicateListenerException,

			// ErrCodeDuplicateLoadBalancerNameException for service response error code
			// "DuplicateLoadBalancerName".
			//
			// A load balancer with the specified name already exists.
			elbv2.ErrCodeDuplicateLoadBalancerNameException,

			// ErrCodeDuplicateTagKeysException for service response error code
			// "DuplicateTagKeys".
			//
			// A tag key was specified more than once.
			elbv2.ErrCodeDuplicateTagKeysException,

			// ErrCodeDuplicateTargetGroupNameException for service response error code
			// "DuplicateTargetGroupName".
			//
			// A target group with the specified name already exists.
			elbv2.ErrCodeDuplicateTargetGroupNameException,

			// ErrCodePriorityInUseException for service response error code
			// "PriorityInUse".
			//
			// The specified priority is in use.
			elbv2.ErrCodePriorityInUseException,

			// ErrCodeResourceInUseException for service response error code
			// "ResourceInUse".
			//
			// A specified resource is in use.
			elbv2.ErrCodeResourceInUseException,

			// ErrCodeTargetGroupAssociationLimitException for service response error code
			// "TargetGroupAssociationLimit".
			//
			// You've reached the limit on the number of load balancers per target group.
			elbv2.ErrCodeTargetGroupAssociationLimitException:

			return apierror.New(apierror.ErrConflict, msg, aerr)
		case

			// ErrCodeAvailabilityZoneNotSupportedException for service response error code
			// "AvailabilityZoneNotSupported".
			//
			// The specified Availability Zone is not supported.
			elbv2.ErrCodeAvailabilityZoneNotSupportedException,

			// ErrCodeCertificateNotFoundException for service response error code
			// "CertificateNotFound".
			//
			// The specified certificate does not exist.
			elbv2.ErrCodeCertificateNotFoundException,

			// ErrCodeHealthUnavailableException for service response error code
			// "HealthUnavailable".
			//
			// The health of the specified targets could not be retrieved due to an internal
			// error.
			elbv2.ErrCodeHealthUnavailableException,

			// ErrCodeIncompatibleProtocolsException for service response error code
			// "IncompatibleProtocols".
			//
			// The specified configuration is not valid with this protocol.
			elbv2.ErrCodeIncompatibleProtocolsException,

			// ErrCodeInvalidConfigurationRequestException for service response error code
			// "InvalidConfigurationRequest".
			//
			// The requested configuration is not valid.
			elbv2.ErrCodeInvalidConfigurationRequestException,

			// ErrCodeInvalidLoadBalancerActionException for service response error code
			// "InvalidLoadBalancerAction".
			//
			// The requested action is not valid.
			elbv2.ErrCodeInvalidLoadBalancerActionException,

			// ErrCodeInvalidSchemeException for service response error code
			// "InvalidScheme".
			//
			// The requested scheme is not valid.
			elbv2.ErrCodeInvalidSchemeException,

			// ErrCodeInvalidSecurityGroupException for service response error code
			// "InvalidSecurityGroup".
			//
			// The specified security group does not exist.
			elbv2.ErrCodeInvalidSecurityGroupException,

			// ErrCodeInvalidSubnetException for service response error code
			// "InvalidSubnet".
			//
			// The specified subnet is out of available addresses.
			elbv2.ErrCodeInvalidSubnetException,

			// ErrCodeInvalidTargetException for service response error code
			// "InvalidTarget".
			//
			// The specified target does not exist, is not in the same VPC as the target
			// group, or has an unsupported instance type.
			elbv2.ErrCodeInvalidTargetException,

			// ErrCodeUnsupportedProtocolException for service response error code
			// "UnsupportedProtocol".
			//
			// The specified protocol is not supported.
			elbv2.ErrCodeUnsupportedProtocolException:

			return apierror.New(apierror.ErrBadRequest, msg, aerr)
		case

			// ErrCodeListenerNotFoundException for service response error code
			// "ListenerNotFound".
			//
			// The specified listener does not exist.
			elbv2.ErrCodeListenerNotFoundException,

			// ErrCodeLoadBalancerNotFoundException for service response error code
			// "LoadBalancerNotFound".
			//
			// The specified load balancer does not exist.
			elbv2.ErrCodeLoadBalancerNotFoundException,

			// ErrCodeAllocationIdNotFoundException for service response error code
			// "AllocationIdNotFound".
			//
			// The specified allocation ID does not exist.
			elbv2.ErrCodeAllocationIdNotFoundException,

			// ErrCodeRuleNotFoundException for service response error code
			// "RuleNotFound".
			//
			// The specified rule does not exist.
			elbv2.ErrCodeRuleNotFoundException,

			// ErrCodeSSLPolicyNotFoundException for service response error code
			// "SSLPolicyNotFound".
			//
			// The specified SSL policy does not exist.
			elbv2.ErrCodeSSLPolicyNotFoundException,

			// ErrCodeSubnetNotFoundException for service response error code
			// "SubnetNotFound".
			//
			// The specified subnet does not exist.
			elbv2.ErrCodeSubnetNotFoundException,

			// ErrCodeTargetGroupNotFoundException for service response error code
			// "TargetGroupNotFound".
			//
			// The specified target group does not exist.
			elbv2.ErrCodeTargetGroupNotFoundException:

			return apierror.New(apierror.ErrNotFound, msg, aerr)
		default:
			m := msg + ": " + aerr.Message()
			return apierror.New(apierror.ErrBadRequest, m, aerr)
		}
	}

	return apierror.New(apierror.ErrInternalError, msg, err)
}
