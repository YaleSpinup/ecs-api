package iam

import (
	"encoding/json"

	"github.com/YaleSpinup/apierror"
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

func DocumentFromPolicy(policy *PolicyDoc) (string, error) {
	if policy == nil || len(policy.Statement) == 0 {
		return "", apierror.New(apierror.ErrBadRequest, "invalid input", nil)
	}

	log.Debugf("generating policy document from policy %+v", policy)

	policyDoc, err := json.Marshal(policy)
	if err != nil {
		log.Errorf("failed to generate policy document from policy: %s", err)
		return "", err
	}

	return string(policyDoc), nil
}

func PolicyFromDocument(doc string) (*PolicyDoc, error) {
	if doc == "" {
		return nil, apierror.New(apierror.ErrBadRequest, "invalid input", nil)
	}

	log.Debugf("generating policy struct from document %s", doc)

	policy := PolicyDoc{}
	if err := json.Unmarshal([]byte(doc), &policy); err != nil {
		return nil, err
	}

	if len(policy.Statement) == 0 {
		return nil, apierror.New(apierror.ErrBadRequest, "invalid input", nil)
	}

	return &policy, nil
}
