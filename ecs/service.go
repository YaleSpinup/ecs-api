package ecs

import (
	"context"

	"github.com/YaleSpinup/apierror"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ecs"
	log "github.com/sirupsen/logrus"
)

// GetService describes an ECS service in a cluster by the service name
func (e *ECS) GetService(ctx context.Context, cluster, service string) (*ecs.Service, error) {
	if cluster == "" || service == "" {
		return nil, apierror.New(apierror.ErrBadRequest, "invalid input", nil)
	}

	log.Infof("getting details about service %s/%s", cluster, service)

	output, err := e.Service.DescribeServicesWithContext(ctx, &ecs.DescribeServicesInput{
		Cluster: aws.String(cluster),
		Services: []*string{
			aws.String(service),
		},
	})
	if err != nil {
		return nil, ErrCode("failed to get service", err)
	}

	log.Debugf("got service from DescribeServices: %+v", output)

	if len(output.Services) != 1 {
		return nil, apierror.New(apierror.ErrBadRequest, "unexpected service length in describe services", nil)
	}

	return output.Services[0], nil
}

// DeleteService removes an ECS service in a cluster by the service name (forcefully)
func (e *ECS) DeleteService(ctx context.Context, input *ecs.DeleteServiceInput) error {
	if input == nil {
		return apierror.New(apierror.ErrBadRequest, "invalid input", nil)
	}

	log.Infof("deleting service %s/%s ", aws.StringValue(input.Cluster), aws.StringValue(input.Service))

	output, err := e.Service.DeleteServiceWithContext(ctx, input)
	if err != nil {
		return ErrCode("failed to delete service", err)
	}

	log.Debugf("output from delete service:\n%+v", output)

	return err
}

// ListServices lists the ECS services in a cluster
func (e *ECS) ListServices(ctx context.Context, cluster string) ([]string, error) {
	if cluster == "" {
		return nil, apierror.New(apierror.ErrBadRequest, "invalid input", nil)
	}

	log.Infof("listing services in cluster %s ", cluster)

	input := ecs.ListServicesInput{
		Cluster:    aws.String(cluster),
		LaunchType: aws.String("FARGATE"),
	}

	output := []string{}
	for {
		out, err := e.Service.ListServicesWithContext(ctx, &input)
		if err != nil {
			return output, ErrCode("failed to list services", err)
		}

		for _, t := range out.ServiceArns {
			output = append(output, aws.StringValue(t))
		}

		if out.NextToken == nil {
			break
		}
		input.NextToken = out.NextToken
	}

	log.Debugf("got list of services on cluster '%s': %+v", cluster, output)

	return output, nil
}

// CreateService creates an ECS Service
func (e *ECS) CreateService(ctx context.Context, input *ecs.CreateServiceInput) (*ecs.CreateServiceOutput, error) {
	if input == nil {
		return nil, apierror.New(apierror.ErrBadRequest, "invalid input", nil)
	}

	log.Infof("creating service %s/%s", aws.StringValue(input.Cluster), aws.StringValue(input.ServiceName))

	output, err := e.Service.CreateServiceWithContext(ctx, input)
	if err != nil {
		return nil, ErrCode("failed to create service", err)
	}

	return output, nil
}

// UpdateService updates an ECS service
func (e *ECS) UpdateService(ctx context.Context, input *ecs.UpdateServiceInput) (*ecs.UpdateServiceOutput, error) {
	if input == nil {
		return nil, apierror.New(apierror.ErrBadRequest, "invalid input", nil)
	}

	log.Infof("updating service %s/%s", aws.StringValue(input.Cluster), aws.StringValue(input.Service))

	output, err := e.Service.UpdateServiceWithContext(ctx, input)
	if err != nil {
		return nil, ErrCode("failed to update service", err)
	}

	return output, nil
}
