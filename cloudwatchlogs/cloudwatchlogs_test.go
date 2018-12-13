package cloudwatchlogs

import (
	"reflect"
	"testing"

	"github.com/YaleSpinup/ecs-api/common"
)

func TestNewSession(t *testing.T) {
	cw := NewSession(common.Account{})
	to := reflect.TypeOf(cw).String()
	if to != "cloudwatchlogs.CloudWatchLogs" {
		t.Errorf("expected type to be 'cloudwatchlogs.CloudWatchLogs', got %s", to)
	}
}
