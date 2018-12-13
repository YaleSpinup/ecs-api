package orchestration

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"testing"

	"github.com/aws/aws-sdk-go/service/servicediscovery/servicediscoveryiface"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/request"
	"github.com/aws/aws-sdk-go/service/ecs"
	"github.com/aws/aws-sdk-go/service/ecs/ecsiface"
	"github.com/aws/aws-sdk-go/service/servicediscovery"
)

type mockECSClient struct {
	ecsiface.ECSAPI
}

type mockSDClient struct {
	servicediscoveryiface.ServiceDiscoveryAPI
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

func (m *mockECSClient) CreateClusterWithContext(ctx aws.Context, input *ecs.CreateClusterInput, opts ...request.Option) (*ecs.CreateClusterOutput, error) {
	if aws.StringValue(input.ClusterName) == "goodclu" {
		return &ecs.CreateClusterOutput{
			Cluster: goodClu,
		}, nil
	}
	return nil, errors.New("Failed to create mock cluster")
}

func (m *mockECSClient) DescribeClustersWithContext(ctx aws.Context, input *ecs.DescribeClustersInput, opts ...request.Option) (*ecs.DescribeClustersOutput, error) {
	if len(input.Clusters) == 1 {
		if aws.StringValue(input.Clusters[0]) == "goodclu" {
			return &ecs.DescribeClustersOutput{
				Clusters: []*ecs.Cluster{goodClu},
			}, nil
		}
		msg := fmt.Sprintf("Failed to get mock cluster %s", aws.StringValue(input.Clusters[0]))
		return nil, errors.New(msg)
	} else if len(input.Clusters) > 1 {
		return &ecs.DescribeClustersOutput{
			Clusters: []*ecs.Cluster{
				goodClu,
				&ecs.Cluster{ClusterName: aws.String("fooclu")},
				&ecs.Cluster{ClusterName: aws.String("barclu")},
			},
		}, nil
	}
	return nil, errors.New("Failed to describe mock clusters")
}

func TestCreateCluster(t *testing.T) {
	client := &mockECSClient{}
	cluster, err := createCluster(context.TODO(), client, &ecs.CreateClusterInput{ClusterName: aws.String("goodclu")})
	if err != nil {
		t.Fatal("expected no error from create cluster, got", err)
	}
	t.Log("got cluster response for good cluster", cluster)
	if !reflect.DeepEqual(goodClu, cluster) {
		t.Fatalf("Expected %+v\nGot %+v", goodClu, cluster)
	}

	cluster, err = createCluster(context.TODO(), client, &ecs.CreateClusterInput{ClusterName: aws.String("badclu")})
	if err == nil {
		t.Fatal("expected error from create cluster, got", err, cluster)
	}
	t.Log("got error response for bad cluster", err)
}

func TestGetCluster(t *testing.T) {
	client := &mockECSClient{}
	cluster, err := getCluster(context.TODO(), client, aws.String("goodclu"))
	if err != nil {
		t.Fatal("expected no error from get cluster, got", err)
	}
	t.Log("got cluster response for good cluster", cluster)
	if !reflect.DeepEqual(goodClu, cluster) {
		t.Fatalf("Expected %+v\nGot %+v", goodClu, cluster)
	}

	cluster, err = createCluster(context.TODO(), client, &ecs.CreateClusterInput{ClusterName: aws.String("badclu")})
	if err == nil {
		t.Fatal("expected error from get cluster, got", err, cluster)
	}
	t.Log("got expected error response for bad cluster", err)
}

func (m *mockECSClient) RegisterTaskDefinitionWithContext(ctx aws.Context, input *ecs.RegisterTaskDefinitionInput, opts ...request.Option) (*ecs.RegisterTaskDefinitionOutput, error) {
	if aws.StringValue(input.Family) == "goodtd" {
		goodTd.Compatibilities = input.RequiresCompatibilities
		goodTd.NetworkMode = input.NetworkMode
		return &ecs.RegisterTaskDefinitionOutput{
			TaskDefinition: goodTd,
		}, nil
	}
	return nil, errors.New("Failed to create mock task definition")
}

func (m *mockECSClient) DescribeTaskDefinitionWithContext(ctx aws.Context, input *ecs.DescribeTaskDefinitionInput, opts ...request.Option) (*ecs.DescribeTaskDefinitionOutput, error) {
	if aws.StringValue(input.TaskDefinition) == "goodtd" {
		return &ecs.DescribeTaskDefinitionOutput{
			TaskDefinition: goodTd,
		}, nil
	}
	msg := fmt.Sprintf("Failed to get mock task definition %s", aws.StringValue(input.TaskDefinition))
	return nil, errors.New(msg)
}

func TestCreateTaskDefinition(t *testing.T) {
	client := &mockECSClient{}

	// test a boring task definition
	td, err := createTaskDefinition(context.TODO(), client, &ecs.RegisterTaskDefinitionInput{Family: aws.String("goodtd")})
	if err != nil {
		t.Fatal("expected no error from create task definition, got:", err)
	}
	t.Log("got task definition response for good task definition", td)
	if !reflect.DeepEqual(goodTd, td) {
		t.Fatalf("Expected %+v\nGot %+v", goodTd, td)
	}

	// test an error task definition
	td, err = createTaskDefinition(context.TODO(), client, &ecs.RegisterTaskDefinitionInput{})
	if err == nil {
		t.Fatal("expected error from create task definition, got", err, td)
	}
	t.Log("got expected error response for bad task definition", err)

	// test a task definition with custom compatabilities
	td, err = createTaskDefinition(context.TODO(), client, &ecs.RegisterTaskDefinitionInput{
		Family:                  aws.String("goodtd"),
		RequiresCompatibilities: aws.StringSlice([]string{"FOOBAR"}),
	})
	if err != nil {
		t.Fatal("expected no error from create task definition with custom compatabilities, got:", err)
	}
	if !reflect.DeepEqual([]string{"FOOBAR"}, aws.StringValueSlice(td.Compatibilities)) {
		t.Fatal("Expected compatabilitieis to be custom:", []string{"FOOBAR"}, "got:", aws.StringValueSlice(td.Compatibilities))
	}
	t.Log("got task definition response for good task definition with custom compatablilities", td)
}

func TestDescribeTaskDefinition(t *testing.T) {
	client := &mockECSClient{}
	td, err := getTaskDefinition(context.TODO(), client, aws.String("goodtd"))
	if err != nil {
		t.Fatal("expected no error from describe task definition, got:", err)
	}
	t.Log("got task definition response for good task definition", td)
	if !reflect.DeepEqual(goodTd, td) {
		t.Fatalf("Expected %+v\nGot %+v", goodTd, td)
	}

	// test an error task definition
	td, err = getTaskDefinition(context.TODO(), client, aws.String("badtd"))
	if err == nil {
		t.Fatal("expected error from create task definition, got", err, td)
	}
	t.Log("got expected error response for bad task definition", err)
}

func (m *mockSDClient) CreateServiceWithContext(ctx aws.Context, input *servicediscovery.CreateServiceInput, opts ...request.Option) (*servicediscovery.CreateServiceOutput, error) {
	if aws.StringValue(input.Name) == "goodsd" {
		return &servicediscovery.CreateServiceOutput{
			Service: goodSd,
		}, nil
	}
	msg := fmt.Sprintf("Failed to get mock service discovery service %s", aws.StringValue(input.Name))
	return nil, errors.New(msg)
}

func (m *mockSDClient) GetServiceWithContext(ctx aws.Context, input *servicediscovery.GetServiceInput, opts ...request.Option) (*servicediscovery.GetServiceOutput, error) {
	if aws.StringValue(input.Id) == "srv-goodsd" {
		return &servicediscovery.GetServiceOutput{
			Service: goodSd,
		}, nil
	}
	msg := fmt.Sprintf("Failed to get mock service discovery service %s", aws.StringValue(input.Id))
	return nil, errors.New(msg)
}

func TestCreateServiceDiscovery(t *testing.T) {
	client := &mockSDClient{}
	sd, err := createServiceDiscoveryService(context.TODO(), client, &servicediscovery.CreateServiceInput{
		Name: aws.String("goodsd"),
		DnsConfig: &servicediscovery.DnsConfig{
			DnsRecords: []*servicediscovery.DnsRecord{
				&servicediscovery.DnsRecord{
					TTL:  aws.Int64(30),
					Type: aws.String("A"),
				},
			},
			NamespaceId: aws.String("ns-p5g6iyxdh5c5h3dr"),
		},
	})
	if err != nil {
		t.Fatal("expected no error from create service discovery service, got", err)
	}
	t.Log("Got service discovery create service output", sd)
	if !reflect.DeepEqual(sd, goodSd) {
		t.Fatalf("expected: %+v\nGot:%+v", goodSd, sd)
	}

	sd, err = createServiceDiscoveryService(context.TODO(), client, &servicediscovery.CreateServiceInput{
		Name: aws.String("badsd"),
	})
	if err == nil {
		t.Fatalf("expected error from bad create service discovery service, got %+v", sd)
	}
	t.Log("Got expected error from bad service discovery create service", err)
}

func TestGetServiceDiscovery(t *testing.T) {
	client := &mockSDClient{}
	sd, err := getServiceDiscoveryService(context.TODO(), client, aws.String("srv-goodsd"))
	if err != nil {
		t.Fatal("expected no error from get service discovery service, got", err)
	}
	t.Log("Got service discovery get service output", sd)
	if !reflect.DeepEqual(sd, goodSd) {
		t.Fatalf("expected: %+v\n Got: %+v", goodSd, sd)
	}

	sd, err = getServiceDiscoveryService(context.TODO(), client, aws.String("srv-badsd"))
	if err == nil {
		t.Fatalf("expected error from bad get service discovery service, got %+v", sd)
	}
	t.Log("Got expected error frmo bad service discovery service", err)
}
