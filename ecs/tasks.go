package ecs

import (
	"context"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ecs"
	log "github.com/sirupsen/logrus"
)

// TaskRequest is a request for a new task
type TaskRequest struct {
	Count     int64
	Name      string
	Overrides map[string]ContainerOverride
	Public    bool
	Sgs       []string
	StartedBy string
	Subnets   []string
	TaskDef   string
}

// Task describes an ECS Task
type Task struct {
	ID            string
	Containers    []*Container
	CPU           string
	DesiredStatus string
	HealthStatus  string
	LastStatus    string
	Memory        string
	Name          string
	Overrides     map[string]*ContainerOverride
	Reason        string
	StartedBy     string
	StartedAt     string
	StoppedAt     string
	TaskDef       string
}

// GetTask gets a task by ARN
func (e ECS) GetTask(ctx context.Context, cluster, task string) (*Task, error) {
	log.Infof("describing task %s in cluster %s", task, cluster)

	out, err := e.Service.DescribeTasksWithContext(ctx, &ecs.DescribeTasksInput{
		Cluster: aws.String(cluster),
		Tasks:   aws.StringSlice([]string{task}),
	})

	if err != nil {
		log.Errorf("error describing task: %s", err)
		return nil, err
	}

	log.Debugf("describing task output: %+v", out)

	if len(out.Tasks) != 1 {
		log.Errorf("unexpected task response (length: %d)", len(out.Tasks))
		return nil, fmt.Errorf("unexpected task response (length: %d)", len(out.Tasks))
	}

	return newTaskFromECSTask(out.Tasks[0]), nil
}

// ListTasks lists the tasks in a cluster
func (e ECS) ListTasks(ctx context.Context, cluster, service, family string) (map[string][]string, error) {
	log.Infof("returning a list of tasks in cluster %s", cluster)

	input := &ecs.ListTasksInput{
		Cluster:    aws.String(cluster),
		LaunchType: aws.String("FARGATE"),
	}

	if family != "" {
		log.Infof("filtering task response by family %s", family)
		input.Family = aws.String(family)
	}

	if service != "" {
		log.Infof("filtering task response by service %s", service)
		input.ServiceName = aws.String(service)
	}

	outRunning, err := e.Service.ListTasksWithContext(ctx, input)
	if err != nil {
		log.Errorf("error listing running tasks: %s", err)
		return map[string][]string{}, err
	}

	input.DesiredStatus = aws.String("STOPPED")
	outStopped, err := e.Service.ListTasksWithContext(ctx, input)
	if err != nil {
		log.Errorf("error listing stopped tasks: %s", err)
		return map[string][]string{}, err
	}

	out := map[string][]string{
		"running": aws.StringValueSlice(outRunning.TaskArns),
		"stopped": aws.StringValueSlice(outStopped.TaskArns),
	}

	log.Debugf("listing tasks output: %+v", out)
	return out, nil
}

// DeleteTask stops a task in a cluster
func (e ECS) DeleteTask(ctx context.Context, cluster, task string) (*Task, error) {
	log.Infof("stopping task %s in cluster %s", task, cluster)

	out, err := e.Service.StopTaskWithContext(ctx, &ecs.StopTaskInput{
		Cluster: aws.String(cluster),
		Task:    aws.String(task),
	})

	if err != nil {
		log.Errorf("error stopping task: %s", err)
		return nil, err
	}

	log.Debugf("stopping task output: %+v", out)

	return newTaskFromECSTask(out.Task), nil
}

// CreateTask creates a task in a cluster
func (e ECS) CreateTask(ctx context.Context, cluster string, req TaskRequest) (*Task, error) {
	task, _, err := e.createTask(ctx, cluster, req)
	if err != nil {
		log.Errorf("error creating task %v: %s", req, err)
		return nil, err
	}

	log.Debug("create task output %+v", task)

	return newTaskFromECSTask(task), err
}

// CreateTask creates a task in a cluster and waits for it to be running
func (e ECS) CreateTaskWithWait(ctx context.Context, cluster string, req TaskRequest, wait time.Duration) (*Task, error) {
	task, _, err := e.createTask(ctx, cluster, req)
	if err != nil {
		log.Errorf("error creating task %v: %s", req, err)
		return nil, err
	}

	log.Infof("waiting for task request %v to start in cluster %s", req, cluster)

	ctx, cancel := context.WithTimeout(ctx, wait)
	defer cancel()

	err = e.Service.WaitUntilTasksRunningWithContext(ctx, &ecs.DescribeTasksInput{
		Cluster: aws.String(cluster),
		Tasks:   []*string{task.TaskArn},
	})

	if err != nil {
		log.Errorf("error waiting until tasks is running %v: %s", task, err)
		return nil, err
	}

	return e.GetTask(ctx, cluster, aws.StringValue(task.TaskArn))
}

