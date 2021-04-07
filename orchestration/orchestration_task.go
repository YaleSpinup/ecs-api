package orchestration

import (
	"context"
	"errors"

	"github.com/aws/aws-sdk-go/service/secretsmanager"

	"github.com/aws/aws-sdk-go/service/ecs"

	log "github.com/sirupsen/logrus"
)

// TaskCreateOrchestrationInput is the input payload for creating a task
type TaskCreateOrchestrationInput struct {
	Cluster        *ecs.CreateClusterInput
	TaskDefinition *ecs.RegisterTaskDefinitionInput
	Credentials    map[string]*secretsmanager.CreateSecretInput
	Tags           []*Tag
}

// TaskCreateOrchestrationOutput is the output payload for a task creation
type TaskCreateOrchestrationOutput struct {
	Cluster *ecs.Cluster
}

// CreateTask orchestrates the creation of a task
func (o *Orchestrator) CreateTask(ctx context.Context, input *TaskCreateOrchestrationInput) (*TaskCreateOrchestrationOutput, error) {
	log.Debugf("got create task orchestration input object:\n %+v", input.TaskDefinition)
	if input.TaskDefinition == nil {
		return nil, errors.New("task definition is required")
	}

	ct, err := cleanTags(o.Org, input.Tags)
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

	output := &TaskCreateOrchestrationOutput{}
	cluster, rbfunc, err := o.processTaskCluster(ctx, input)
	if err != nil {
		return nil, err
	}
	output.Cluster = cluster
	rollBackTasks = append(rollBackTasks, rbfunc)

	return nil, nil
}
