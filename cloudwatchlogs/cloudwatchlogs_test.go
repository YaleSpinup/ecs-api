package cloudwatchlogs

import (
	"context"
	"reflect"
	"testing"

	"github.com/YaleSpinup/apierror"
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

func (m *mockCWLClient) CreateLogGroupWithContext(ctx context.Context, input *cloudwatchlogs.CreateLogGroupInput, opts ...request.Option) (*cloudwatchlogs.CreateLogGroupOutput, error) {
	if m.err != nil {
		return nil, m.err
	}

	return &cloudwatchlogs.CreateLogGroupOutput{}, nil
}

func (m *mockCWLClient) PutRetentionPolicyWithContext(ctx context.Context, input *cloudwatchlogs.PutRetentionPolicyInput, opts ...request.Option) (*cloudwatchlogs.PutRetentionPolicyOutput, error) {
	if m.err != nil {
		return nil, m.err
	}

	return &cloudwatchlogs.PutRetentionPolicyOutput{}, nil
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
		t.Error("expected error, got nil")
	} else {
		if aerr, ok := errors.Cause(err).(apierror.Error); ok {
			t.Logf("got apierror '%s'", aerr)
		} else {
			t.Errorf("expected error to be an apierror.Error, got %s", err)
		}
	}
}

func TestCreateLogGroup(t *testing.T) {
	client := CloudWatchLogs{Service: newmockCWLClient(t, nil)}
	if err := client.CreateLogGroup(context.TODO(), &cloudwatchlogs.CreateLogGroupInput{
		LogGroupName: aws.String("log-group-01"),
	}); err != nil {
		t.Errorf("expected nil error, got %s", err)
	}

	if err := client.CreateLogGroup(context.TODO(), nil); err == nil {
		t.Errorf("expected err for nil input")
	}

	client = CloudWatchLogs{Service: newmockCWLClient(t, awserr.New(cloudwatchlogs.ErrCodeInvalidOperationException, "The operation is not valid on the specified resource.", nil))}
	if err := client.CreateLogGroup(context.TODO(), &cloudwatchlogs.CreateLogGroupInput{}); err == nil {
		t.Error("expected error, got nil")
	} else {
		if aerr, ok := errors.Cause(err).(apierror.Error); ok {
			t.Logf("got apierror '%s'", aerr)
		} else {
			t.Errorf("expected error to be an apierror.Error, got %s", err)
		}
	}
}

func TestUpdateRetention(t *testing.T) {
	client := CloudWatchLogs{Service: newmockCWLClient(t, nil)}
	if err := client.UpdateRetention(context.TODO(), &cloudwatchlogs.PutRetentionPolicyInput{
		LogGroupName:    aws.String("log-group-01"),
		RetentionInDays: aws.Int64(int64(365)),
	}); err != nil {
		t.Errorf("expected nil error, got %s", err)
	}

	if err := client.UpdateRetention(context.TODO(), nil); err == nil {
		t.Errorf("expected err for nil input")
	}

	client = CloudWatchLogs{Service: newmockCWLClient(t, awserr.New(cloudwatchlogs.ErrCodeInvalidOperationException, "The operation is not valid on the specified resource.", nil))}
	if err := client.UpdateRetention(context.TODO(), &cloudwatchlogs.PutRetentionPolicyInput{
		LogGroupName:    aws.String("log-group-01"),
		RetentionInDays: aws.Int64(int64(365)),
	}); err == nil {
		t.Error("expected error, got nil")
	} else {
		if aerr, ok := errors.Cause(err).(apierror.Error); ok {
			t.Logf("got apierror '%s'", aerr)
		} else {
			t.Errorf("expected error to be an apierror.Error, got %s", err)
		}
	}
}
