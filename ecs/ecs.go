package ecs

import (
	"git.yale.edu/spinup/ecs-api/common"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ecs"
	log "github.com/sirupsen/logrus"
)

type ECS struct {
	Service        *ecs.ECS
	DefaultSgs     []string
	DefaultSubnets []string
}

func NewSession(account common.Account) ECS {
	e := ECS{}
	log.Infof("Creating new session with key id %s in region %s", account.Akid, account.Region)
	sess := session.Must(session.NewSession(&aws.Config{
		Credentials: credentials.NewStaticCredentials(account.Akid, account.Secret, ""),
		Region:      aws.String(account.Region),
	}))
	e.Service = ecs.New(sess)

	e.DefaultSgs = account.DefaultSgs
	e.DefaultSubnets = account.DefaultSubnets

	return e
}
