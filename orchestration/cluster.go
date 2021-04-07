package orchestration

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go/aws"

	"github.com/aws/aws-sdk-go/service/ecs"
	log "github.com/sirupsen/logrus"
)

// processServiceCluster processes the cluster portion of the input.  If the cluster is defined on ths service object
// it will be used, otherwise if the ClusterName is given, it will be created.  If neither is provided, an error
// will be returned.
func (o *Orchestrator) processServiceCluster(ctx context.Context, input *ServiceOrchestrationInput) (*ecs.Cluster, rollbackFunc, error) {
	rbfunc := defaultRbfunc("processServiceCluster")

	client := o.ECS

	// if the user provided a cluster name with the service definition, get it and return it
	if input.Service != nil && input.Service.Cluster != nil {
		log.Infof("using provided cluster name (input.Service.Cluster) %s", aws.StringValue(input.Service.Cluster))

		cluster, err := client.GetCluster(ctx, input.Service.Cluster)
		if err != nil {
			return nil, rbfunc, err
		}

		log.Debugf("got cluster %+v", cluster)

		return cluster, rbfunc, nil
	}

	// if a cluster input was provided, try to create the cluster
	if input.Cluster != nil {
		cluster, rbfunc, err := o.createCluster(ctx, input.Cluster, input.Tags)
		if err != nil {
			return nil, rbfunc, err
		}
		input.Service.Cluster = cluster.ClusterName

		log.Debugf("created cluster %+v", cluster)

		return cluster, rbfunc, nil
	}

	return nil, rbfunc, errors.New("a new or existing cluster is required")
}

func (o *Orchestrator) processTaskCluster(ctx context.Context, input *TaskCreateOrchestrationInput) (*ecs.Cluster, rollbackFunc, error) {
	rbfunc := defaultRbfunc("processTaskCluster")

	return nil, rbfunc, nil
}

// createCluster sets defaults and creates a a tagged ecs cluster
func (o *Orchestrator) createCluster(ctx context.Context, input *ecs.CreateClusterInput, tags []*Tag) (*ecs.Cluster, rollbackFunc, error) {
	rbfunc := defaultRbfunc("createCluster")

	ecsTags := make([]*ecs.Tag, len(input.Tags))
	for i, t := range input.Tags {
		ecsTags[i] = &ecs.Tag{Key: t.Key, Value: t.Value}
	}
	input.Tags = ecsTags

	// set the default capacity providers if they are not set in the request
	if input.CapacityProviders == nil {
		input.CapacityProviders = []*string{
			aws.String("FARGATE"),
			aws.String("FARGATE_SPOT"),
		}
	}

	// set the default capacity providers if they are not set in the request
	if input.DefaultCapacityProviderStrategy == nil {
		input.DefaultCapacityProviderStrategy = []*ecs.CapacityProviderStrategyItem{
			{
				Base:             aws.Int64(1),
				CapacityProvider: aws.String("FARGATE"),
				Weight:           aws.Int64(0),
			},
			{
				CapacityProvider: aws.String("FARGATE_SPOT"),
				Weight:           aws.Int64(1),
			},
		}
	}

	cluster, err := o.ECS.CreateCluster(ctx, input)
	if err != nil {
		return cluster, rbfunc, err
	}

	rbfunc = func(ctx context.Context) error {
		cluCtx, cancel := context.WithTimeout(ctx, 60*time.Second)
		defer cancel()

		cluChan := o.ECS.DeleteClusterWithRetry(ctx, cluster.ClusterArn)
		select {
		case <-cluCtx.Done():
			return fmt.Errorf("timeout waiting for successful cluster %s rollback", aws.StringValue(cluster.ClusterArn))
		case <-cluChan:
			log.Infof("successfully rolled back cluster %s", aws.StringValue(cluster.ClusterArn))
		}

		return nil
	}

	return cluster, rbfunc, nil
}

func (o *Orchestrator) deleteCluster(ctx context.Context, arn *string) (bool, error) {
	cluster, err := o.ECS.GetCluster(ctx, arn)
	if err != nil {
		return false, err
	}

	activeServicesCount := aws.Int64Value(cluster.ActiveServicesCount)
	log.Debugf("ACTIVE SERVICES COUNT: %d", activeServicesCount)

	// if the active services count is 0, attempt to cleanup the cluster and the role
	if activeServicesCount > 0 {
		log.Infof("not cleaning up cluster '%s' active services count %d > 0", aws.StringValue(arn), activeServicesCount)
		return false, nil
	}

	cluCtx, cluCancel := context.WithTimeout(ctx, 120*time.Second)
	defer cluCancel()

	cluChan := o.ECS.DeleteClusterWithRetry(cluCtx, arn)

	// wait for a done context
	select {
	case <-cluCtx.Done():
		return false, fmt.Errorf("timeout waiting for successful cluster '%s' delete", aws.StringValue(arn))
	case <-cluChan:
		log.Infof("successfully deleted cluster %s", aws.StringValue(arn))
	}

	return true, nil
}
