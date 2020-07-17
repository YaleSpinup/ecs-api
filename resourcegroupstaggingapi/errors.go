package resourcegroupstaggingapi

import (
	"github.com/YaleSpinup/apierror"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/resourcegroupstaggingapi"
	"github.com/pkg/errors"
)

func ErrCode(msg string, err error) error {
	if aerr, ok := errors.Cause(err).(awserr.Error); ok {
		switch aerr.Code() {
		case

			// ErrCodeInternalServiceException for service response error code
			// "InternalServiceException".
			//
			// The request processing failed because of an unknown error, exception, or
			// failure. You can retry the request.
			resourcegroupstaggingapi.ErrCodeInternalServiceException:

			return apierror.New(apierror.ErrInternalError, msg, err)
		case

			// ErrCodeConcurrentModificationException for service response error code
			// "ConcurrentModificationException".
			//
			// The target of the operation is currently being modified by a different request.
			// Try again later.
			resourcegroupstaggingapi.ErrCodeConcurrentModificationException,

			// ErrCodeThrottledException for service response error code
			// "ThrottledException".
			//
			// The request was denied to limit the frequency of submitted requests.
			resourcegroupstaggingapi.ErrCodeThrottledException:
			return apierror.New(apierror.ErrConflict, msg, aerr)
		case

			// ErrCodeConstraintViolationException for service response error code
			// "ConstraintViolationException".
			//
			// The request was denied because performing this operation violates a constraint.
			//
			// Some of the reasons in the following list might not apply to this specific
			// operation.
			//
			//    * You must meet the prerequisites for using tag policies. For information,
			//    see Prerequisites and Permissions for Using Tag Policies (http://docs.aws.amazon.com/organizations/latest/userguide/orgs_manage_policies_tag-policies-prereqs.html)
			//    in the AWS Organizations User Guide.
			//
			//    * You must enable the tag policies service principal (tagpolicies.tag.amazonaws.com)
			//    to integrate with AWS Organizations For information, see EnableAWSServiceAccess
			//    (http://docs.aws.amazon.com/organizations/latest/APIReference/API_EnableAWSServiceAccess.html).
			//
			//    * You must have a tag policy attached to the organization root, an OU,
			//    or an account.
			resourcegroupstaggingapi.ErrCodeConstraintViolationException,

			// ErrCodeInvalidParameterException for service response error code
			// "InvalidParameterException".
			//
			// This error indicates one of the following:
			//
			//    * A parameter is missing.
			//
			//    * A malformed string was supplied for the request parameter.
			//
			//    * An out-of-range value was supplied for the request parameter.
			//
			//    * The target ID is invalid, unsupported, or doesn't exist.
			//
			//    * You can't access the Amazon S3 bucket for report storage. For more information,
			//    see Additional Requirements for Organization-wide Tag Compliance Reports
			//    (http://docs.aws.amazon.com/organizations/latest/userguide/orgs_manage_policies_tag-policies-prereqs.html#bucket-policies-org-report)
			//    in the AWS Organizations User Guide.
			resourcegroupstaggingapi.ErrCodeInvalidParameterException,

			// ErrCodePaginationTokenExpiredException for service response error code
			// "PaginationTokenExpiredException".
			//
			// A PaginationToken is valid for a maximum of 15 minutes. Your request was
			// denied because the specified PaginationToken has expired.
			resourcegroupstaggingapi.ErrCodePaginationTokenExpiredException:

			return apierror.New(apierror.ErrBadRequest, msg, aerr)
		case
			"Not Found":
			return apierror.New(apierror.ErrNotFound, msg, aerr)
		default:
			m := msg + ": " + aerr.Message()
			return apierror.New(apierror.ErrBadRequest, m, aerr)
		}
	}

	return apierror.New(apierror.ErrInternalError, msg, err)
}
