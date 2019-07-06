package secretsmanager

import (
	"reflect"
	"testing"

	"github.com/YaleSpinup/ecs-api/common"
	"github.com/aws/aws-sdk-go/service/secretsmanager/secretsmanageriface"
)

// mockSecretsManagerClient is a fake secretsmanager client
type mockSecretsManagerClient struct {
	secretsmanageriface.SecretsManagerAPI
	t   *testing.T
	err error
}

func newmockSecretsManagerClient(t *testing.T, err error) secretsmanageriface.SecretsManagerAPI {
	return &mockSecretsManagerClient{
		t:   t,
		err: err,
	}
}

func TestNewSession(t *testing.T) {
	e := NewSession(common.Account{})
	to := reflect.TypeOf(e).String()
	if to != "secretsmanager.SecretsManager" {
		t.Errorf("expected type to be 'secretsmanager.SecretsManager', got %s", to)
	}
}
