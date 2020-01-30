package cloudwatchlogs

import (
	"github.com/YaleSpinup/ecs-api/apierror"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/cloudwatchlogs"
	"github.com/pkg/errors"
)

func ErrCode(msg string, err error) error {
	if aerr, ok := errors.Cause(err).(awserr.Error); ok {
		switch aerr.Code() {
		case

			// ErrCodeUnrecognizedClientException for service response error code
			// "UnrecognizedClientException".
			//
			// The most likely cause is an invalid AWS access key ID or secret key.
			cloudwatchlogs.ErrCodeUnrecognizedClientException:

			return apierror.New(apierror.ErrForbidden, msg, aerr)
		case

			// ErrCodeServiceUnavailableException for service response error code
			// "ServiceUnavailableException".
			//
			// The service cannot complete the request.
			cloudwatchlogs.ErrCodeServiceUnavailableException:

			return apierror.New(apierror.ErrInternalError, msg, err)
		case

			// ErrCodeDataAlreadyAcceptedException for service response error code
			// "DataAlreadyAcceptedException".
			//
			// The event was already logged.
			cloudwatchlogs.ErrCodeDataAlreadyAcceptedException,

			// ErrCodeOperationAbortedException for service response error code
			// "OperationAbortedException".
			//
			// Multiple requests to update the same resource were in conflict.
			cloudwatchlogs.ErrCodeOperationAbortedException,

			// ErrCodeResourceAlreadyExistsException for service response error code
			// "ResourceAlreadyExistsException".
			//
			// The specified resource already exists.
			cloudwatchlogs.ErrCodeResourceAlreadyExistsException:

			return apierror.New(apierror.ErrConflict, msg, aerr)
		case

			// ErrCodeInvalidOperationException for service response error code
			// "InvalidOperationException".
			//
			// The operation is not valid on the specified resource.
			cloudwatchlogs.ErrCodeInvalidOperationException,

			// ErrCodeInvalidParameterException for service response error code
			// "InvalidParameterException".
			//
			// A parameter is specified incorrectly.
			cloudwatchlogs.ErrCodeInvalidParameterException,

			// ErrCodeInvalidSequenceTokenException for service response error code
			// "InvalidSequenceTokenException".
			//
			// The sequence token is not valid. You can get the correct sequence token in
			// the expectedSequenceToken field in the InvalidSequenceTokenException message.
			cloudwatchlogs.ErrCodeInvalidSequenceTokenException,

			// ErrCodeMalformedQueryException for service response error code
			// "MalformedQueryException".
			//
			// The query string is not valid. Details about this error are displayed in
			// a QueryCompileError object. For more information, see .
			//
			// For more information about valid query syntax, see CloudWatch Logs Insights
			// Query Syntax (https://docs.aws.amazon.com/AmazonCloudWatch/latest/logs/CWL_QuerySyntax.html).
			cloudwatchlogs.ErrCodeMalformedQueryException:

			return apierror.New(apierror.ErrBadRequest, msg, aerr)
		case

			// ErrCodeResourceNotFoundException for service response error code
			// "ResourceNotFoundException".
			//
			// The specified resource does not exist.
			cloudwatchlogs.ErrCodeResourceNotFoundException:

			return apierror.New(apierror.ErrNotFound, msg, aerr)
		case

			// ErrCodeLimitExceededException for service response error code
			// "LimitExceededException".
			//
			// You have reached the maximum number of resources that can be created.
			cloudwatchlogs.ErrCodeLimitExceededException:

			return apierror.New(apierror.ErrLimitExceeded, msg, aerr)
		default:
			m := msg + ": " + aerr.Message()
			return apierror.New(apierror.ErrBadRequest, m, aerr)
		}
	}

	return apierror.New(apierror.ErrInternalError, msg, err)
}
