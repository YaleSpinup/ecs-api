package secretsmanager

import (
	"github.com/YaleSpinup/ecs-api/common"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/secretsmanager"
	"github.com/aws/aws-sdk-go/service/secretsmanager/secretsmanageriface"
	log "github.com/sirupsen/logrus"
)

// SecretsManager is a wrapper around the aws secretsmanager service with some default config info
type SecretsManager struct {
	Service         secretsmanageriface.SecretsManagerAPI
	DefaultKmsKeyId string
}

// NewSession creates a new cloudfront session
func NewSession(account common.Account) SecretsManager {
	s := SecretsManager{}
	log.Infof("creating new aws session for secretsmanager with key id %s in region %s", account.Akid, account.Region)
	sess := session.Must(session.NewSession(&aws.Config{
		Credentials: credentials.NewStaticCredentials(account.Akid, account.Secret, ""),
		Region:      aws.String(account.Region),
	}))
	s.Service = secretsmanager.New(sess)
	s.DefaultKmsKeyId = account.DefaultKmsKeyId
	return s
}
