package orchestration

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/YaleSpinup/apierror"
	"github.com/YaleSpinup/ecs-api/resourcegroupstaggingapi"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/arn"
	"github.com/aws/aws-sdk-go/service/secretsmanager"

	"github.com/aws/aws-sdk-go/service/ecs"

	log "github.com/sirupsen/logrus"
)

// TaskCreateOrchestrationInput is the input payload for creating a task
type TaskDefCreateOrchestrationInput struct {
	Cluster        *ecs.CreateClusterInput
	TaskDefinition *ecs.RegisterTaskDefinitionInput
	Credentials    map[string]*secretsmanager.CreateSecretInput
	Tags           []*Tag
}

// TaskCreateOrchestrationOutput is the output payload for a task creation
type TaskDefCreateOrchestrationOutput struct {
	Cluster *ecs.Cluster
	// map of container definition names to private repository credentials
	// https://docs.aws.amazon.com/sdk-for-go/api/service/secretsmanager/#CreateSecretOutput
	Credentials    map[string]*secretsmanager.CreateSecretOutput
	TaskDefinition *ecs.TaskDefinition
}

// TaskDefUpdateOrchestrationInput is the input payload for updating a taskdef
type TaskDefUpdateOrchestrationInput struct {
	ClusterName    string
	TaskDefinition *ecs.RegisterTaskDefinitionInput
	Credentials    map[string]*secretsmanager.CreateSecretInput
	Tags           []*Tag
}

// TaskDefUpdateOrchestrationOutput is the output payload for updating a taskdef
type TaskDefUpdateOrchestrationOutput struct {
	Cluster             *ecs.Cluster
	TaskDefinition      *ecs.TaskDefinition
	Credentials         map[string]interface{}
	CloudwatchLogGroups []string
	Tags                []*Tag
}

// TaskDefDeleteInput encapsulates a request to delete a taskdef with optional recursion.  If force is
// truthy, running tasks will be stopped before deleting, otherwise running tasks will result in an error.
type TaskDefDeleteInput struct {
	Cluster        string
	TaskDefinition string
	Recursive      bool
	Force          bool
}

// TaskDefDeleteOutput is the orchestration response for taskdef deletion
type TaskDefDeleteOutput struct {
	Cluster        string
	TaskDefinition string
	Tasks          []string
}

type TaskDefShowOutput struct {
	Cluster        *ecs.Cluster
	TaskDefinition *ecs.TaskDefinition
	Tags           []*ecs.Tag
}

type TaskDefRunOrchestrationInput *ecs.RunTaskInput

// CreateTask orchestrates the creation of a task.  It creates a cluster, creates repository credrentials in
// secretsmanager, and then creates the task definition.
func (o *Orchestrator) CreateTaskDef(ctx context.Context, input *TaskDefCreateOrchestrationInput) (*TaskDefCreateOrchestrationOutput, error) {
	log.Debugf("got create task orchestration input object:\n %+v", input.TaskDefinition)
	if input.TaskDefinition == nil {
		return nil, apierror.New(apierror.ErrBadRequest, "task definition is required", nil)
	}

	spaceid := aws.StringValue(input.Cluster.ClusterName)

	ct, err := cleanTags(o.Org, spaceid, "container", "task", input.Tags)
	if err != nil {
		return nil, err
	}
	input.Tags = ct

	// setup err var, rollback function list and defer execution, note that we depend on the err variable defined above this
	var rollBackTasks []rollbackFunc
	defer func() {
		if err != nil {
			log.Errorf("recovering from error: %s, executing %d rollback tasks", err, len(rollBackTasks))
			go rollBack(&rollBackTasks)
		}
	}()

	output := &TaskDefCreateOrchestrationOutput{}

	cluster, rbfunc, err := o.processTaskDefCluster(ctx, input)
	if err != nil {
		return nil, err
	}
	output.Cluster = cluster
	rollBackTasks = append(rollBackTasks, rbfunc)

	creds, rbfunc, err := o.processTaskDefRepositoryCredentialsCreate(ctx, input)
	if err != nil {
		return nil, err
	}
	output.Credentials = creds
	rollBackTasks = append(rollBackTasks, rbfunc)

	td, rbfunc, err := o.processTaskDefTaskDefinitionCreate(ctx, input)
	if err != nil {
		return nil, err
	}
	output.TaskDefinition = td
	rollBackTasks = append(rollBackTasks, rbfunc)

	return output, nil
}

