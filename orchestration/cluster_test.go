package orchestration

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/request"
	"github.com/aws/aws-sdk-go/service/ecs"
)

var testClusters = []*ecs.Cluster{
	{
		ActiveServicesCount: aws.Int64(0),
		ClusterArn:          aws.String("arn:aws:ecs:us-east-1:1234567890:cluster/cluster0"),
		ClusterName:         aws.String("cluster0"),
		DefaultCapacityProviderStrategy: []*ecs.CapacityProviderStrategyItem{
			{
				Base:             aws.Int64(1),
				CapacityProvider: aws.String("FARGATE"),
				Weight:           aws.Int64(0),
			},
			{
				CapacityProvider: aws.String("FARGATE_SPOT"),
				Weight:           aws.Int64(1),
			},
		},
		PendingTasksCount:                 aws.Int64(0),
		RegisteredContainerInstancesCount: aws.Int64(0),
		RunningTasksCount:                 aws.Int64(0),
		Status:                            aws.String("ACTIVE"),
	},
	{
		ActiveServicesCount: aws.Int64(1),
		CapacityProviders:   []*string{aws.String("FARGATE")},
		ClusterArn:          aws.String("arn:aws:ecs:us-east-1:1234567890:cluster/cluster1"),
		ClusterName:         aws.String("cluster1"),
		DefaultCapacityProviderStrategy: []*ecs.CapacityProviderStrategyItem{
			{
				Base:             aws.Int64(1),
				CapacityProvider: aws.String("FARGATE"),
				Weight:           aws.Int64(0),
			},
			{
				CapacityProvider: aws.String("FARGATE_SPOT"),
				Weight:           aws.Int64(1),
			},
		},
		PendingTasksCount:                 aws.Int64(1),
		RegisteredContainerInstancesCount: aws.Int64(1),
		RunningTasksCount:                 aws.Int64(1),
		Status:                            aws.String("ACTIVE"),
	},
	{
		ActiveServicesCount: aws.Int64(2),
		CapacityProviders:   []*string{aws.String("FARGATE_SPOT")},
		ClusterArn:          aws.String("arn:aws:ecs:us-east-1:1234567890:cluster/cluster2"),
		ClusterName:         aws.String("cluster2"),
		DefaultCapacityProviderStrategy: []*ecs.CapacityProviderStrategyItem{
			{
				Base:             aws.Int64(1),
				CapacityProvider: aws.String("FARGATE"),
				Weight:           aws.Int64(0),
			},
			{
				CapacityProvider: aws.String("FARGATE_SPOT"),
				Weight:           aws.Int64(1),
			},
		},
		PendingTasksCount:                 aws.Int64(1),
		RegisteredContainerInstancesCount: aws.Int64(1),
		RunningTasksCount:                 aws.Int64(1),
		Status:                            aws.String("ACTIVE"),
		Tags: []*ecs.Tag{
			{
				Key:   aws.String("fuz"),
				Value: aws.String("biz"),
			},
		},
	},
	{
		ActiveServicesCount: aws.Int64(3),
		CapacityProviders: []*string{
			aws.String("FARGATE"),
			aws.String("FARGATE_SPOT"),
		},
		ClusterArn:  aws.String("arn:aws:ecs:us-east-1:1234567890:cluster/cluster3"),
		ClusterName: aws.String("cluster3"),
		DefaultCapacityProviderStrategy: []*ecs.CapacityProviderStrategyItem{
			{
				Base:             aws.Int64(1),
				CapacityProvider: aws.String("FARGATE"),
				Weight:           aws.Int64(0),
			},
			{
				CapacityProvider: aws.String("FARGATE_SPOT"),
				Weight:           aws.Int64(1),
			},
		},
		PendingTasksCount:                 aws.Int64(1),
		RegisteredContainerInstancesCount: aws.Int64(1),
		RunningTasksCount:                 aws.Int64(1),
		Status:                            aws.String("ACTIVE"),
		Tags: []*ecs.Tag{
			{
				Key:   aws.String("foo"),
				Value: aws.String("bar"),
			},
		},
	},
}

func (m *mockECSClient) CreateClusterWithContext(ctx context.Context, input *ecs.CreateClusterInput, opts ...request.Option) (*ecs.CreateClusterOutput, error) {
	if m.err != nil {
		return nil, m.err
	}

	for _, cluster := range testClusters {
		if aws.StringValue(input.ClusterName) == aws.StringValue(cluster.ClusterName) {
			return &ecs.CreateClusterOutput{
				Cluster: cluster,
			}, nil
		}

	}

	return nil, errors.New("boom!")
}

func (m *mockECSClient) DescribeClustersWithContext(ctx aws.Context, input *ecs.DescribeClustersInput, opts ...request.Option) (*ecs.DescribeClustersOutput, error) {
	if m.err != nil {
		return nil, m.err
	}

	clusters := []*ecs.Cluster{}
	for _, inputClu := range input.Clusters {
		for _, cluster := range testClusters {
			if aws.StringValue(inputClu) == aws.StringValue(cluster.ClusterName) {
				clusters = append(clusters, cluster)
			}

		}
	}

	return &ecs.DescribeClustersOutput{Clusters: clusters}, nil
}

func TestProcessCluster(t *testing.T) {
	orchestrator := newMockOrchestrator(t, "myorg", nil, nil, nil, nil, nil)

	if _, _, err := orchestrator.processServiceCluster(context.TODO(), &ServiceOrchestrationInput{}); err == nil {
		t.Error("expected error for missing cluster, got nil")
	}

	for _, c := range testClusters {
		// test including cluster in service
		input := ServiceOrchestrationInput{
			Service: &ecs.CreateServiceInput{
				Cluster: c.ClusterName,
			},
		}

		out, _, err := orchestrator.processServiceCluster(context.TODO(), &input)
		if err != nil {
			t.Errorf("expected nil error, got %s", err)
		}
		t.Logf("got output from process cluster %+v", out)

		if !reflect.DeepEqual(out, c) {
			t.Errorf("expected %+v, got %+v", c, out)
		}

	}

	for _, c := range testClusters {
		// test including cluster in service
		input := ServiceOrchestrationInput{
			Service: &ecs.CreateServiceInput{},
			Cluster: &ecs.CreateClusterInput{
				CapacityProviders: c.CapacityProviders,
				ClusterName:       c.ClusterName,
				Tags:              c.Tags,
			},
		}

		out, _, err := orchestrator.processServiceCluster(context.TODO(), &input)
		if err != nil {
			t.Errorf("expected nil error, got %s", err)
		}
		t.Logf("got output from process cluster %+v", out)

		if !reflect.DeepEqual(out, c) {
			t.Errorf("expected %+v, got %+v", c, out)
		}
	}

	err := awserr.New(ecs.ErrCodeUpdateInProgressException, "broke", nil)
	orchestrator = newMockOrchestrator(t, "myorg", nil, err, nil, nil, nil)

	input := ServiceOrchestrationInput{
		Service: &ecs.CreateServiceInput{
			Cluster: aws.String("cluster0"),
		},
	}
	if _, _, err := orchestrator.processServiceCluster(context.TODO(), &input); err == nil {
		t.Error("expected error, got nil")
	}

	input = ServiceOrchestrationInput{
		Service: &ecs.CreateServiceInput{},
		Cluster: &ecs.CreateClusterInput{
			ClusterName: aws.String("cluster0"),
		},
	}
	if _, _, err := orchestrator.processServiceCluster(context.TODO(), &input); err == nil {
		t.Error("expected error, got nil")
	}
}
