package orchestration

import (
	"testing"

	"github.com/YaleSpinup/ecs-api/cloudwatchlogs"
	"github.com/YaleSpinup/ecs-api/ecs"
	"github.com/YaleSpinup/ecs-api/iam"
	"github.com/YaleSpinup/ecs-api/resourcegroupstaggingapi"
	"github.com/YaleSpinup/ecs-api/secretsmanager"
	"github.com/YaleSpinup/ecs-api/servicediscovery"
	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
)

func newMockOrchestrator(t *testing.T, org string, cwlerr, ecserr, iamerr, rgtaerr, smerr, sderr error) *Orchestrator {
	o := Orchestrator{
		CloudWatchLogs:           cloudwatchlogs.CloudWatchLogs{Service: newMockCWLClient(t, cwlerr)},
		ECS:                      ecs.ECS{Service: newMockECSClient(t, ecserr)},
		IAM:                      iam.IAM{Service: newMockIAMClient(t, iamerr)},
		ResourceGroupsTaggingAPI: resourcegroupstaggingapi.ResourceGroupsTaggingAPI{Service: newMockResourceGroupTaggingApiClient(t, rgtaerr)},
		SecretsManager:           secretsmanager.SecretsManager{Service: newMockSMClient(t, smerr)},
		ServiceDiscovery:         servicediscovery.ServiceDiscovery{Service: newMockSDClient(t, sderr)},
		Token:                    uuid.New().String(),
		Org:                      org,
	}

	log.Infof("Returning new mock orchestrator: %+v", o)

	return &o
}
