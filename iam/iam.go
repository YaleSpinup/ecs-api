package iam

import (
	"encoding/json"
	"fmt"

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
	Principal string `json:",omitempty"`
	Action    []string
	Resource  []string
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
func (i *IAM) DefaultTaskExecutionPolicy(cluster *string) ([]byte, error) {
	c := aws.StringValue(cluster)
	log.Debugf("generating default task execution policy for cluster %s", c)

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
					fmt.Sprintf("arn:aws:ssm:::parameter/*"),
					fmt.Sprintf("arn:aws:kms:::key/%s", i.DefaultKmsKeyID),
				},
			},
		},
	})

	if err != nil {
		log.Errorf("failed to generate default task execution policy for cluster %s: %s", c, err)
		return []byte{}, err
	}

	return policyDoc, nil
}
