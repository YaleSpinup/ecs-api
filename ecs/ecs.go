package ecs

import (
	"github.com/YaleSpinup/ecs-api/common"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ecs"
	log "github.com/sirupsen/logrus"
)

// ECS is a wrapper around the aws ECS service with some default config info
type ECS struct {
	Service        *ecs.ECS
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
