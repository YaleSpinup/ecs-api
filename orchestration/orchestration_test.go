package orchestration

import (
	"github.com/aws/aws-sdk-go/service/iam/iamiface"
	"github.com/aws/aws-sdk-go/service/secretsmanager/secretsmanageriface"
	"github.com/aws/aws-sdk-go/service/servicediscovery/servicediscoveryiface"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ecs"
	"github.com/aws/aws-sdk-go/service/ecs/ecsiface"
	"github.com/aws/aws-sdk-go/service/servicediscovery"
)

type mockECSClient struct {
	ecsiface.ECSAPI
}

type mockIAMClient struct {
	iamiface.IAMAPI
}

type mockSDClient struct {
	servicediscoveryiface.ServiceDiscoveryAPI
}

type mockSMClient struct {
	secretsmanageriface.SecretsManagerAPI
}

var (
	goodClu = &ecs.Cluster{
		ActiveServicesCount:               aws.Int64(1),
		ClusterArn:                        aws.String("arn:aws:ecs:us-east-1:1234567890:cluster/goodclu"),
		ClusterName:                       aws.String("goodclu"),
		PendingTasksCount:                 aws.Int64(1),
		RegisteredContainerInstancesCount: aws.Int64(1),
		RunningTasksCount:                 aws.Int64(0),
		Status:                            aws.String("ACTIVE"),
	}

	goodTd = &ecs.TaskDefinition{
		Compatibilities: aws.StringSlice([]string{"EC2", "FARGATE"}),
		ContainerDefinitions: []*ecs.ContainerDefinition{
			&ecs.ContainerDefinition{
				Name:  aws.String("webserver"),
				Image: aws.String("nginx:alpine"),
			},
		},
		Cpu:               aws.String("256"),
		Family:            aws.String("goodtd"),
		Memory:            aws.String("512"),
		Revision:          aws.Int64(666),
		Status:            aws.String("ACTIVE"),
		TaskDefinitionArn: aws.String("arn:aws:ecs:us-east-1:1234567890:task-definition/goodtd:666"),
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
)
