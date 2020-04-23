package elbv2

import (
	"context"
	"reflect"
	"testing"

	"github.com/YaleSpinup/ecs-api/common"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/request"
	"github.com/aws/aws-sdk-go/service/elbv2"
	"github.com/aws/aws-sdk-go/service/elbv2/elbv2iface"
)

// mockELBV2APIClient is a fake resourcegroupstaggingapi client
type mockELBV2APIClient struct {
	elbv2iface.ELBV2API
	t   *testing.T
	err error
}

func newmockELBV2APIClient(t *testing.T, err error) elbv2iface.ELBV2API {
	return &mockELBV2APIClient{
		t:   t,
		err: err,
	}
}

func TestNewSession(t *testing.T) {
	e := NewSession(common.Account{})
	to := reflect.TypeOf(e).String()
	if to != "elbv2.ELBV2API" {
		t.Errorf("expected type to be 'elbv2.ELBV2API', got %s", to)
	}
}

var testTargetGroups = []*elbv2.TargetGroup{
	{
		HealthCheckEnabled:         aws.Bool(true),
		HealthCheckIntervalSeconds: aws.Int64(int64(30)),
		HealthCheckPath:            aws.String("/ping"),
		HealthCheckPort:            aws.String("traffic-port"),
		HealthCheckProtocol:        aws.String("HTTP"),
		HealthCheckTimeoutSeconds:  aws.Int64(int64(5)),
		HealthyThresholdCount:      aws.Int64(int64(5)),
		Matcher: &elbv2.Matcher{
			HttpCode: aws.String("200"),
		},
		Port:                    aws.Int64(int64(8080)),
		Protocol:                aws.String("HTTP"),
		TargetGroupArn:          aws.String("arn:aws:elasticloadbalancing:us-east-1:123456789:targetgroup/test-tg-1/123456789"),
		TargetGroupName:         aws.String("test-tg-1"),
		TargetType:              aws.String("instance"),
		UnhealthyThresholdCount: aws.Int64(int64(2)),
		VpcId:                   aws.String("vpc-123456789"),
	},
	{
		HealthCheckEnabled:         aws.Bool(true),
		HealthCheckIntervalSeconds: aws.Int64(int64(30)),
		HealthCheckPath:            aws.String("/ping"),
		HealthCheckPort:            aws.String("traffic-port"),
		HealthCheckProtocol:        aws.String("HTTP"),
		HealthCheckTimeoutSeconds:  aws.Int64(int64(5)),
		HealthyThresholdCount:      aws.Int64(int64(5)),
		Matcher: &elbv2.Matcher{
			HttpCode: aws.String("200"),
		},
		Port:                    aws.Int64(int64(80)),
		Protocol:                aws.String("HTTP"),
		TargetGroupArn:          aws.String("arn:aws:elasticloadbalancing:us-east-1:123456789:targetgroup/test-tg-2/123456789"),
		TargetGroupName:         aws.String("test-tg-2"),
		TargetType:              aws.String("instance"),
		UnhealthyThresholdCount: aws.Int64(int64(2)),
		VpcId:                   aws.String("vpc-123456789"),
	},
}

func (m *mockELBV2APIClient) DescribeTargetGroupsWithContext(ctx context.Context, input *elbv2.DescribeTargetGroupsInput, opts ...request.Option) (*elbv2.DescribeTargetGroupsOutput, error) {
	if m.err != nil {
		return nil, m.err
	}

	return &elbv2.DescribeTargetGroupsOutput{
		TargetGroups: testTargetGroups,
	}, nil
}

func TestGetTargetGroups(t *testing.T) {
	e := ELBV2API{Service: newmockELBV2APIClient(t, nil)}
	out, err := e.GetTargetGroups(context.TODO(), []string{"foo"})
	if err != nil {
		t.Errorf("expected nil error, got %s", err)
	}

	if !reflect.DeepEqual(testTargetGroups, out) {
		t.Errorf("expected %+v, got %+v", testTargetGroups, out)
	}

	_, err = e.GetTargetGroups(context.TODO(), []string{})
	if err == nil {
		t.Error("expected error, got nil")
	}
}
