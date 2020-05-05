package elbv2

import (
	"context"

	"github.com/YaleSpinup/ecs-api/apierror"
	"github.com/YaleSpinup/ecs-api/common"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/elbv2"
	"github.com/aws/aws-sdk-go/service/elbv2/elbv2iface"
	log "github.com/sirupsen/logrus"
)

// ELBV2API is a wrapper around the aws elbv2 service with some default config info
type ELBV2API struct {
	Service elbv2iface.ELBV2API
}

// NewSession creates a new cloudfront session
func NewSession(account common.Account) ELBV2API {
	s := ELBV2API{}
	log.Infof("creating new aws session for elbv2 with key id %s in region %s", account.Akid, account.Region)
	sess := session.Must(session.NewSession(&aws.Config{
		Credentials: credentials.NewStaticCredentials(account.Akid, account.Secret, ""),
		Region:      aws.String(account.Region),
	}))
	s.Service = elbv2.New(sess)
	return s
}

func (e *ELBV2API) GetTargetGroups(ctx context.Context, arns []string) ([]*elbv2.TargetGroup, error) {
	if len(arns) == 0 {
		return nil, apierror.New(apierror.ErrBadRequest, "invalid input", nil)
	}

	log.Infof("describing target groups with arns %+v", arns)

	out, err := e.Service.DescribeTargetGroupsWithContext(ctx, &elbv2.DescribeTargetGroupsInput{
		TargetGroupArns: aws.StringSlice(arns),
	})
	if err != nil {
		return nil, err
	}

	log.Debugf("got output from describe target groups %+v", out)

	return out.TargetGroups, nil
}
