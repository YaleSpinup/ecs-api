package ecs

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ecs"
	log "github.com/sirupsen/logrus"
)

// ClusterRequest is a request for a new cluster
type ClusterRequest struct {
	Name string
}

// Cluster is an ecs cluster
type Cluster struct {
	ActiveServices               int64
	ID                           string
	Name                         string
	PendingTasks                 int64
	RegisteredContainerInstances int64
	RunningTasks                 int64
	Statistics                   map[string]string
	Status                       string
}

// GetCluster gets the details of a cluster
func (e ECS) GetCluster(ctx context.Context, name string) (*Cluster, error) {
	log.Infof("getting cluster %s details", name)
	out, err := e.Service.DescribeClustersWithContext(ctx, &ecs.DescribeClustersInput{
		Clusters: aws.StringSlice([]string{name}),
	})
	if err != nil {
		log.Errorf("error describing cluster: %s", err)
		return nil, err
	}

	log.Debugf("get cluster output: %+v", out)

	if len(out.Clusters) != 1 {
		log.Errorf("unexpected cluster response (length: %d): %v", len(out.Clusters), out.Clusters)
		return nil, fmt.Errorf("unexpected cluster response (length != 1: %d)", len(out.Clusters))
	}

	return newClusterFromECSCluster(out.Clusters[0]), nil
}

// CreateCluster creates a new ECS cluster
func (e ECS) CreateCluster(ctx context.Context, name string) (*Cluster, error) {
	log.Infof("creating a cluster with name %s", name)

	out, err := e.Service.CreateClusterWithContext(ctx, &ecs.CreateClusterInput{
		ClusterName: aws.String(name),
	})

	if err != nil {
		log.Errorf("error creating cluster %s: %s", name, err)
		return nil, err
	}

	log.Debugf("cluster create output: %+v", out)

	return newClusterFromECSCluster(out.Cluster), nil
}

// ListClusters lists the ECS clusters
func (e ECS) ListClusters(ctx context.Context) ([]string, error) {
	log.Info("Listing clusters")

	out, err := e.Service.ListClustersWithContext(ctx, &ecs.ListClustersInput{})
	if err != nil {
		log.Errorf("error listing clusters: %s", err)
		return []string{}, err
	}

	log.Debugf("output: %+v", out)

	return aws.StringValueSlice(out.ClusterArns), nil
}

// DeleteCluster deletes an ECS cluster
func (e ECS) DeleteCluster(ctx context.Context, name string) (*Cluster, error) {
	log.Infof("deleting cluster %s", name)

	out, err := e.Service.DeleteClusterWithContext(ctx, &ecs.DeleteClusterInput{
		Cluster: aws.String(name),
	})

	if err != nil {
		log.Errorf("error deleting cluster %s: %s", name, err)
		return nil, err
	}

	log.Debugf("cluster delete output: %+v", out)

	return newClusterFromECSCluster(out.Cluster), nil
}

func newClusterFromECSCluster(c *ecs.Cluster) *Cluster {
	statistics := make(map[string]string)
	for _, kmap := range c.Statistics {
		statistics[aws.StringValue(kmap.Name)] = aws.StringValue(kmap.Value)
	}
	return &Cluster{
		ActiveServices:               aws.Int64Value(c.ActiveServicesCount),
		ID:                           aws.StringValue(c.ClusterArn),
		Name:                         aws.StringValue(c.ClusterName),
		PendingTasks:                 aws.Int64Value(c.PendingTasksCount),
		RegisteredContainerInstances: aws.Int64Value(c.RegisteredContainerInstancesCount),
		RunningTasks:                 aws.Int64Value(c.RunningTasksCount),
		Statistics:                   statistics,
		Status:                       aws.StringValue(c.Status),
	}
}
