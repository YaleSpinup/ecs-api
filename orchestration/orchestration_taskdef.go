package orchestration

import (
	"context"
	"errors"

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
