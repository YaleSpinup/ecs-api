package orchestration

import (
	"context"
	"errors"
	"fmt"

	"github.com/aws/aws-sdk-go/aws"
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

	cluster, rbfunc, err := o.processTaskCluster(ctx, input)
	if err != nil {
		return nil, err
	}
	output.Cluster = cluster
	rollBackTasks = append(rollBackTasks, rbfunc)

	creds, rbfunc, err := o.processTaskRepositoryCredentialsCreate(ctx, input)
	if err != nil {
		return nil, err
	}
	output.Credentials = creds
	rollBackTasks = append(rollBackTasks, rbfunc)

	td, rbfunc, err := o.processTaskTaskDefinitionCreate(ctx, input)
	if err != nil {
		return nil, err
	}
	output.TaskDefinition = td
	rollBackTasks = append(rollBackTasks, rbfunc)

	return output, nil
}

func (o *Orchestrator) DeleteTaskDef(ctx context.Context, input *TaskDefDeleteInput) (*TaskDefDeleteOutput, error) {
	output := TaskDefDeleteOutput{
		Cluster:        input.Cluster,
		TaskDefinition: input.TaskDefinition,
	}

	taskDefinition, err := o.ECS.GetTaskDefinition(ctx, aws.String(input.TaskDefinition))
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
	taskDefinition, err := o.ECS.GetTaskDefinition(ctx, aws.String(revision))
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
