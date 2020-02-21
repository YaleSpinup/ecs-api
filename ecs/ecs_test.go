package ecs

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"github.com/YaleSpinup/ecs-api/common"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/request"
	"github.com/aws/aws-sdk-go/service/ecs"
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

var testResourceTags = []*ecs.Tag{
	&ecs.Tag{
		Key:   aws.String("foo"),
		Value: aws.String("bar"),
	},
	&ecs.Tag{
		Key:   aws.String("fiz"),
		Value: aws.String("biz"),
	},
	&ecs.Tag{
		Key:   aws.String("fuz"),
		Value: aws.String("boz"),
	},
}

func (m *mockECSClient) ListTagsForResourceWithContext(ctx aws.Context, input *ecs.ListTagsForResourceInput, opts ...request.Option) (*ecs.ListTagsForResourceOutput, error) {
	if m.err != nil {
		return nil, m.err
	}

	if aws.StringValue(input.ResourceArn) == "myarn" {
		return &ecs.ListTagsForResourceOutput{
			Tags: testResourceTags,
		}, nil
	}

	return nil, errors.New("Failed get test resource tags")
}

func TestNewSession(t *testing.T) {
	e := NewSession(common.Account{})
	to := reflect.TypeOf(e).String()
	if to != "ecs.ECS" {
		t.Errorf("expected type to be 'ecs.ECS', got %s", to)
	}
}

func TestListTags(t *testing.T) {
	client := ECS{Service: &mockECSClient{t: t}}
	tags, err := client.ListTags(context.TODO(), "myarn")
	if err != nil {
		t.Fatal("expected no error from listing tags, got", err)
	}
	t.Log("got list tags response", tags)
	if !reflect.DeepEqual(testResourceTags, tags) {
		t.Fatalf("Expected %+v\nGot %+v", testResourceTags, tags)
	}

	_, err = client.ListTags(context.TODO(), "")
	if err == nil {
		t.Fatal("expected error from empty arn, got nil")
	}

	client = ECS{
		Service: &mockECSClient{
			t:   t,
			err: awserr.New(ecs.ErrCodeUpdateInProgressException, "wont fix", nil),
		},
	}
	_, err = client.ListTags(context.TODO(), "myarn")
	if err == nil {
		t.Fatal("expected error from list tags, got nil")
	}
}
