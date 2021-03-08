package ecs

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/YaleSpinup/apierror"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/ecs"
	log "github.com/sirupsen/logrus"
)

// CreateCluster creates a cluster with context and name
func (e *ECS) CreateCluster(ctx context.Context, cluster *ecs.CreateClusterInput) (*ecs.Cluster, error) {
	if cluster == nil {
		return nil, apierror.New(apierror.ErrBadRequest, "invalid input", nil)
	}

	log.Infof("creating cluster %s", aws.StringValue(cluster.ClusterName))

	output, err := e.Service.CreateClusterWithContext(ctx, cluster)
	if err != nil {
		return nil, ErrCode("failed to create cluster "+aws.StringValue(cluster.ClusterName), err)
	}

	log.Debugf("created cluster %+v", output)

	return output.Cluster, err
}

// GetCluster gets the details of a cluster with context by the cluster name
func (e *ECS) GetCluster(ctx context.Context, name *string) (*ecs.Cluster, error) {
	if name == nil {
		return nil, apierror.New(apierror.ErrBadRequest, "invalid input", nil)
	}

	log.Infof("getting cluster %s", aws.StringValue(name))

	output, err := e.Service.DescribeClustersWithContext(ctx, &ecs.DescribeClustersInput{
		Clusters: []*string{name},
	})

	if err != nil {
		return nil, err
	}

	log.Debugf("describe cluster output %+v", output)

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

// DeleteCluster deletes a(n empty) cluster
func (e *ECS) DeleteCluster(ctx context.Context, name *string) error {
	if name == nil {
		return apierror.New(apierror.ErrBadRequest, "invalid input", nil)
	}

	log.Infof("deleting cluster %s", aws.StringValue(name))

	_, err := e.Service.DeleteClusterWithContext(ctx, &ecs.DeleteClusterInput{Cluster: name})
	if err != nil {
		return err
	}
	return nil
}

// DeleteClusterWithRetry continues to retry deleting a cluster until the context is cancelled or it succeeds
func (e *ECS) DeleteClusterWithRetry(ctx context.Context, arn *string) chan string {
	log.Infof("deleting cluster %s with retry", aws.StringValue(arn))

	cluChan := make(chan string, 1)
	go func() {
		t := 1 * time.Second
		for {
			if ctx.Err() != nil {
				log.Debug("cluster delete context is cancelled")
				return
			}

			log.Infof("found cluster %s with registered instance count of 0, attempting to delete", aws.StringValue(arn))

			t *= 2
			if err := e.DeleteCluster(ctx, arn); err != nil {
				if awsErr, ok := err.(awserr.Error); ok {
					switch aerr := awsErr.Code(); aerr {
					case ecs.ErrCodeClusterContainsContainerInstancesException,
						ecs.ErrCodeClusterContainsServicesException,
						ecs.ErrCodeClusterContainsTasksException,
						ecs.ErrCodeLimitExceededException,
						ecs.ErrCodeResourceInUseException,
						ecs.ErrCodeServerException,
						ecs.ErrCodeUpdateInProgressException:
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

			cluChan <- "success"
			return
		}
	}()

	return cluChan
}
