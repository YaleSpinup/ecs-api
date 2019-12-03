package ecs

import (
	"reflect"
	"testing"

	"github.com/YaleSpinup/ecs-api/common"
	"github.com/aws/aws-sdk-go/service/ecs/ecsiface"
)

// mockECSClient is a fake ecs client
type mockECSClient struct {
	ecsiface.ECSAPI
	t   *testing.T
	err error
}

func newmockECSClient(t *testing.T, err error) ecsiface.ECSAPI {
	return &mockECSClient{
		t:   t,
		err: err,
	}
}

func TestNewSession(t *testing.T) {
	e := NewSession(common.Account{})
	to := reflect.TypeOf(e).String()
	if to != "ecs.ECS" {
		t.Errorf("expected type to be 'ecs.ECS', got %s", to)
	}
}