// UpdateTaskDef takes the task definition update input and orchestrates the update for a task definition and related resources
func (o *Orchestrator) UpdateTaskDef(ctx context.Context, cluster, family string, input *TaskDefUpdateOrchestrationInput) (*TaskDefUpdateOrchestrationOutput, error) {
	output := &TaskDefUpdateOrchestrationOutput{}

	clu, err := o.ECS.GetCluster(ctx, aws.String(cluster))
	if err != nil {
		return nil, err
	}
	output.Cluster = clu

	input.ClusterName = cluster

	taskdef, tags, err := o.ECS.GetTaskDefinition(ctx, aws.String(family), true)
	if err != nil {
		return nil, err
	}

	log.Debugf("got task definition %+v", taskdef)

	output.TaskDefinition = taskdef

	if input.TaskDefinition != nil {
		// if the tags are empty for the task definition, apply the existing tags
		if input.TaskDefinition.Tags == nil {
			input.TaskDefinition.Tags = tags
		}

		// updates active Credentials
		if err := o.processTaskDefRepositoryCredentialsUpdate(ctx, input, output); err != nil {
			return nil, err
		}

		// updates taskDefinition
		if err := o.processTaskDefTaskDefinitionUpdate(ctx, input, output); err != nil {
			return nil, err
		}
	}

	cwlgs, err := o.cloudwatchLogGroups(ctx, output.TaskDefinition.ContainerDefinitions)
	if err != nil {
		return nil, err
	}
	output.CloudwatchLogGroups = cwlgs

	// if the input tags are passed, clean them and use them, otherwise set to the active tags
	if input.Tags != nil {
		ct, err := cleanTags(o.Org, cluster, "container", "service", input.Tags)
		if err != nil {
			return nil, err
		}
		input.Tags = ct
	} else {
		inputTags := make([]*Tag, len(tags))
		for i, t := range tags {
			inputTags[i] = &Tag{Key: t.Key, Value: t.Value}
		}
		input.Tags = inputTags
	}

	// updates active.Tags
	if err := o.processTaskDefTagsUpdate(ctx, output, input.Tags); err != nil {
		return nil, err
	}

	return output, nil
}

// DeleteTaskDef deletes all task definition revisions and related resources.
func (o *Orchestrator) DeleteTaskDef(ctx context.Context, input *TaskDefDeleteInput) (*TaskDefDeleteOutput, error) {
	output := TaskDefDeleteOutput{
		Cluster:        input.Cluster,
		TaskDefinition: input.TaskDefinition,
	}

	taskDefinition, _, err := o.ECS.GetTaskDefinition(ctx, aws.String(input.TaskDefinition), false)
	if err != nil {
		return nil, err
	}

	runningTasks, err := o.ListTaskDefTasks(ctx, input.Cluster, *taskDefinition.Family, "", []string{"RUNNING"})
	if err != nil {
		return nil, err
	}

	if l := len(runningTasks); l > 0 && !input.Force {
		msg := fmt.Sprintf("running tasks > 0 (%d) and 'force' is not set", l)
		return nil, apierror.New(apierror.ErrBadRequest, msg, nil)
	}

	// list all of the revisions in the task definition family
	taskDefinitionRevisions, err := o.ECS.ListTaskDefinitionRevisions(ctx, taskDefinition.Family)
	if err != nil {
		return nil, err
	}

	if len(taskDefinitionRevisions) == 0 {
		return nil, fmt.Errorf("expected more than 0 task definition revisions for %s", aws.StringValue(taskDefinition.Family))
	}

	// for each task definition revision in the task definition family, delete any existing repository credentials, keeping track
	// of ones we delete so we don't try to re-delete them.
	// TODO: if we want to share repository credentials, we need to look for multiple container definitions using the same credentials.
	deletedCredentials := make(map[string]struct{})

	// delete the first task definition
	if err := o.deleteTaskDefinitionRevision(ctx, taskDefinitionRevisions[0], deletedCredentials); err != nil {
		return nil, fmt.Errorf("failed to delete task definition revision %s: %+v", taskDefinitionRevisions[0], err)
	}

	// delete the remaining revisions in the background
	if len(taskDefinitionRevisions) > 1 {
		go func(revList []string) {
			cleanupCtx := context.Background()
			for _, revision := range taskDefinitionRevisions {
				if err := o.deleteTaskDefinitionRevision(cleanupCtx, revision, deletedCredentials); err != nil {
					log.Errorf("failed to delete task def revision %s: %+v", revision, err)
					continue
				}
			}
		}(taskDefinitionRevisions[1:])
	}

	// stop the running tasks and cleanup in the background
	go func() {
		// create a new context for the cleanup
		cleanupCtx, cancel := context.WithCancel(context.Background())
		defer cancel()

		if l := len(runningTasks); l > 0 {
			taskIds := make([]string, len(runningTasks))
			reason := fmt.Sprintf("Deleting task definition %s", input.TaskDefinition)

			// set desired status for all tasks to STOPPED
			for i, t := range runningTasks {
				splitTask := strings.SplitN(t, "/", 2)
				taskIds[i] = splitTask[1]

				if _, err := o.ECS.StopTask(cleanupCtx, &ecs.StopTaskInput{
					Cluster: aws.String(input.Cluster),
					Reason:  aws.String(reason),
					Task:    aws.String(splitTask[1]),
				}); err != nil {
					log.Errorf("failed calling StopTask for %s", t)
					return
				}
			}

			// wait for tasks to become STOPPED
			if err := retry(10, 10*time.Second, func() error {
				log.Infof("waiting for tasks %s to be stopped...", strings.Join(taskIds, ","))

				out, err := o.ECS.GetTasks(cleanupCtx, &ecs.DescribeTasksInput{
					Cluster: &input.Cluster,
					Tasks:   aws.StringSlice(taskIds),
				})
				if err != nil {
					return err
				}

				pending := []string{}
				for _, t := range out.Tasks {
					if status := aws.StringValue(t.LastStatus); status != "STOPPED" {
						pending = append(pending, status)
					}
				}

				if len(pending) > 0 {
					return fmt.Errorf("%s still running", strings.Join(pending, ","))
				}

				return nil
			}); err != nil {
				log.Errorf("failed to stop tasks %s: %s", strings.Join(taskIds, ","), err)
				return
			}
		}

		if input.Recursive {
			deletedCluster, err := o.deleteCluster(cleanupCtx, &input.Cluster)
			if err != nil {
				log.Errorf("failed to delete cluster: %s", err)
				return
			}

			if deletedCluster {
				log.Infof("deleted cluster %s", input.Cluster)

				executionRoleName := fmt.Sprintf("%s-ecsTaskExecution", input.Cluster)
				if err := o.deleteDefaultTaskExecutionRole(cleanupCtx, executionRoleName); err != nil {
					log.Errorf("failed to cleanup default task execution role: %s", err)
				}

				log.Infof("deleted default task execution role: %s", executionRoleName)
			}
		}
	}()

	return &output, nil
}

