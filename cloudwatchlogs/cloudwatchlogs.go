package cloudwatchlogs

import (
	"github.com/YaleSpinup/ecs-api/common"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/cloudwatchlogs"
	log "github.com/sirupsen/logrus"
)

// CloudWatchLogs is the internal cloudwatch logsobject which holds session
// and configuration information
type CloudWatchLogs struct {
	Service *cloudwatchlogs.CloudWatchLogs
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
