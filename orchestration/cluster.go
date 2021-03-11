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

// processCluster processes the cluster portion of the input.  If the cluster is defined on ths service object
// it will be used, otherwise if the ClusterName is given, it will be created.  If neither is provided, an error
// will be returned.
func (o *Orchestrator) processCluster(ctx context.Context, input *ServiceOrchestrationInput) (*ecs.Cluster, rollbackFunc, error) {
	rbfunc := func(_ context.Context) error {
		log.Infof("processCluster rollback, nothing to do")
		return nil
	}

	client := o.ECS
	if input.Service != nil && input.Service.Cluster != nil {
		log.Infof("using provided cluster name (input.Service.Cluster) %s", aws.StringValue(input.Service.Cluster))

		cluster, err := client.GetCluster(ctx, input.Service.Cluster)
		if err != nil {
			return nil, rbfunc, err
		}

		log.Debugf("got cluster %+v", cluster)

		return cluster, rbfunc, nil
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

		cluster, err := client.CreateCluster(ctx, input.Cluster)
		if err != nil {
			return nil, rbfunc, err
		}
		input.Service.Cluster = cluster.ClusterName

		rbfunc = func(ctx context.Context) error {
			cluCtx, cancel := context.WithTimeout(ctx, 60*time.Second)
			defer cancel()

			cluChan := client.DeleteClusterWithRetry(ctx, cluster.ClusterArn)
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

	return nil, rbfunc, errors.New("a new or existing cluster is required")
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
