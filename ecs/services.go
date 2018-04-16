package ecs

import (
	"context"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ecs"
	uuid "github.com/satori/go.uuid"
	log "github.com/sirupsen/logrus"
)

// ServiceRequest is a request for a new ECS service
type ServiceRequest struct {
	Count     int64
	Grace     string
	Name      string
	Overrides map[string]ContainerOverride
	Public    bool
	Sgs       []string
	Subnets   []string
	TaskDef   string
}

// Service describes an ECS service
type Service struct {
	ClusterID    string
	CreatedAt    string
	Count        int64
	Deployments  []*ServiceDeployment
	DesiredCount int64
	Events       []*ServiceEvent
	ID           string
	Name         string
	Overrides    map[string]*ContainerOverride
	PendingCount int64
	Public       bool
	Sgs          []string
	Status       string
	Subnets      []string
	TaskDef      string
}

// ServiceDeployment describes an ECS service deployment
type ServiceDeployment struct {
	Count           int64
	CreatedAt       string
	DesiredCount    int64
	ID              string
	Public          bool
	Sgs             []string
	Subnets         []string
	PendingCount    int64
	PlatformVersion string
	Status          string
	TaskDef         string
	UpdatedAt       string
}

// ServiceEvent is an event entry for a service
type ServiceEvent struct {
	CreatedAt string
	ID        string
	Message   string
}

// GetService describes a service in a cluster
func (e ECS) GetService(ctx context.Context, cluster, service string) (*Service, error) {
	log.Infof("describing service %s in cluster %s", service, cluster)

	out, err := e.Service.DescribeServicesWithContext(ctx, &ecs.DescribeServicesInput{
		Cluster:  aws.String(cluster),
		Services: aws.StringSlice([]string{service}),
	})

	if err != nil {
		log.Errorf("error describing service: %s", err)
		return nil, err
	}

	log.Debugf("output from get service %+v", out)

	if len(out.Services) != 1 {
		log.Errorf("unexpected service response (length: %d)", len(out.Services))
		return nil, fmt.Errorf("unexpected service response (length: %d)", len(out.Services))
	}

	return newServiceFromECSService(out.Services[0]), nil
}

// GetServiceEvents returns a list of service events
func (e ECS) GetServiceEvents(ctx context.Context, cluster, service string) ([]*ServiceEvent, error) {
	log.Infof("getting events for service %s in cluster %s", service, cluster)

	out, err := e.Service.DescribeServicesWithContext(ctx, &ecs.DescribeServicesInput{
		Cluster:  aws.String(cluster),
		Services: aws.StringSlice([]string{service}),
	})

	if err != nil {
		log.Errorf("error describing service events: %s", err)
		return nil, err
	}

	log.Debugf("output from get service %+v", out)

	if len(out.Services) != 1 {
		log.Errorf("unexpected service response (length: %d)", len(out.Services))
		return nil, fmt.Errorf("unexpected service response (length: %d)", len(out.Services))
	}

	return newServiceFromECSService(out.Services[0]).Events, nil
}

// ListServices lists services in a cluster
func (e ECS) ListServices(ctx context.Context, cluster string) ([]string, error) {
	log.Infof("listing services in cluster %s", cluster)

	out, err := e.Service.ListServicesWithContext(ctx, &ecs.ListServicesInput{
		Cluster:    aws.String(cluster),
		LaunchType: aws.String("FARGATE"),
	})

	if err != nil {
		log.Errorf("error listing services: %s", err)
		return nil, err
	}

	log.Debugf("output from listing services %+v", out)

	return aws.StringValueSlice(out.ServiceArns), nil
}

// DeleteService deletes a service in a cluster
func (e ECS) DeleteService(ctx context.Context, cluster, service string) (*Service, error) {
	log.Infof("deleting service %s in cluster %s", service, cluster)

	_, err := e.Service.UpdateServiceWithContext(ctx, &ecs.UpdateServiceInput{
		Cluster:      aws.String(cluster),
		DesiredCount: aws.Int64(0),
		Service:      aws.String(service),
	})

	if err != nil {
		log.Errorf("error setting service count to 0: %s", err)
		return nil, err
	}

	out, err := e.Service.DeleteServiceWithContext(ctx, &ecs.DeleteServiceInput{
		Cluster: aws.String(cluster),
		Service: aws.String(service),
	})

	if err != nil {
		log.Errorf("error deleting service: %s", err)
		return nil, err
	}

	log.Debugf("output from delete service %+v", out)

	return newServiceFromECSService(out.Service), nil
}

// CreateService creates a new service in a cluster
func (e ECS) CreateService(ctx context.Context, cluster string, req ServiceRequest) (*Service, error) {
	service, err := e.createService(ctx, cluster, req)
	if err != nil {
		log.Errorf("error creating service: %s", err)
		return nil, err
	}

	log.Debugf("output from create service: %v", service)

	return e.GetService(ctx, cluster, aws.StringValue(service.ServiceArn))
}

