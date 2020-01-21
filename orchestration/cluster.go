package orchestration

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"

	"github.com/aws/aws-sdk-go/service/ecs"
	"github.com/aws/aws-sdk-go/service/ecs/ecsiface"
	log "github.com/sirupsen/logrus"
)

// processCluster processes the cluster portion of the input.  If the cluster is defined on ths service object
// it will be used, otherwise if the ClusterName is given, it will be created.  If neither is provided, an error
// will be returned.
func (o *Orchestrator) processCluster(ctx context.Context, input *ServiceOrchestrationInput) (*ecs.Cluster, error) {
	client := o.ECS.Service
	if input.Service.Cluster != nil {
		log.Infof("Using provided cluster name (input.Service.Cluster) %s", aws.StringValue(input.Service.Cluster))

		cluster, err := getCluster(ctx, client, input.Service.Cluster)
		if err != nil {
			return nil, err
		}

		log.Debugf("Got cluster %+v", cluster)
		return cluster, nil
	} else if input.Cluster != nil {
		log.Infof("Creating cluster %s", aws.StringValue(input.Cluster.ClusterName))

		newTags := []*ecs.Tag{
			&ecs.Tag{
				Key:   aws.String("spinup:org"),
				Value: aws.String(o.Org),
			},
		}

		for _, t := range input.Cluster.Tags {
			if aws.StringValue(t.Key) != "spinup:org" && aws.StringValue(t.Key) != "yale:org" {
				newTags = append(newTags, t)
			}
		}
		input.Cluster.Tags = newTags

		// set the default capacity providers if they are not set in the request
		if input.Cluster.CapacityProviders == nil {
			input.Cluster.CapacityProviders = []*string{
				aws.String("FARGATE"),
				aws.String("FARGATE_SPOT"),
			}
		}

		// set the default capacity providers if they are not set in the request
		if input.Cluster.DefaultCapacityProviderStrategy == nil {
			input.Cluster.DefaultCapacityProviderStrategy = []*ecs.CapacityProviderStrategyItem{
				&ecs.CapacityProviderStrategyItem{
					Base:             aws.Int64(1),
					CapacityProvider: aws.String("FARGATE"),
					Weight:           aws.Int64(0),
				},
				&ecs.CapacityProviderStrategyItem{
					CapacityProvider: aws.String("FARGATE_SPOT"),
					Weight:           aws.Int64(1),
				},
			}
		}

		cluster, err := createCluster(ctx, client, input.Cluster)
		if err != nil {
			return nil, err
		}
		log.Debugf("Created cluster %+v", cluster)
		input.Service.Cluster = cluster.ClusterName
		return cluster, nil
	}
	return nil, errors.New("A new or existing cluster is required")
}

// createCluster creates a cluster with context and name
func createCluster(ctx context.Context, client ecsiface.ECSAPI, cluster *ecs.CreateClusterInput) (*ecs.Cluster, error) {
	log.Debugf("creating cluster with input %+v", cluster)

	output, err := client.CreateClusterWithContext(ctx, cluster)
	if err != nil {
		return nil, err
	}
	return output.Cluster, err
}

// getCluster gets the details of a cluster with context by the cluster name
func getCluster(ctx context.Context, client ecsiface.ECSAPI, name *string) (*ecs.Cluster, error) {
	output, err := client.DescribeClustersWithContext(ctx, &ecs.DescribeClustersInput{
		Clusters: []*string{name},
	})

	if err != nil {
		return nil, err
	}

	if len(output.Failures) > 0 {
		log.Warnf("describe clusters %s returned failures %+v", aws.StringValue(name), output.Failures)
	}

	if len(output.Clusters) == 0 {
		msg := fmt.Sprintf("cluster %s not found", aws.StringValue(name))
		return nil, errors.New(msg)
	} else if len(output.Clusters) > 1 {
		return nil, errors.New("unexpected number of clusters returned")
	}

	return output.Clusters[0], err
}

// deleteCluster deletes a(n empty) cluster
func deleteCluster(ctx context.Context, client ecsiface.ECSAPI, name *string) error {
	_, err := client.DeleteClusterWithContext(ctx, &ecs.DeleteClusterInput{Cluster: name})
	if err != nil {
		return err
	}
	return nil
}

// deleteClusterWithRetry continues to retry deleting a cluster until the context is cancelled or it succeeds
func deleteClusterWithRetry(ctx context.Context, client ecsiface.ECSAPI, arn *string) chan string {
	cluChan := make(chan string, 1)
	go func() {
		t := 1 * time.Second
		for {
			if ctx.Err() != nil {
				log.Debug("cluster delete context is cancelled")
				return
			}

			cluster, err := getCluster(ctx, client, arn)
			if err != nil {
				log.Errorf("error finding cluster to delete %s: %s", aws.StringValue(arn), err)
				cluChan <- "unknown"
				return
			}
			log.Debugf("found cluster %+v", cluster)

			t *= 2
			c := aws.Int64Value(cluster.RegisteredContainerInstancesCount)
			if c > 0 {
				log.Infof("found cluster %s, but registered instance count is > 0 (%d)", aws.StringValue(cluster.ClusterName), c)
				time.Sleep(t)
				continue
			} else {
				log.Infof("found cluster %s with registered instance count of 0, attempting to delete", aws.StringValue(cluster.ClusterName))
				err := deleteCluster(ctx, client, arn)
				if err != nil {
					if awsErr, ok := err.(awserr.Error); ok {
						switch aerr := awsErr.Code(); aerr {
						case ecs.ErrCodeClusterContainsContainerInstancesException,
							ecs.ErrCodeClusterContainsServicesException,
							ecs.ErrCodeClusterContainsTasksException,
							ecs.ErrCodeLimitExceededException,
							ecs.ErrCodeResourceInUseException,
							ecs.ErrCodeServerException:
							log.Warnf("unable to remove cluster %s: %s", aws.StringValue(arn), err)
							time.Sleep(t)
							continue
						default:
							log.Errorf("failed removing cluster %s: %s", aws.StringValue(arn), err)
							cluChan <- "failure"
							return
						}
					}
				}
			}

			cluChan <- "success"
			return
		}
	}()

	return cluChan
}
