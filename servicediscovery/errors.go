package servicediscovery

import (
	"github.com/YaleSpinup/apierror"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/servicediscovery"
	"github.com/pkg/errors"
)

func ErrCode(msg string, err error) error {
	if aerr, ok := errors.Cause(err).(awserr.Error); ok {
		switch aerr.Code() {
		case

			// ErrCodeResourceLimitExceeded for service response error code
			// "ResourceLimitExceeded".
			//
			// The resource can't be created because you've reached the limit on the number
			// of resources.
			servicediscovery.ErrCodeResourceLimitExceeded:

			return apierror.New(apierror.ErrLimitExceeded, msg, err)
		case

			// ErrCodeNamespaceAlreadyExists for service response error code
			// "NamespaceAlreadyExists".
			//
			// The namespace that you're trying to create already exists.
			servicediscovery.ErrCodeNamespaceAlreadyExists,

			// ErrCodeResourceInUse for service response error code
			// "ResourceInUse".
			//
			// The specified resource can't be deleted because it contains other resources.
			// For example, you can't delete a service that contains any instances.
			servicediscovery.ErrCodeResourceInUse,

			// ErrCodeServiceAlreadyExists for service response error code
			// "ServiceAlreadyExists".
			//
			// The service can't be created because a service with the same name already
			// exists.
			servicediscovery.ErrCodeServiceAlreadyExists:

			return apierror.New(apierror.ErrConflict, msg, aerr)
		case

			// ErrCodeDuplicateRequest for service response error code
			// "DuplicateRequest".
			//
			// The operation is already in progress.
			servicediscovery.ErrCodeDuplicateRequest,

			// ErrCodeInvalidInput for service response error code
			// "InvalidInput".
			//
			// One or more specified values aren't valid. For example, a required value
			// might be missing, a numeric value might be outside the allowed range, or
			// a string value might exceed length constraints.
			servicediscovery.ErrCodeInvalidInput:

			return apierror.New(apierror.ErrBadRequest, msg, aerr)
		case

			// ErrCodeCustomHealthNotFound for service response error code
			// "CustomHealthNotFound".
			//
			// The health check for the instance that is specified by ServiceId and InstanceId
			// is not a custom health check.
			servicediscovery.ErrCodeCustomHealthNotFound,

			// ErrCodeInstanceNotFound for service response error code
			// "InstanceNotFound".
			//
			// No instance exists with the specified ID, or the instance was recently registered,
			// and information about the instance hasn't propagated yet.
			servicediscovery.ErrCodeInstanceNotFound,

			// ErrCodeNamespaceNotFound for service response error code
			// "NamespaceNotFound".
			//
			// No namespace exists with the specified ID.
			servicediscovery.ErrCodeNamespaceNotFound,

			// ErrCodeOperationNotFound for service response error code
			// "OperationNotFound".
			//
			// No operation exists with the specified ID.
			servicediscovery.ErrCodeOperationNotFound,

			// ErrCodeServiceNotFound for service response error code
			// "ServiceNotFound".
			//
			// No service exists with the specified ID.
			servicediscovery.ErrCodeServiceNotFound:

			return apierror.New(apierror.ErrNotFound, msg, aerr)
		default:
			m := msg + ": " + aerr.Message()
			return apierror.New(apierror.ErrBadRequest, m, aerr)
		}
	}

	return apierror.New(apierror.ErrInternalError, msg, err)
}
