package ssm

import (
	"github.com/YaleSpinup/ecs-api/common"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ssm"
	"github.com/aws/aws-sdk-go/service/ssm/ssmiface"
	log "github.com/sirupsen/logrus"
)

// SSM is a wrapper around the aws ssm service with some default config info
type SSM struct {
	Service         ssmiface.SSMAPI
	DefaultKmsKeyId string
}

// NewSession creates a new cloudfront session
func NewSession(account common.Account) SSM {
	s := SSM{}
	log.Infof("creating new aws session for ssm with key id %s in region %s", account.Akid, account.Region)
	sess := session.Must(session.NewSession(&aws.Config{
		Credentials: credentials.NewStaticCredentials(account.Akid, account.Secret, ""),
		Region:      aws.String(account.Region),
	}))
	s.Service = ssm.New(sess)
	s.DefaultKmsKeyId = account.DefaultKmsKeyId
	return s
}
