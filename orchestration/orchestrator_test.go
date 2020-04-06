package orchestration

import (
	"testing"

	"github.com/YaleSpinup/ecs-api/cloudwatchlogs"
	"github.com/YaleSpinup/ecs-api/ecs"
	"github.com/YaleSpinup/ecs-api/iam"
	"github.com/YaleSpinup/ecs-api/secretsmanager"
	"github.com/YaleSpinup/ecs-api/servicediscovery"
	"github.com/google/uuid"
)

func newMockOrchestrator(t *testing.T, org string, cwlerr, ecserr, iamerr, smerr, sderr error) *Orchestrator {
	return &Orchestrator{
		CloudWatchLogs:   cloudwatchlogs.CloudWatchLogs{Service: newMockCWLClient(t, cwlerr)},
		ECS:              ecs.ECS{Service: newMockECSClient(t, ecserr)},
		IAM:              iam.IAM{Service: newMockIAMClient(t, iamerr)},
		SecretsManager:   secretsmanager.SecretsManager{Service: newMockSMClient(t, smerr)},
		ServiceDiscovery: servicediscovery.ServiceDiscovery{Service: newMockSDClient(t, sderr)},
		Token:            uuid.New().String(),
		Org:              org,
	}
}
