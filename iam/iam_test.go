package iam

import (
	"bytes"
	"encoding/json"
	"fmt"
	"reflect"
	"testing"
	"time"

	"github.com/YaleSpinup/ecs-api/common"
	"github.com/aws/aws-sdk-go/service/iam/iamiface"
)

var testTime = time.Now()

// mockIAMClient is a fake IAM client
type mockIAMClient struct {
	iamiface.IAMAPI
	t   *testing.T
	err error
}

func newMockIAMClient(t *testing.T, err error) iamiface.IAMAPI {
	return &mockIAMClient{
		t:   t,
		err: err,
	}
}

func TestNewSession(t *testing.T) {
	e := NewSession(common.Account{})
	to := reflect.TypeOf(e).String()
	if to != "iam.IAM" {
		t.Errorf("expected type to be 'iam.IAM', got %s", to)
	}
}

var path = "org/super-why"

var i = &IAM{
	DefaultKmsKeyID: "123",
}

var defaultPolicyDoc = PolicyDoc{
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
}

func TestDefaultTaskExecutionPolicy(t *testing.T) {
	p, err := json.Marshal(defaultPolicyDoc)
	if err != nil {
		t.Errorf("expected to marshall defaultPolicyDoc with nil error, got %s", err)
	}

	policyBytes, err := i.DefaultTaskExecutionPolicy(path)
	if err != nil {
		t.Errorf("expected DefaultTaskExecutionPolicy to return nil error, got %s", err)
	}

	if !bytes.Equal(policyBytes, p) {
		t.Errorf("expected: %s\ngot: %s", defaultPolicyDoc, policyBytes)
	}
}
