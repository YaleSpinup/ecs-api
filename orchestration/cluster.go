package orchestration

import (
	"context"
	"errors"

	"github.com/aws/aws-sdk-go/aws"

	"github.com/aws/aws-sdk-go/service/ecs"
	log "github.com/sirupsen/logrus"
)

// processCluster processes the cluster portion of the input.  If the cluster is defined on ths service object
// it will be used, otherwise if the ClusterName is given, it will be created.  If neither is provided, an error
// will be returned.
func (o *Orchestrator) processCluster(ctx context.Context, input *ServiceOrchestrationInput) (*ecs.Cluster, error) {
	client := o.ECS
	if input.Service != nil && input.Service.Cluster != nil {
		log.Infof("using provided cluster name (input.Service.Cluster) %s", aws.StringValue(input.Service.Cluster))

		cluster, err := client.GetCluster(ctx, input.Service.Cluster)
		if err != nil {
			return nil, err
		}

		log.Debugf("got cluster %+v", cluster)
		return cluster, nil
	}

	if input.Cluster != nil {
		ecsTags := make([]*ecs.Tag, len(input.Tags))
		for i, t := range input.Tags {
			ecsTags[i] = &ecs.Tag{Key: t.Key, Value: t.Value}
		}
		input.Cluster.Tags = ecsTags

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

		cluster, err := client.CreateCluster(ctx, input.Cluster)
		if err != nil {
			return nil, err
		}
		log.Debugf("created cluster %+v", cluster)
		input.Service.Cluster = cluster.ClusterName
		return cluster, nil
	}

	return nil, errors.New("a new or existing cluster is required")
}
