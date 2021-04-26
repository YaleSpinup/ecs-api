package orchestration

import (
	"testing"

	"github.com/aws/aws-sdk-go/service/cloudwatchlogs/cloudwatchlogsiface"
	"github.com/aws/aws-sdk-go/service/resourcegroupstaggingapi/resourcegroupstaggingapiiface"
	log "github.com/sirupsen/logrus"

	"github.com/aws/aws-sdk-go/service/iam/iamiface"
	"github.com/aws/aws-sdk-go/service/secretsmanager/secretsmanageriface"
	"github.com/aws/aws-sdk-go/service/servicediscovery/servicediscoveryiface"

	"github.com/aws/aws-sdk-go/service/ecs/ecsiface"
)

type mockCWLClient struct {
	cloudwatchlogsiface.CloudWatchLogsAPI
	t   *testing.T
	err error
}

type mockECSClient struct {
	ecsiface.ECSAPI
	t   *testing.T
	err error
}

type mockIAMClient struct {
	iamiface.IAMAPI
	t   *testing.T
	err error
}

type mockRGTAClient struct {
	resourcegroupstaggingapiiface.ResourceGroupsTaggingAPIAPI
	t   *testing.T
	err error
}

type mockSDClient struct {
	servicediscoveryiface.ServiceDiscoveryAPI
	t   *testing.T
	err error
}

type mockSMClient struct {
	secretsmanageriface.SecretsManagerAPI
	t   *testing.T
	err error
}

func newMockCWLClient(t *testing.T, err error) cloudwatchlogsiface.CloudWatchLogsAPI {
	m := mockCWLClient{
		t:   t,
		err: err,
	}

	log.Infof("returning mock cloudwatchlogs client %+v", m)

	return &m
}

func newMockECSClient(t *testing.T, err error) ecsiface.ECSAPI {
	m := mockECSClient{
		t:   t,
		err: err,
	}

	log.Infof("returning mock ecs client %+v", m)

	return &m
}

func newMockIAMClient(t *testing.T, err error) iamiface.IAMAPI {
	m := mockIAMClient{
		t:   t,
		err: err,
	}

	log.Infof("returning mock iam client %+v", m)

	return &m
}

func newMockResourceGroupTaggingApiClient(t *testing.T, err error) resourcegroupstaggingapiiface.ResourceGroupsTaggingAPIAPI {
	m := mockRGTAClient{
		t:   t,
		err: err,
	}

	log.Infof("returning mock resourcegrouptaggingapi client %+v", m)

	return &m
}

func newMockSMClient(t *testing.T, err error) secretsmanageriface.SecretsManagerAPI {
	m := mockSMClient{
		t:   t,
		err: err,
	}

	log.Infof("returning mock secretsmanager client %+v", m)

	return &m
}

func newMockSDClient(t *testing.T, err error) servicediscoveryiface.ServiceDiscoveryAPI {
	m := mockSDClient{
		t:   t,
		err: err,
	}

	log.Infof("returning mock servicediscovery client %+v", m)

	return &m
}
