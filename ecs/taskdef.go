package ecs

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ecs"
	log "github.com/sirupsen/logrus"
)

// ContainerDef is a Container definition used in task definitions to describe the different
// containers that are launched as part of a task
type ContainerDef struct {
	Command     []string
	Entrypoint  []string
	Environment map[string]string
	Image       string
	Labels      map[string]string
	Name        string
	Ports       []int64
}

// TaskDefReq is a task definition request
type TaskDefReq struct {
	Name       string // ecs family
	Size       string // "cpu-mem"
	Containers []ContainerDef
}

// TaskDef is the task definition
type TaskDef struct {
	Compatablities  []string
	Containers      []ContainerDef
	ExecutionRoleID string
	ID              string
	Name            string
	Revision        int64
	Size            string
	Status          string
	TaskRoleID      string
}

// CreateTaskDef creates a ecs task definition
func (e ECS) CreateTaskDef(ctx context.Context, t TaskDefReq) (*TaskDef, error) {
	log.Infof("Creating task definition %s", t.Name)

	cpu, mem, err := resourcesfromSize(t.Size)
	if err != nil {
		log.Error(err)
		return nil, err
	}

	var containerdefs []*ecs.ContainerDefinition
	for _, c := range t.Containers {
		def := newEcsContainerDefFromContainerReq(c)
		containerdefs = append(containerdefs, def)
	}

	out, err := e.Service.RegisterTaskDefinitionWithContext(ctx, &ecs.RegisterTaskDefinitionInput{
		Family:                  aws.String(t.Name),
		Cpu:                     aws.String(cpu),
		Memory:                  aws.String(mem),
		NetworkMode:             aws.String("awsvpc"),
		RequiresCompatibilities: aws.StringSlice([]string{"FARGATE"}),
		ContainerDefinitions:    containerdefs,
	})

	if err != nil {
		log.Errorf("error registering task definition: %s", err)
		return nil, err
	}

	log.Debugf("output: %+v", out)

	return newTaskDefFromECSTaskDefinition(out.TaskDefinition)
}

// ListTaskDefs lists the task definitions
func (e ECS) ListTaskDefs(ctx context.Context, status string) ([]string, error) {
	if status == "" {
		status = "ACTIVE"
	}

	out, err := e.Service.ListTaskDefinitionsWithContext(ctx, &ecs.ListTaskDefinitionsInput{Status: aws.String(status)})
	if err != nil {
		log.Errorf("error listing task definitions: %s", err)
		return []string{}, err
	}

	log.Debugf("output listing task definitions: %+v", out)

	return aws.StringValueSlice(out.TaskDefinitionArns), nil
}

// GetTaskDef gets a task definition
func (e ECS) GetTaskDef(ctx context.Context, name string) (*TaskDef, error) {
	log.Infof("returning information about task definition %s", name)

	out, err := e.Service.DescribeTaskDefinitionWithContext(ctx, &ecs.DescribeTaskDefinitionInput{
		TaskDefinition: aws.String(name),
	})
	if err != nil {
		log.Errorf("error describing task definition: %s", err)
		return nil, err
	}

	log.Debugf("output getting task definition: %+v", out)

	return newTaskDefFromECSTaskDefinition(out.TaskDefinition)

}

// DeleteTaskDef deregisters a task definition
func (e ECS) DeleteTaskDef(ctx context.Context, name string) (*TaskDef, error) {
	log.Infof("Deregistering task definition taskdef %s", name)

	out, err := e.Service.DeregisterTaskDefinitionWithContext(ctx, &ecs.DeregisterTaskDefinitionInput{
		TaskDefinition: aws.String(name),
	})

	if err != nil {
		log.Errorf("error deregistering task definition %s: %s", name, err)
		return nil, err
	}

	log.Debugf("output deregistering task definition: %+v", out)

	return newTaskDefFromECSTaskDefinition(out.TaskDefinition)
}