// deleteTaskDefinitionRevision deletes a task definition revision and associated secretsmanager secrets.  It keeps track
// of deleted secrets through the deleteCredentials map
func (o *Orchestrator) deleteTaskDefinitionRevision(ctx context.Context, revision string, deletedCredentials map[string]struct{}) []error {
	var errors []error
	taskDefinition, _, err := o.ECS.GetTaskDefinition(ctx, aws.String(revision), false)
	if err != nil {
		log.Errorf("failed to get task definition revisions '%s' to delete: %s", revision, err)
		return []error{err}
	}

	for _, cd := range taskDefinition.ContainerDefinitions {
		tdArn := aws.StringValue(taskDefinition.TaskDefinitionArn)
		log.Debugf("cleaning '%s' container definition '%s' components", tdArn, aws.StringValue(cd.Name))

		if cd.RepositoryCredentials != nil && aws.StringValue(cd.RepositoryCredentials.CredentialsParameter) != "" {
			credsArn := aws.StringValue(cd.RepositoryCredentials.CredentialsParameter)

			if _, ok := deletedCredentials[credsArn]; !ok {
				_, err = o.SecretsManager.DeleteSecret(ctx, credsArn, 0)
				if err != nil {
					errors = append(errors, err)
					continue
				}

				deletedCredentials[credsArn] = struct{}{}
				log.Infof("successfully deleted secretsmanager secret '%s'", credsArn)
			}
		}
	}

	out, err := o.ECS.DeleteTaskDefinition(ctx, aws.String(revision))
	if err != nil {
		log.Errorf("failed to delete task definition '%s': %s", revision, err)
		return append(errors, err)
	}

	log.Debugf("successfully deleted task definition revision %s: %+v", revision, out)

	return errors
}

