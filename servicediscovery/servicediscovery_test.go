package servicediscovery

import (
	"reflect"
	"testing"

	"github.com/YaleSpinup/ecs-api/common"
)

func TestNewSession(t *testing.T) {
	sd := NewSession(common.Account{})
	to := reflect.TypeOf(sd).String()
	if to != "servicediscovery.ServiceDiscovery" {
		t.Errorf("expected type to be 'servicediscovery.ServiceDiscovery', got %s", to)
	}
}
