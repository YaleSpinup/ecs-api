package cloudwatchlogs

import (
	"context"
	"fmt"

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

	log.Debugf("getting log events with input: %+v", input)

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

	log.Debugf("creating log group with input: %+v", input)

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

	log.Debugf("updating log retention with input: %+v", input)

	_, err := c.Service.PutRetentionPolicyWithContext(ctx, input)
	if err != nil {
		return ErrCode("failed to update retention policy for log group", err)
	}

	return nil
}

// GetLogGroup gets the details about a log group using the prefix (which should be unique in our case)
func (c *CloudWatchLogs) GetLogGroup(ctx context.Context, prefix string) (*cloudwatchlogs.LogGroup, error) {
	if prefix == "" {
		return nil, apierror.New(apierror.ErrBadRequest, "invalid input", nil)
	}

	log.Infof("getting details about log group %s", prefix)

	out, err := c.Service.DescribeLogGroupsWithContext(ctx, &cloudwatchlogs.DescribeLogGroupsInput{
		LogGroupNamePrefix: aws.String(prefix),
	})
	if err != nil {
		return nil, ErrCode("failed describing log groups", err)
	}

	log.Debugf("got output from describe log groups: %+v", out)

	if len(out.LogGroups) != 1 {
		msg := fmt.Sprintf("unexpected number of log groups returned (%d)", len(out.LogGroups))
		return nil, apierror.New(apierror.ErrBadRequest, msg, nil)
	}

	return out.LogGroups[0], nil
}
