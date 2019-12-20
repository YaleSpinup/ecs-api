package orchestration

import (
	"testing"

	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/iam/iamiface"
	"github.com/aws/aws-sdk-go/service/secretsmanager"
	"github.com/aws/aws-sdk-go/service/secretsmanager/secretsmanageriface"
	"github.com/aws/aws-sdk-go/service/servicediscovery/servicediscoveryiface"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ecs"
	"github.com/aws/aws-sdk-go/service/ecs/ecsiface"
	"github.com/aws/aws-sdk-go/service/servicediscovery"
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
	goodClu = &ecs.Cluster{
		ActiveServicesCount:               aws.Int64(1),
		ClusterArn:                        aws.String("arn:aws:ecs:us-east-1:1234567890:cluster/goodclu"),
		ClusterName:                       aws.String("goodclu"),
		PendingTasksCount:                 aws.Int64(1),
		RegisteredContainerInstancesCount: aws.Int64(1),
		RunningTasksCount:                 aws.Int64(1),
		Status:                            aws.String("ACTIVE"),
	}

	badClu = &ecs.Cluster{
		ActiveServicesCount:               aws.Int64(1),
		ClusterArn:                        aws.String("arn:aws:ecs:us-east-1:1234567890:cluster/badclu"),
		ClusterName:                       aws.String("badclu"),
		PendingTasksCount:                 aws.Int64(1),
		RegisteredContainerInstancesCount: aws.Int64(0),
		RunningTasksCount:                 aws.Int64(0),
		Status:                            aws.String("ACTIVE"),
	}

	goodContainerDefs = []*ecs.ContainerDefinition{
		&ecs.ContainerDefinition{
			Name:  aws.String("webserver"),
			Image: aws.String("nginx:alpine"),
		},
		&ecs.ContainerDefinition{
			Name:  aws.String("testDef1"),
			Image: aws.String("secretImage1"),
		},
		&ecs.ContainerDefinition{
			Name:  aws.String("testDef2"),
			Image: aws.String("secretImage2"),
		},
	}

	goodTd = &ecs.TaskDefinition{
		Compatibilities:      aws.StringSlice([]string{"EC2", "FARGATE"}),
		ContainerDefinitions: goodContainerDefs,
		Cpu:                  aws.String("256"),
		Family:               aws.String("goodtd"),
		Memory:               aws.String("512"),
		Revision:             aws.Int64(666),
		Status:               aws.String("ACTIVE"),
		TaskDefinitionArn:    aws.String("arn:aws:ecs:us-east-1:1234567890:task-definition/goodtd:666"),
	}

	goodSd = &servicediscovery.Service{
		Name: aws.String("goodsd"),
		Arn:  aws.String("arn:aws:servicediscovery:us-east-1:1234567890:service/srv-goodsd"),
		Id:   aws.String("srv-goodsd"),
		DnsConfig: &servicediscovery.DnsConfig{
			DnsRecords: []*servicediscovery.DnsRecord{
				&servicediscovery.DnsRecord{
					TTL:  aws.Int64(30),
					Type: aws.String("A"),
				},
			},
			NamespaceId: aws.String("ns-p5g6iyxdh5c5h3dr"),
		},
	}

	retryableCluErrs = []awserr.Error{
		awserr.New(ecs.ErrCodeClusterContainsContainerInstancesException, "ClusterContainsContainerInstancesException", nil),
		awserr.New(ecs.ErrCodeClusterContainsServicesException, "ClusterContainsServicesException", nil),
		awserr.New(ecs.ErrCodeClusterContainsTasksException, "ClusterContainsTasksException", nil),
		awserr.New(ecs.ErrCodeLimitExceededException, "LimitExceededException", nil),
		awserr.New(ecs.ErrCodeResourceInUseException, "ResourceInUseException", nil),
		awserr.New(ecs.ErrCodeServerException, "ServerException", nil),
	}

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

	tdInput = &ecs.RegisterTaskDefinitionInput{
		ContainerDefinitions: goodContainerDefs,
	}
)
