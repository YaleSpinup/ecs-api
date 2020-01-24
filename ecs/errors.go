package ecs

import (
	"github.com/YaleSpinup/ecs-api/apierror"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/pkg/errors"
)

func ErrCode(msg string, err error) error {
	if aerr, ok := errors.Cause(err).(awserr.Error); ok {
		switch aerr.Code() {
		case
			"InternalServerError":

			return apierror.New(apierror.ErrInternalError, msg, err)
		case
			"Conflict":

			return apierror.New(apierror.ErrConflict, msg, aerr)
		case
			"Bad Request":

			return apierror.New(apierror.ErrBadRequest, msg, aerr)
		case
			"Not Found":

			return apierror.New(apierror.ErrNotFound, msg, aerr)
		case
			"Limit Exceeded":

			return apierror.New(apierror.ErrLimitExceeded, msg, aerr)
		default:
			m := msg + ": " + aerr.Message()
			return apierror.New(apierror.ErrBadRequest, m, aerr)
		}
	}

	return apierror.New(apierror.ErrInternalError, msg, err)
}
