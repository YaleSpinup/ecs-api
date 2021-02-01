package orchestration

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go/aws"

	"github.com/aws/aws-sdk-go/service/ecs"
	log "github.com/sirupsen/logrus"
)

// processService processes the service input.  It normalizes inputs and creates the ECS service.
func (o *Orchestrator) processService(ctx context.Context, input *ServiceOrchestrationInput) (*ecs.Service, rollbackFunc, error) {
	rbfunc := func(_ context.Context) error {
		log.Infof("processService rollback, nothing to do")
		return nil
	}

	if input.Service.ClientToken == nil {
		input.Service.ClientToken = aws.String(o.Token)
	}

	if input.Service.NetworkConfiguration == nil {
		input.Service.NetworkConfiguration = &ecs.NetworkConfiguration{
			AwsvpcConfiguration: &ecs.AwsVpcConfiguration{
				AssignPublicIp: aws.String(o.DefaultPublic),
				SecurityGroups: aws.StringSlice(o.DefaultSecurityGroups),
				Subnets:        aws.StringSlice(o.DefaultSubnets),
			},
		}
	}

	if input.Service.PropagateTags == nil {
		input.Service.PropagateTags = aws.String("SERVICE")
	}

	ecsTags := make([]*ecs.Tag, len(input.Tags))
	for i, t := range input.Tags {
		ecsTags[i] = &ecs.Tag{Key: t.Key, Value: t.Value}
	}
	input.Service.Tags = ecsTags

	log.Debugf("processing service with input:\n%+v", input.Service)
	output, err := o.ECS.CreateService(ctx, input.Service)
	if err != nil {
		return nil, rbfunc, err
	}

	rbfunc = func(ctx context.Context) error {
		name := aws.StringValue(output.Service.ServiceName)
		log.Debugf("rolling back service %s", name)

		if err = o.ECS.DeleteService(ctx, &ecs.DeleteServiceInput{
			Cluster: output.Service.ClusterArn,
			Service: output.Service.ServiceName,
			Force:   aws.Bool(true),
		}); err != nil {
			return fmt.Errorf("failed to rollback service %s: %s", name, err)
		}

		log.Infof("successfully rolled back service %s", name)

		return nil
	}

	return output.Service, rbfunc, nil
}

// processServiceUpdate processes the service update input.  It normalizes inputs and updates and/or redeploys the service.
func (o *Orchestrator) processServiceUpdate(ctx context.Context, input *ServiceOrchestrationUpdateInput, active *ServiceOrchestrationUpdateOutput) error {
	if input.Service != nil {
		// set cluster and service, disallow assigning public IP, default to active service network config
		u := input.Service
		u.Cluster = active.Service.ClusterArn
		u.Service = active.Service.ServiceArn
		if u.NetworkConfiguration != nil && u.NetworkConfiguration.AwsvpcConfiguration != nil {
			subnets := active.Service.NetworkConfiguration.AwsvpcConfiguration.Subnets
			if u.NetworkConfiguration.AwsvpcConfiguration.Subnets != nil {
				subnets = u.NetworkConfiguration.AwsvpcConfiguration.Subnets
			}

			sgs := active.Service.NetworkConfiguration.AwsvpcConfiguration.SecurityGroups
			if u.NetworkConfiguration.AwsvpcConfiguration.SecurityGroups != nil {
				sgs = u.NetworkConfiguration.AwsvpcConfiguration.SecurityGroups
			}

			u.NetworkConfiguration.AwsvpcConfiguration = &ecs.AwsVpcConfiguration{
				AssignPublicIp: aws.String("DISABLED"),
				Subnets:        subnets,
				SecurityGroups: sgs,
			}
		}

		// if we pass in a new capacityproviderstrategy, we must force a new deployment
		if input.ForceNewDeployment || len(input.Service.CapacityProviderStrategy) > 0 {
			u.ForceNewDeployment = aws.Bool(true)
		}

		out, err := o.ECS.UpdateService(ctx, u)
		if err != nil {
			return err
		}

		// override active service with new service
		active.Service = out.Service
	} else if input.ForceNewDeployment {
		out, err := o.ECS.UpdateService(ctx, &ecs.UpdateServiceInput{
			ForceNewDeployment: aws.Bool(true),
			Service:            active.Service.ServiceName,
			Cluster:            active.Service.ClusterArn,
		})
		if err != nil {
			return err
		}
		active.Service = out.Service
	}

	return nil
}
