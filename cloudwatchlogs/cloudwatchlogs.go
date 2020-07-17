package cloudwatchlogs

import (
	"context"

	"github.com/YaleSpinup/apierror"
	"github.com/YaleSpinup/ecs-api/common"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/cloudwatchlogs"
	"github.com/aws/aws-sdk-go/service/cloudwatchlogs/cloudwatchlogsiface"
	log "github.com/sirupsen/logrus"
)

// CloudWatchLogs is the internal cloudwatch logsobject which holds session
// and configuration information
type CloudWatchLogs struct {
	Service cloudwatchlogsiface.CloudWatchLogsAPI
}

// NewSession builds a new aws cloudwatchlogs session
func NewSession(account common.Account) CloudWatchLogs {
	c := CloudWatchLogs{}
	log.Infof("Creating new session with key id %s in region %s", account.Akid, account.Region)
	sess := session.Must(session.NewSession(&aws.Config{
		Credentials: credentials.NewStaticCredentials(account.Akid, account.Secret, ""),
		Region:      aws.String(account.Region),
	}))
	c.Service = cloudwatchlogs.New(sess)
	return c
}

func (c *CloudWatchLogs) GetLogEvents(ctx context.Context, input *cloudwatchlogs.GetLogEventsInput) (*cloudwatchlogs.GetLogEventsOutput, error) {
	if input == nil {
		return nil, apierror.New(apierror.ErrBadRequest, "invalid input", nil)
	}

	output, err := c.Service.GetLogEventsWithContext(ctx, input)
	if err != nil {
		return nil, ErrCode("failed to get log events", err)
	}

	return output, nil
}

// CreateLogGroup creates a cloudwatchlogs log group
func (c *CloudWatchLogs) CreateLogGroup(ctx context.Context, input *cloudwatchlogs.CreateLogGroupInput) error {
	if input == nil {
		return apierror.New(apierror.ErrBadRequest, "invalid input", nil)
	}

	_, err := c.Service.CreateLogGroupWithContext(ctx, input)
	if err != nil {
		return ErrCode("failed to create log group", err)
	}

	return nil
}

// UpdateRetention changes the retention (in days) for logs in a log group
func (c *CloudWatchLogs) UpdateRetention(ctx context.Context, input *cloudwatchlogs.PutRetentionPolicyInput) error {
	if input == nil {
		return apierror.New(apierror.ErrBadRequest, "invalid input", nil)
	}

	_, err := c.Service.PutRetentionPolicyWithContext(ctx, input)
	if err != nil {
		return ErrCode("failed to update retention policy for log group", err)
	}

	return nil
}