// ListTaskDefs gets a list of task definitions in a cluster using tags
func (o *Orchestrator) ListTaskDefs(ctx context.Context, cluster string) ([]string, error) {
	log.Infof("listing task definitions in cluster '%s'", cluster)

	tagFilters := []*resourcegroupstaggingapi.TagFilter{
		{
			Key:   "spinup:org",
			Value: []string{o.Org},
		},
		{
			Key:   "spinup:type",
			Value: []string{"container"},
		},
		{
			Key:   "spinup:flavor",
			Value: []string{"task"},
		},
	}

	if cluster != "" {
		tagFilters = append(tagFilters, &resourcegroupstaggingapi.TagFilter{
			Key:   "spinup:spaceid",
			Value: []string{cluster},
		})
	}

	taskDefinitionRevisions, err := o.ResourceGroupsTaggingAPI.GetResourcesWithTags(ctx, []string{"ecs:task-definition"}, tagFilters)
	if err != nil {
		return nil, err
	}

	// get a unique list of task definition families from the list of revisions
	families := map[string]struct{}{}
	for _, td := range taskDefinitionRevisions {
		tdArn, err := arn.Parse(td)
		if err != nil {
			log.Warnf("failed to parse ARN %s: %s", tdArn, err)
			families[td] = struct{}{}
			continue
		}

		parts := strings.SplitN(tdArn.Resource, ":", 2)
		family := strings.TrimPrefix(parts[0], "task-definition/")

		log.Debugf("got family %s from arn %s", family, tdArn)

		families[family] = struct{}{}
	}

	taskDefinitionFamilies := make([]string, len(families))
	i := 0
	for f := range families {
		taskDefinitionFamilies[i] = f
		i++
	}

	return taskDefinitionFamilies, nil
}

// GetTaskDef gets the details about a task definition
func (o *Orchestrator) GetTaskDef(ctx context.Context, cluster, family string) (*TaskDefShowOutput, error) {
	if cluster == "" || family == "" {
		return nil, apierror.New(apierror.ErrBadRequest, "cluster and task def family are required", nil)
	}

	log.Debugf("getting task definition for %s/%s", cluster, family)

	cluOutput, err := o.ECS.GetCluster(ctx, aws.String(cluster))
	if err != nil {
		return nil, err
	}

	tdOutput, tags, err := o.ECS.GetTaskDefinition(ctx, aws.String(family), true)
	if err != nil {
		return nil, err
	}

	for _, t := range tags {
		if aws.StringValue(t.Key) != "spinup:spaceid" {
			continue
		}

		if aws.StringValue(t.Value) != cluster {
			return nil, apierror.New(apierror.ErrNotFound, "taskdef not found in cluster", nil)
		}

		break
	}

	return &TaskDefShowOutput{
		Cluster:        cluOutput,
		TaskDefinition: tdOutput,
		Tags:           tags,
	}, nil
}

func (o *Orchestrator) RunTaskDef(ctx context.Context, cluster, family string, input TaskDefRunOrchestrationInput) (*TaskOutput, error) {
	clu, err := o.ECS.GetCluster(ctx, aws.String(cluster))
	if err != nil {
		return nil, err
	}
	input.Cluster = clu.ClusterArn

	taskdef, _, err := o.ECS.GetTaskDefinition(ctx, aws.String(family), false)
	if err != nil {
		return nil, err
	}
	input.TaskDefinition = taskdef.TaskDefinitionArn

	log.Debugf("got task definition %+v", taskdef)

	input.EnableECSManagedTags = aws.Bool(true)

	if input.CapacityProviderStrategy == nil {
		input.LaunchType = aws.String("FARGATE")
	}

	if input.NetworkConfiguration == nil {
		input.NetworkConfiguration = &ecs.NetworkConfiguration{
			AwsvpcConfiguration: &ecs.AwsVpcConfiguration{
				AssignPublicIp: aws.String(o.DefaultPublic),
				SecurityGroups: aws.StringSlice(o.DefaultSecurityGroups),
				Subnets:        aws.StringSlice(o.DefaultSubnets),
			},
		}
	}

	input.PropagateTags = aws.String("TASK_DEFINITION")

	in := ecs.RunTaskInput(*input)
	out, err := o.ECS.RunTask(ctx, &in)
	if err != nil {
		return nil, err
	}

	output, err := toTaskOutput(out.Tasks, out.Failures)
	if err != nil {
		return nil, err
	}

	return output, nil
}

func (o *Orchestrator) ListTaskDefTasks(ctx context.Context, cluster, taskdef, startedBy string, status []string) ([]string, error) {
	input := ecs.ListTasksInput{
		MaxResults: aws.Int64(100),
		Cluster:    aws.String(cluster),
		Family:     aws.String(taskdef),
	}

	if startedBy != "" {
		input.StartedBy = aws.String(startedBy)
	}

	tasks := []*string{}
	for _, s := range status {
		input.DesiredStatus = aws.String(s)

		out, err := o.ECS.ListTasks(ctx, &input)
		if err != nil {
			return nil, err
		}

		tasks = append(tasks, out...)
	}

	return aws.StringValueSlice(tasks), nil
}