func (e ECS) createTask(ctx context.Context, cluster string, req TaskRequest) (*ecs.Task, []*ecs.Failure, error) {
	input := &ecs.RunTaskInput{
		Cluster:        aws.String(cluster),
		Count:          aws.Int64(req.Count),
		TaskDefinition: aws.String(req.TaskDef),
		LaunchType:     aws.String("FARGATE"),
	}

	if req.Name != "" {
		input.SetGroup(req.Name)
	}

	if len(req.Overrides) > 0 {
		var containerOverrides []*ecs.ContainerOverride
		for name, override := range req.Overrides {
			o := &ecs.ContainerOverride{
				Name: aws.String(name),
			}

			if len(override.Command) > 0 {
				o.SetCommand(aws.StringSlice(override.Command))
			}

			if len(override.Environment) > 0 {
				var env []*ecs.KeyValuePair
				for k, v := range override.Environment {
					e := ecs.KeyValuePair{
						Name:  aws.String(k),
						Value: aws.String(v),
					}
					env = append(env, &e)
				}
				o.SetEnvironment(env)
			}

			containerOverrides = append(containerOverrides, o)
		}
		input.SetOverrides(&ecs.TaskOverride{
			ContainerOverrides: containerOverrides,
		})
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

	if req.StartedBy != "" {
		input.SetStartedBy(req.StartedBy)
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

	log.Debugf("submitting request to run task with parameters: %+v", input)

	out, err := e.Service.RunTaskWithContext(ctx, input)
	if err != nil {
		log.Errorf("error running task: %s", err)
		return nil, nil, err
	}

	log.Debugf("run task output: %+v", out)

	if len(out.Tasks) != 1 {
		log.Errorf("unexpected task response (length: %d)", len(out.Tasks))
		return nil, nil, fmt.Errorf("unexpected task response (length: %d)", len(out.Tasks))
	}

	if len(out.Failures) > 0 {
		log.Errorf("failures running tasks > 0: %+v", out.Failures)
		return nil, nil, fmt.Errorf("failures running tasks > 0: %+v", out.Failures)
	}

	return out.Tasks[0], out.Failures, nil
}

// newTaskFromECSTask builds up the Task from an ECS Task response
func newTaskFromECSTask(t *ecs.Task) *Task {
	// Build a map of attachments for the task
	attachments := make(map[string]Attachment)
	for _, attachment := range t.Attachments {
		details := make(map[string]string)
		for _, keymap := range attachment.Details {
			details[aws.StringValue(keymap.Name)] = aws.StringValue(keymap.Value)
		}

		attachments[aws.StringValue(attachment.Id)] = Attachment{
			Status:  aws.StringValue(attachment.Status),
			Type:    aws.StringValue(attachment.Type),
			Details: details,
		}
	}

	// Build a list of containers for the task
	var containers []*Container
	for _, c := range t.Containers {
		var networkInterfaces []*NetworkInterface
		for _, i := range c.NetworkInterfaces {
			aID := aws.StringValue(i.AttachmentId)
			n := NetworkInterface{
				AttachmentID: aID,
				ID:           attachments[aID].Details["networkInterfaceId"],
				MacAddress:   attachments[aID].Details["macAddress"],
				PrivateIP:    attachments[aID].Details["privateIPv4Address"],
				Subnet:       attachments[aID].Details["subnetId"],
			}
			networkInterfaces = append(networkInterfaces, &n)
		}

		container := Container{
			ID:                aws.StringValue(c.ContainerArn),
			Health:            aws.StringValue(c.HealthStatus),
			Name:              aws.StringValue(c.Name),
			Status:            aws.StringValue(c.LastStatus),
			TaskID:            aws.StringValue(c.TaskArn),
			NetworkInterfaces: networkInterfaces,
		}
		containers = append(containers, &container)
	}

	overrides := map[string]*ContainerOverride{}
	return &Task{
		ID:            aws.StringValue(t.TaskArn),
		Containers:    containers,
		CPU:           aws.StringValue(t.Cpu),
		DesiredStatus: aws.StringValue(t.DesiredStatus),
		HealthStatus:  aws.StringValue(t.HealthStatus),
		LastStatus:    aws.StringValue(t.LastStatus),
		Memory:        aws.StringValue(t.Memory),
		Name:          aws.StringValue(t.Group),
		Overrides:     overrides,
		Reason:        aws.StringValue(t.StoppedReason),
		StartedBy:     aws.StringValue(t.StartedBy),
		StartedAt:     aws.TimeValue(t.StartedAt).Format("2006/01/02 15:04:05"),
		StoppedAt:     aws.TimeValue(t.StoppedAt).Format("2006/01/02 15:04:05"),
		TaskDef:       aws.StringValue(t.TaskDefinitionArn),
	}
}
