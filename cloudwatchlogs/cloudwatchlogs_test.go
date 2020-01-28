package cloudwatchlogs

import (
	"context"
	"reflect"
	"testing"

	"github.com/YaleSpinup/ecs-api/apierror"
	"github.com/YaleSpinup/ecs-api/common"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/request"
	"github.com/aws/aws-sdk-go/service/cloudwatchlogs"
	"github.com/aws/aws-sdk-go/service/cloudwatchlogs/cloudwatchlogsiface"
	"github.com/pkg/errors"
)

// mockCWLClient is a fake cloudwatchlogs client
type mockCWLClient struct {
	cloudwatchlogsiface.CloudWatchLogsAPI
	t   *testing.T
	err error
}

func newmockCWLClient(t *testing.T, err error) cloudwatchlogsiface.CloudWatchLogsAPI {
	return &mockCWLClient{
		t:   t,
		err: err,
	}
}

func (m *mockCWLClient) GetLogEventsWithContext(ctx context.Context, input *cloudwatchlogs.GetLogEventsInput, opts ...request.Option) (*cloudwatchlogs.GetLogEventsOutput, error) {
	if m.err != nil {
		return nil, m.err
	}

	return &cloudwatchlogs.GetLogEventsOutput{}, nil
}

func TestNewSession(t *testing.T) {
	cw := NewSession(common.Account{})
	to := reflect.TypeOf(cw).String()
	if to != "cloudwatchlogs.CloudWatchLogs" {
		t.Errorf("expected type to be 'cloudwatchlogs.CloudWatchLogs', got %s", to)
	}
}

func TestGetLogEvents(t *testing.T) {
	client := CloudWatchLogs{Service: newmockCWLClient(t, nil)}
	expected := &cloudwatchlogs.GetLogEventsOutput{}
	out, err := client.GetLogEvents(context.TODO(), &cloudwatchlogs.GetLogEventsInput{
		LogGroupName:  aws.String("clu0"),
		LogStreamName: aws.String("logStream0"),
	})
	if err != nil {
		t.Errorf("expected nil error, got %s", err)
	}

	if !reflect.DeepEqual(out, expected) {
		t.Errorf("expected %+v, got %+v", expected, out)
	}

	if _, err = client.GetLogEvents(context.TODO(), nil); err == nil {
		t.Errorf("expected err for nil input")
	}

	client = CloudWatchLogs{Service: newmockCWLClient(t, awserr.New(cloudwatchlogs.ErrCodeInvalidOperationException, "The operation is not valid on the specified resource.", nil))}
	_, err = client.GetLogEvents(context.TODO(), &cloudwatchlogs.GetLogEventsInput{})
	if err == nil {
		t.Errorf("expected nil error, got %s", err)
	} else {
		if aerr, ok := errors.Cause(err).(apierror.Error); ok {
			t.Logf("got apierror '%s'", aerr)
		} else {
			t.Errorf("expected error to be an apierror.Error, got %s", err)
		}
	}
}
