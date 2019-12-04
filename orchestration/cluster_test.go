package orchestration

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/request"
	"github.com/aws/aws-sdk-go/service/ecs"
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
