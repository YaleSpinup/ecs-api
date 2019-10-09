package ssm

import (
	"reflect"
	"testing"

	"github.com/YaleSpinup/ecs-api/common"
	"github.com/aws/aws-sdk-go/service/ssm/ssmiface"
)

// mockSSMClient is a fake ssm client
type mockSSMClient struct {
	ssmiface.SSMAPI
	t   *testing.T
	err error
}

func newmockSSMClient(t *testing.T, err error) ssmiface.SSMAPI {
	return &mockSSMClient{
		t:   t,
		err: err,
	}
}

func TestNewSession(t *testing.T) {
	e := NewSession(common.Account{})
	to := reflect.TypeOf(e).String()
	if to != "ssm.SSM" {
		t.Errorf("expected type to be 'ssm.SSM', got %s", to)
	}
}
