package orchestration

import (
	"context"
	"fmt"
	"time"

	"github.com/YaleSpinup/apierror"
	"github.com/aws/aws-sdk-go/aws"

	"github.com/aws/aws-sdk-go/service/ecs"
	log "github.com/sirupsen/logrus"
)

// processServiceCluster processes the cluster portion of the input, creates a cluster if required and assigns it to the service input
func (o *Orchestrator) processServiceCluster(ctx context.Context, input *ServiceOrchestrationInput) (*ecs.Cluster, rollbackFunc, error) {
	if input.Cluster == nil {
		return nil, defaultRbfunc("processServiceCluster"), apierror.New(apierror.ErrBadRequest, "cluster cannot be empty", nil)
	}

	cluster, rbfunc, err := o.createCluster(ctx, input.Cluster, input.Tags)
	if err != nil {
		return nil, rbfunc, err
	}
	input.Service.Cluster = cluster.ClusterName

	log.Debugf("created cluster %+v", cluster)

	return cluster, rbfunc, nil
}

// processTaskCluster ensures the cluster exists for a task definition
func (o *Orchestrator) processTaskCluster(ctx context.Context, input *TaskDefCreateOrchestrationInput) (*ecs.Cluster, rollbackFunc, error) {
	if input.Cluster == nil {
		return nil, defaultRbfunc("processTaskCluster"), apierror.New(apierror.ErrBadRequest, "cluster cannot be empty", nil)
	}

	cluster, rbfunc, err := o.createCluster(ctx, input.Cluster, input.Tags)
	if err != nil {
		return nil, rbfunc, err
	}

	log.Debugf("created cluster %+v", cluster)

	return cluster, rbfunc, nil
}

// createCluster sets defaults and creates a a tagged ecs cluster
func (o *Orchestrator) createCluster(ctx context.Context, input *ecs.CreateClusterInput, tags []*Tag) (*ecs.Cluster, rollbackFunc, error) {
	rbfunc := defaultRbfunc("createCluster")

	// check if the cluster already exists.  prevents unnecessary api calls and
	// prevents deleting a pre-existing cluster on rollback in the case of error
	if cluster, err := o.ECS.GetCluster(ctx, input.ClusterName); err == nil {
		log.Infof("cluser already exists, returning")
		return cluster, rbfunc, nil
	}

	input.Tags = ecsTags(tags)

	// set the default capacity providers if they are not set in the request
	if input.CapacityProviders == nil {
		input.CapacityProviders = DefaultCapacityProviders
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
