package orchestration

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/request"
	"github.com/aws/aws-sdk-go/service/ecs"
)

func (m *mockECSClient) CreateClusterWithContext(ctx aws.Context, input *ecs.CreateClusterInput, opts ...request.Option) (*ecs.CreateClusterOutput, error) {
	if m.err != nil {
		return nil, m.err
	}

	if aws.StringValue(input.ClusterName) == "goodclu" {
		return &ecs.CreateClusterOutput{
			Cluster: goodClu,
		}, nil
	}

	return nil, errors.New("Failed to create mock cluster")
}

func (m *mockECSClient) DeleteClusterWithContext(ctx aws.Context, input *ecs.DeleteClusterInput, opts ...request.Option) (*ecs.DeleteClusterOutput, error) {
	if m.err != nil {
		return nil, m.err
	}

	for _, c := range []*ecs.Cluster{goodClu, badClu} {
		if aws.StringValue(input.Cluster) == aws.StringValue(c.ClusterName) {
			return &ecs.DeleteClusterOutput{
				Cluster: c,
			}, nil
		}
	}

	return nil, awserr.New(ecs.ErrCodeClusterNotFoundException, "not found", nil)
}

func (m *mockECSClient) DescribeClustersWithContext(ctx aws.Context, input *ecs.DescribeClustersInput, opts ...request.Option) (*ecs.DescribeClustersOutput, error) {
	if m.err != nil {
		match := false
		if awsErr, ok := m.err.(awserr.Error); ok {
			for _, e := range retryableCluErrs {
				if awsErr.Code() == e.Code() {
					match = true
					break
				}
			}

		}

		if !match {
			return nil, m.err
		}
	}

	if len(input.Clusters) > 1 || aws.StringValue(input.Clusters[0]) == "multiclu" {
		return &ecs.DescribeClustersOutput{
			Clusters: []*ecs.Cluster{
				goodClu,
				&ecs.Cluster{ClusterName: aws.String("fooclu")},
				&ecs.Cluster{ClusterName: aws.String("barclu")},
			},
		}, nil
	} else if len(input.Clusters) == 1 {
		if aws.StringValue(input.Clusters[0]) == "goodclu" {
			return &ecs.DescribeClustersOutput{
				Clusters: []*ecs.Cluster{goodClu},
			}, nil
		} else if aws.StringValue(input.Clusters[0]) == "badclu" {
			return &ecs.DescribeClustersOutput{
				Clusters: []*ecs.Cluster{badClu},
				Failures: []*ecs.Failure{
					&ecs.Failure{
						Arn:    aws.String("arn:aws:ecs:us-east-1:1234567890:thing/broke"),
						Detail: aws.String("something is broken"),
						Reason: aws.String("derpin"),
					},
				},
			}, nil
		}
	}

	return &ecs.DescribeClustersOutput{
		Clusters: []*ecs.Cluster{},
	}, nil
}

func TestCreateCluster(t *testing.T) {
	client := &mockECSClient{t: t}
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

func TestDeleteCluster(t *testing.T) {
	client := &mockECSClient{t: t}
	err := deleteCluster(context.TODO(), client, aws.String("goodclu"))
	if err != nil {
		t.Fatal("expected no error from delete cluster, got", err)
	}

	client.err = awserr.New(ecs.ErrCodeClusterContainsContainerInstancesException, "doin stuff", nil)
	err = deleteCluster(context.TODO(), client, aws.String("goodclu"))
	if err == nil {
		t.Fatal("expected error from delete cluster, got nil")
	}
}

func TestDeleteClusterWithRetry(t *testing.T) {
	client := &mockECSClient{t: t}

	ctx1, cancel1 := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel1()

	cluChan1 := deleteClusterWithRetry(ctx1, client, aws.String("badclu"))
	select {
	case <-ctx1.Done():
		t.Fatal("unexpected context timeout")
	case <-cluChan1:
		t.Log("successfully deleted cluster")
	}

	ctx2, cancel2 := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel2()

	cluChan2 := deleteClusterWithRetry(ctx2, client, aws.String("goodclu"))
	select {
	case <-ctx2.Done():
		t.Log("got expected context timeout")
	case <-cluChan2:
		t.Fatal("expected to timeout, successfully deleted cluster")
	}

	ctx3, cancel3 := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel3()

	client.err = awserr.New(ecs.ErrCodeUpdateInProgressException, "wont fix", nil)
	cluChan3 := deleteClusterWithRetry(ctx3, client, aws.String("missing"))
	select {
	case <-ctx3.Done():
		t.Fatal("unexpected context timeout")
	case <-cluChan3:
		t.Log("successfully deleted cluster")
	}

	for _, e := range retryableCluErrs {
		t.Logf("testing error %s", e.Code())

		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()

		client.err = e
		cluChan := deleteClusterWithRetry(ctx3, client, aws.String("goodclu"))
		select {
		case <-ctx.Done():
			t.Logf("expected context timeout for error code %s", e.Code())
		case <-cluChan:
			t.Fatal("expected to timeout, successfully deleted cluster")
		}
	}

}

func TestGetCluster(t *testing.T) {
	client := &mockECSClient{t: t}
	cluster, err := getCluster(context.TODO(), client, aws.String("goodclu"))
	if err != nil {
		t.Fatal("expected no error from get cluster, got", err)
	}
	t.Log("got cluster response for good cluster", cluster)
	if !reflect.DeepEqual(goodClu, cluster) {
		t.Fatalf("Expected %+v\nGot %+v", goodClu, cluster)
	}

	cluster, err = getCluster(context.TODO(), client, aws.String("missingclu"))
	t.Log("got cluster response for missing cluster", cluster)
	if err == nil {
		t.Fatal("expected error from get missing cluster, got nil")
	} else if err.Error() != "cluster missingclu not found" {
		t.Fatalf("expected error 'cluster missingclu not found' from get cluster, got '%s'", err)
	}

	_, err = getCluster(context.TODO(), client, aws.String("multiclu"))
	if err == nil {
		t.Fatal("expected error from get for multiple clusters, got nil")
	} else if err.Error() != "unexpected number of clusters returned" {
		t.Fatalf("expected error 'cluster missingclu not found' from get cluster, got '%s'", err)
	}

	client.err = awserr.New(ecs.ErrCodeUpdateInProgressException, "wont fix", nil)
	_, err = getCluster(context.TODO(), client, aws.String("goodclu"))
	if err == nil {
		t.Fatal("expected error from get cluster, got nil")
	}
}
