package orchestration

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/YaleSpinup/ecs-api/resourcegroupstaggingapi"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/arn"
	"github.com/aws/aws-sdk-go/aws/awsutil"
	"github.com/aws/aws-sdk-go/service/iam"
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
	Cluster        *ecs.Cluster
	TaskDefinition *ecs.TaskDefinition
	Credentials    map[string]interface{}
	Tags           []*Tag
}

// TaskDefDeleteInput encapsulates a request to delete a taskdef with optional recursion
type TaskDefDeleteInput struct {
	Cluster        string
	TaskDefinition string
	Recursive      bool
}

// TaskDefDeleteOutput is the orchestration response for taskdef deletion
type TaskDefDeleteOutput struct {
	Cluster        string
	TaskDefinition string
}

type TaskDefShowOutput struct {
	Cluster        *ecs.Cluster
	TaskDefinition *ecs.TaskDefinition
	Tags           []*ecs.Tag
}

// CreateTask orchestrates the creation of a task.  It creates a cluster, creates repository credrentials in
// secretsmanager, and then creates the task definition.
func (o *Orchestrator) CreateTaskDef(ctx context.Context, input *TaskDefCreateOrchestrationInput) (*TaskDefCreateOrchestrationOutput, error) {
	log.Debugf("got create task orchestration input object:\n %+v", input.TaskDefinition)
	if input.TaskDefinition == nil {
		return nil, errors.New("task definition is required")
	}

	spaceid := aws.StringValue(input.Cluster.ClusterName)

	ct, err := cleanTags(o.Org, spaceid, input.Tags)
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

func (o *Orchestrator) UpdateTaskDef(ctx context.Context, cluster, family string, input *TaskDefUpdateOrchestrationInput) (*TaskDefUpdateOrchestrationOutput, error) {
	output := &TaskDefUpdateOrchestrationOutput{}

	clu, err := o.ECS.GetCluster(ctx, aws.String(cluster))
	if err != nil {
		return nil, err
	}
	input.ClusterName = cluster
	output.Cluster = clu

	taskdef, tags, err := o.ECS.GetTaskDefinition(ctx, aws.String(family))
	if err != nil {
		return nil, err
	}

	log.Debugf("got task definition %+v", taskdef)

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

	// if the input tags are passed, clean them and use them, otherwise set to the active tags
	if input.Tags != nil {
		ct, err := cleanTags(o.Org, cluster, input.Tags)
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

func (o *Orchestrator) DeleteTaskDef(ctx context.Context, input *TaskDefDeleteInput) (*TaskDefDeleteOutput, error) {
	output := TaskDefDeleteOutput{
		Cluster:        input.Cluster,
		TaskDefinition: input.TaskDefinition,
	}

	taskDefinition, _, err := o.ECS.GetTaskDefinition(ctx, aws.String(input.TaskDefinition))
	if err != nil {
		return nil, err
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

	// TODO cleanup cluster

	return &output, nil
}

// deleteTaskDefinitionRevision deletes a task definition revision and associated secretsmanager secrets.  It keeps track
// of deleted secrets through the deleteCredentials map
func (o *Orchestrator) deleteTaskDefinitionRevision(ctx context.Context, revision string, deletedCredentials map[string]struct{}) []error {
	var errors []error
	taskDefinition, _, err := o.ECS.GetTaskDefinition(ctx, aws.String(revision))
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

func (o *Orchestrator) ListTaskDefs(ctx context.Context, cluster string) ([]string, error) {
	log.Infof("listing task definitions in cluster '%s'", cluster)

	tagFilters := []*resourcegroupstaggingapi.TagFilter{
		{
			Key:   "spinup:org",
			Value: []string{o.Org},
		},
		{
			Key:   "spinup:category",
			Value: []string{"container-taskdef"},
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

func (o *Orchestrator) GetTaskDef(ctx context.Context, cluster, family string) (*TaskDefShowOutput, error) {
	if cluster == "" || family == "" {
		return nil, errors.New("cluster and task def family are required")
	}

	log.Debugf("getting task definition for %s/%s", cluster, family)

	cluOutput, err := o.ECS.GetCluster(ctx, aws.String(cluster))
	if err != nil {
		return nil, err
	}

	tdOutput, tags, err := o.ECS.GetTaskDefinition(ctx, aws.String(family))
	if err != nil {
		return nil, err
	}

	return &TaskDefShowOutput{
		Cluster:        cluOutput,
		TaskDefinition: tdOutput,
		Tags:           tags,
	}, nil
}

func (o *Orchestrator) processTaskDefTagsUpdate(ctx context.Context, active *TaskDefUpdateOrchestrationOutput, tags []*Tag) error {
	log.Debugf("processing tags update with tags list %s", awsutil.Prettify(tags))

	// tag all of our resources
	taskDefTags := make([]*ecs.Tag, 0, len(tags))
	smTags := make([]*secretsmanager.Tag, 0, len(tags))
	clusterTags := []*ecs.Tag{}
	roleTags := []*iam.Tag{}
	for _, t := range tags {
		taskDefTags = append(taskDefTags, &ecs.Tag{Key: t.Key, Value: t.Value})
		smTags = append(smTags, &secretsmanager.Tag{Key: t.Key, Value: t.Value})

		// some services shouldn't be categorized
		if aws.StringValue(t.Key) != "spinup:category" {
			clusterTags = append(clusterTags, &ecs.Tag{Key: t.Key, Value: t.Value})
			roleTags = append(roleTags, &iam.Tag{Key: t.Key, Value: t.Value})
		}
	}

	// tag task definition
	if err := o.ECS.TagResource(ctx, &ecs.TagResourceInput{
		ResourceArn: active.TaskDefinition.TaskDefinitionArn,
		Tags:        taskDefTags,
	}); err != nil {
		return err
	}

	// tag cluster
	if err := o.ECS.TagResource(ctx, &ecs.TagResourceInput{
		ResourceArn: active.Cluster.ClusterArn,
		Tags:        clusterTags,
	}); err != nil {
		return err
	}

	// tag secrets
	for _, containerDef := range active.TaskDefinition.ContainerDefinitions {
		repositoryCredentials := containerDef.RepositoryCredentials
		if repositoryCredentials != nil && repositoryCredentials.CredentialsParameter != nil {
			credentialsArn := aws.StringValue(repositoryCredentials.CredentialsParameter)
			if err := o.SecretsManager.UpdateSecretTags(ctx, credentialsArn, smTags); err != nil {
				return err
			}
		}
	}

	// get the ecs task execution role arn from the active task definition
	ecsTaskExecutionRoleArn, err := arn.Parse(aws.StringValue(active.TaskDefinition.ExecutionRoleArn))
	if err != nil {
		return err
	}

	// determine the ecs task execution role name from the arn
	ecsTaskExecutionRoleName := ecsTaskExecutionRoleArn.Resource[strings.LastIndex(ecsTaskExecutionRoleArn.Resource, "/")+1:]

	// tag the ecs task execution role
	if err := o.IAM.TagRole(ctx, ecsTaskExecutionRoleName, roleTags); err != nil {
		return err
	}

	// set the active tasks to the input tasks for output
	active.Tags = tags
	// end tagging

	return err
}
