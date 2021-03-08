package ecs

import (
	"context"

	"github.com/YaleSpinup/apierror"
	"github.com/YaleSpinup/ecs-api/common"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ecs"
	"github.com/aws/aws-sdk-go/service/ecs/ecsiface"
	log "github.com/sirupsen/logrus"
)

// ECS is a wrapper around the aws ECS service with some default config info
type ECS struct {
	Service        ecsiface.ECSAPI
	DefaultSgs     []string
	DefaultSubnets []string
}

// NewSession creates a new ECS session
func NewSession(account common.Account) ECS {
	e := ECS{}
	log.Infof("creating new session with key id %s in region %s", account.Akid, account.Region)
	sess := session.Must(session.NewSession(&aws.Config{
		Credentials: credentials.NewStaticCredentials(account.Akid, account.Secret, ""),
		Region:      aws.String(account.Region),
	}))
	e.Service = ecs.New(sess)

	e.DefaultSgs = account.DefaultSgs
	e.DefaultSubnets = account.DefaultSubnets

	return e
}

// KeyValuePair maps a key to a value
type KeyValuePair struct {
	Key   string
	Value string
}

// ListTags returns the list of tags for any ECS rsource
func (e *ECS) ListTags(ctx context.Context, arn string) ([]*ecs.Tag, error) {
	if arn == "" {
		return nil, apierror.New(apierror.ErrBadRequest, "invalid input", nil)
	}

	input := ecs.ListTagsForResourceInput{
		ResourceArn: aws.String(arn),
	}

	output, err := e.Service.ListTagsForResourceWithContext(ctx, &input)
	if err != nil {
		return nil, ErrCode("failed to list tags for ecs resource", err)
	}

	log.Debugf("got list of tags for arn '%s': %+v", arn, output)

	return output.Tags, nil
}

func (e *ECS) TagResource(ctx context.Context, input *ecs.TagResourceInput) error {
	if input == nil {
		return apierror.New(apierror.ErrBadRequest, "invalid input", nil)
	}

	log.Infof("tagging ecs resource %s", aws.StringValue(input.ResourceArn))

	_, err := e.Service.TagResourceWithContext(ctx, input)
	if err != nil {
		return ErrCode("failed to tag resource", err)
	}

	log.Debugf("tagged resource with input %+v", input)
	return nil
}
