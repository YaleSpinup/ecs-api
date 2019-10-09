package iam

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/YaleSpinup/ecs-api/common"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/iam"
	"github.com/aws/aws-sdk-go/service/iam/iamiface"
	log "github.com/sirupsen/logrus"
)

// PolicyStatement is an individual IAM Policy statement
type PolicyStatement struct {
	Effect    string
	Action    []string
	Resource  []string            `json:",omitempty"`
	Principal map[string][]string `json:",omitempty"`
}

// PolicyDoc collects the policy statements
type PolicyDoc struct {
	Version   string
	Statement []PolicyStatement
}

// IAM is a wrapper around the aws IAM service with some default config info
type IAM struct {
	Service         iamiface.IAMAPI
	DefaultKmsKeyID string
}

// NewSession creates a new IAM session
func NewSession(account common.Account) IAM {
	i := IAM{}
	log.Infof("creating new aws session for IAM with key id %s in region %s", account.Akid, account.Region)
	sess := session.Must(session.NewSession(&aws.Config{
		Credentials: credentials.NewStaticCredentials(account.Akid, account.Secret, ""),
		Region:      aws.String(account.Region),
	}))

	i.Service = iam.New(sess)
	i.DefaultKmsKeyID = account.DefaultKmsKeyId

	return i
}

// DefaultTaskExecutionPolicy generates the default policy for ECS task execution
func (i *IAM) DefaultTaskExecutionPolicy(path string) ([]byte, error) {
	log.Debugf("generating default task execution policy for %s", path)

	policyDoc, err := json.Marshal(PolicyDoc{
		Version: "2012-10-17",
		Statement: []PolicyStatement{
			PolicyStatement{
				Effect: "Allow",
				Action: []string{
					"ecr:GetAuthorizationToken",
					"ecr:BatchCheckLayerAvailability",
					"ecr:GetDownloadUrlForLayer",
					"ecr:BatchGetImage",
					"logs:CreateLogGroup",
					"logs:CreateLogStream",
					"logs:PutLogEvents",
				},
				Resource: []string{"*"},
			},
			PolicyStatement{
				Effect: "Allow",
				Action: []string{
					"secretsmanager:GetSecretValue",
					"ssm:GetParameters",
					"kms:Decrypt",
				},
				Resource: []string{
					"arn:aws:secretsmanager:::secret:*",
					fmt.Sprintf("arn:aws:ssm:::parameter/%s/*", path),
					fmt.Sprintf("arn:aws:kms:::key/%s", i.DefaultKmsKeyID),
				},
			},
		},
	})

	if err != nil {
		log.Errorf("failed to generate default task execution policy for %s: %s", path, err)
		return []byte{}, err
	}

	return policyDoc, nil
}

// DefaultTaskExecutionRole generates the default role (if it doesn't exist) for ECS task execution and returns the ARN
func (i *IAM) DefaultTaskExecutionRole(ctx context.Context, path string) (string, error) {
	role := fmt.Sprintf("%s-ecsTaskExecution", path[strings.LastIndex(path, "/")+1:])
	log.Infof("generating default task execution role %s", role)

	// check if role already exists
	getRoleOutput, err := i.GetRole(ctx, &iam.GetRoleInput{
		RoleName: aws.String(role),
	})
	if err == nil {
		log.Infof("role already exists: %s", role)
		return *getRoleOutput.Arn, nil
	}
	log.Debugf("unable to find role %s: %s", role, err)

	assumeRolePolicyDoc, err := json.Marshal(PolicyDoc{
		Version: "2012-10-17",
		Statement: []PolicyStatement{
			PolicyStatement{
				Effect: "Allow",
				Action: []string{
					"sts:AssumeRole",
				},
				Principal: map[string][]string{
					"Service": {"ecs-tasks.amazonaws.com"},
				},
			},
		},
	})
	if err != nil {
		log.Errorf("failed to generate default task execution role assume policy for %s: %s", path, err)
		return "", err
	}

	defaultPolicy, err := i.DefaultTaskExecutionPolicy(path)
	if err != nil {
		log.Errorf("failed creating default IAM task execution policy for %s: %s", path, err.Error())
		return "", err
	}

	roleOutput, err := i.CreateRole(ctx, &iam.CreateRoleInput{
		AssumeRolePolicyDocument: aws.String(string(assumeRolePolicyDoc)),
		Description:              aws.String(fmt.Sprintf("ECS task execution policy for %s", path)),
		Path:                     aws.String("/"),
		RoleName:                 aws.String(role),
	})
	if err != nil {
		return "", ErrCode("failed to create role", err)
	}

	// attach default role policy to the role
	err = i.PutRolePolicy(ctx, &iam.PutRolePolicyInput{
		PolicyDocument: aws.String(string(defaultPolicy)),
		PolicyName:     aws.String("ECSTaskAccessPolicy"),
		RoleName:       aws.String(role),
	})
	if err != nil {
		return "", ErrCode("failed to attach policy to role", err)
	}

	return *roleOutput.Arn, nil
}