// resourcesfromSize converts the requested size into vCPU and Memory.
// Allowed combinations are:
//
//    * 256 (.25 vCPU) - Available memory values: 512 (0.5 GB), 1024 (1 GB),
//    2048 (2 GB)
//
//    * 512 (.5 vCPU) - Available memory values: 1024 (1 GB), 2048 (2 GB), 3072
//    (3 GB), 4096 (4 GB)
//
//    * 1024 (1 vCPU) - Available memory values: 2048 (2 GB), 3072 (3 GB), 4096
//    (4 GB), 5120 (5 GB), 6144 (6 GB), 7168 (7 GB), 8192 (8 GB)
//
//    * 2048 (2 vCPU) - Available memory values: Between 4096 (4 GB) and 16384
//    (16 GB) in increments of 1024 (1 GB)
//
//    * 4096 (4 vCPU) - Available memory values: Between 8192 (8 GB) and 30720
//    (30 GB) in increments of 1024 (1 GB)
//
// TODO: validate combo?
func resourcesfromSize(size string) (string, string, error) {
	resources := strings.SplitN(size, "-", 2)
	if len(resources) < 2 {
		return "", "", fmt.Errorf("incorrect size format '%s'", size)
	}

	c, err := strconv.ParseFloat(resources[0], 64)
	if err != nil {
		log.Errorf("Cannot parse cpu value %s as float: %s", resources[0], err)
		return "", "", err
	}

	m, err := strconv.ParseInt(resources[1], 10, 64)
	if err != nil {
		log.Errorf("Cannot parse mem value %s as float: %s", resources[1], err)
		return "", "", err
	}

	return strconv.FormatInt(int64(c*1024), 10), strconv.FormatInt(m, 10), nil
}

// sizeFromResources converts cpu and memory values into a size string
func sizeFromResources(cpu, mem string) (string, error) {
	c, err := strconv.ParseFloat(cpu, 64)
	if err != nil {
		return "", err
	}

	m, err := strconv.ParseFloat(mem, 64)
	if err != nil {
		return "", err
	}

	cString := strconv.FormatFloat(c/1024, 'f', -1, 64)
	mString := strconv.FormatFloat(m, 'f', -1, 64)
	return fmt.Sprintf("%s-%s", cString, mString), nil
}

// newTaskDefFromECSTaskDefinition converts from the ECS Task Definition to a TaskDef
func newTaskDefFromECSTaskDefinition(t *ecs.TaskDefinition) (*TaskDef, error) {
	var cDefs []ContainerDef
	for _, c := range t.ContainerDefinitions {
		def := ContainerDef{
			Command:    aws.StringValueSlice(c.Command),
			Entrypoint: aws.StringValueSlice(c.EntryPoint),
			Image:      aws.StringValue(c.Image),
		}

		env := map[string]string{}
		for _, p := range c.Environment {
			env[aws.StringValue(p.Name)] = aws.StringValue(p.Value)
		}
		def.Environment = env

		ports := []int64{}
		for _, p := range c.PortMappings {
			ports = append(ports, aws.Int64Value(p.ContainerPort))
		}
		def.Ports = ports

		cDefs = append(cDefs, def)
	}

	size, err := sizeFromResources(aws.StringValue(t.Cpu), aws.StringValue(t.Memory))
	if err != nil {
		log.Errorf("unable to get size from resources: %s", err)
		return nil, err
	}

	return &TaskDef{
		Compatablities:  aws.StringValueSlice(t.Compatibilities),
		Containers:      cDefs,
		ExecutionRoleID: aws.StringValue(t.ExecutionRoleArn),
		ID:              aws.StringValue(t.TaskDefinitionArn),
		Name:            aws.StringValue(t.Family),
		Revision:        aws.Int64Value(t.Revision),
		TaskRoleID:      aws.StringValue(t.TaskRoleArn),
		Size:            size,
		Status:          aws.StringValue(t.Status),
	}, nil
}

func newEcsContainerDefFromContainerReq(c ContainerDef) *ecs.ContainerDefinition {
	def := ecs.ContainerDefinition{
		Image: aws.String(c.Image),
		Name:  aws.String(c.Name),
	}

	if len(c.Command) > 0 {
		def.SetCommand(aws.StringSlice(c.Command))
	}

	if len(c.Entrypoint) > 0 {
		def.SetEntryPoint(aws.StringSlice(c.Entrypoint))
	}

	if len(c.Environment) > 0 {
		var env []*ecs.KeyValuePair
		for k, v := range c.Environment {
			e := ecs.KeyValuePair{
				Name:  aws.String(k),
				Value: aws.String(v),
			}
			env = append(env, &e)
		}
		def.SetEnvironment(env)
	}

	if len(c.Labels) > 0 {
		def.SetDockerLabels(aws.StringMap(c.Labels))
	}

	if len(c.Ports) > 0 {
		var ports []*ecs.PortMapping
		for _, p := range c.Ports {
			port := ecs.PortMapping{
				ContainerPort: aws.Int64(p),
			}
			ports = append(ports, &port)
		}
		def.SetPortMappings(ports)
	}

	return &def
}