// CreateServiceWithWait creates a new service in a cluster and waits
func (e ECS) CreateServiceWithWait(ctx context.Context, cluster string, req ServiceRequest, wait time.Duration) (*Service, error) {
	service, err := e.createService(ctx, cluster, req)
	if err != nil {
		log.Errorf("error creating service with wait: %s", err)
		return nil, err
	}

	log.Infof("waiting %s seconds for service request %v to start in cluster %s", (wait * time.Second).String(), req, cluster)

	ctx, cancel := context.WithTimeout(ctx, wait)
	defer cancel()

	err = e.Service.WaitUntilServicesStableWithContext(ctx, &ecs.DescribeServicesInput{
		Cluster:  aws.String(cluster),
		Services: []*string{service.ServiceArn},
	})

	if err != nil {
		log.Errorf("error waiting until service is stable %v: %s", service, err)
		return nil, err
	}

	return e.GetService(ctx, cluster, aws.StringValue(service.ServiceArn))
}

func (e ECS) createService(ctx context.Context, cluster string, req ServiceRequest) (*ecs.Service, error) {
	// clientToken is a unique identifier for this request
	clientToken := uuid.NewV4()
	input := &ecs.CreateServiceInput{
		ClientToken: aws.String(clientToken.String()),
		Cluster:     aws.String(cluster),
		// DeploymentConfiguration: TODO
		DesiredCount: aws.Int64(req.Count),
		LaunchType:   aws.String("FARGATE"),
		// LoadBalancers: TODO
		ServiceName: aws.String(req.Name),
		// ServiceRegistries: TODO
		TaskDefinition: aws.String(req.TaskDef),
	}

	if req.Grace != "" {
		g, err := time.ParseDuration(req.Grace)
		if err != nil {
			log.Errorf("Failed parsing grace %s: %s", req.Grace, err)
			return nil, err
		}
		grace := int64((g * time.Second).Seconds())
		input.SetHealthCheckGracePeriodSeconds(grace)
	}

	public := "DISABLED"
	if req.Public {
		public = "ENABLED"
	}

	var sgs []string
	if len(req.Sgs) > 0 {
		sgs = req.Sgs
	} else {
		sgs = e.DefaultSgs
	}

	var subnets []string
	if len(req.Subnets) > 0 {
		subnets = req.Subnets
	} else {
		subnets = e.DefaultSubnets
	}

	input.SetNetworkConfiguration(&ecs.NetworkConfiguration{
		AwsvpcConfiguration: &ecs.AwsVpcConfiguration{
			AssignPublicIp: aws.String(public),
			SecurityGroups: aws.StringSlice(sgs),
			Subnets:        aws.StringSlice(subnets),
		},
	})

	out, err := e.Service.CreateServiceWithContext(ctx, input)
	if err != nil {
		log.Errorf("error creating service: %s", err)
		return nil, err
	}

	log.Debugf("output from create service: %+v", out)

	return out.Service, nil
}

func newServiceFromECSService(s *ecs.Service) *Service {
	var deployments []*ServiceDeployment
	for _, d := range s.Deployments {
		var p bool
		if aws.StringValue(d.NetworkConfiguration.AwsvpcConfiguration.AssignPublicIp) == "ENABLED" {
			p = true
		}

		deployment := ServiceDeployment{
			Count:           aws.Int64Value(d.RunningCount),
			CreatedAt:       aws.TimeValue(d.CreatedAt).Format("2006/01/02 15:04:05"),
			DesiredCount:    aws.Int64Value(d.DesiredCount),
			ID:              aws.StringValue(d.Id),
			PendingCount:    aws.Int64Value(d.PendingCount),
			PlatformVersion: aws.StringValue(d.PlatformVersion),
			Public:          p,
			Sgs:             aws.StringValueSlice(d.NetworkConfiguration.AwsvpcConfiguration.SecurityGroups),
			Status:          aws.StringValue(d.Status),
			Subnets:         aws.StringValueSlice(d.NetworkConfiguration.AwsvpcConfiguration.Subnets),
			TaskDef:         aws.StringValue(d.TaskDefinition),
			UpdatedAt:       aws.TimeValue(d.UpdatedAt).Format("2006/01/02 15:04:05"),
		}
		deployments = append(deployments, &deployment)
	}

	var events []*ServiceEvent
	for _, e := range s.Events {
		event := ServiceEvent{
			CreatedAt: aws.TimeValue(e.CreatedAt).Format("2006/01/02 15:04:05"),
			ID:        aws.StringValue(e.Id),
			Message:   aws.StringValue(e.Message),
		}
		events = append(events, &event)
	}

	var public bool
	if aws.StringValue(s.NetworkConfiguration.AwsvpcConfiguration.AssignPublicIp) == "ENABLED" {
		public = true
	}

	return &Service{
		ClusterID:    aws.StringValue(s.ClusterArn),
		CreatedAt:    aws.TimeValue(s.CreatedAt).Format("2006/01/02 15:04:05"),
		Count:        aws.Int64Value(s.RunningCount),
		DesiredCount: aws.Int64Value(s.DesiredCount),
		Deployments:  deployments,
		Events:       events,
		ID:           aws.StringValue(s.ServiceArn),
		Name:         aws.StringValue(s.ServiceName),
		PendingCount: aws.Int64Value(s.PendingCount),
		Public:       public,
		Sgs:          aws.StringValueSlice(s.NetworkConfiguration.AwsvpcConfiguration.SecurityGroups),
		Status:       aws.StringValue(s.Status),
		Subnets:      aws.StringValueSlice(s.NetworkConfiguration.AwsvpcConfiguration.Subnets),
		TaskDef:      aws.StringValue(s.TaskDefinition),
	}
}
