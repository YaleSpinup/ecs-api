package orchestration

import (
	"testing"

	"github.com/aws/aws-sdk-go/service/iam/iamiface"
	"github.com/aws/aws-sdk-go/service/secretsmanager"
	"github.com/aws/aws-sdk-go/service/secretsmanager/secretsmanageriface"
	"github.com/aws/aws-sdk-go/service/servicediscovery/servicediscoveryiface"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ecs/ecsiface"
)

type mockECSClient struct {
	ecsiface.ECSAPI
	t   *testing.T
	err error
}

type mockIAMClient struct {
	iamiface.IAMAPI
	t   *testing.T
	err error
}

type mockSDClient struct {
	servicediscoveryiface.ServiceDiscoveryAPI
	t   *testing.T
	err error
}

type mockSMClient struct {
	secretsmanageriface.SecretsManagerAPI
	t   *testing.T
	err error
}

var (
	credentialsMapIn = map[string]*secretsmanager.CreateSecretInput{
		"testDef1": &secretsmanager.CreateSecretInput{
			Name:         aws.String("testDef1"),
			SecretString: aws.String("shhhhhhh"),
		},
		"testDef2": &secretsmanager.CreateSecretInput{
			Name:         aws.String("testDef2"),
			SecretString: aws.String("donttell"),
		},
	}

	credentialsMapOut = map[string]*secretsmanager.CreateSecretOutput{
		"testDef1": &secretsmanager.CreateSecretOutput{
			ARN:       aws.String("arn:testDef1"),
			Name:      aws.String("testDef1"),
			VersionId: aws.String("v1"),
		},
		"testDef2": &secretsmanager.CreateSecretOutput{
			ARN:       aws.String("arn:testDef2"),
			Name:      aws.String("testDef2"),
			VersionId: aws.String("v1"),
		},
	}
)
