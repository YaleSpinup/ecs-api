package resourcegroupstaggingapi

import (
	"context"
	"reflect"
	"testing"

	"github.com/YaleSpinup/ecs-api/common"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/request"
	"github.com/aws/aws-sdk-go/service/resourcegroupstaggingapi"
	"github.com/aws/aws-sdk-go/service/resourcegroupstaggingapi/resourcegroupstaggingapiiface"
)

// mockResourceGroupsTaggingAPIClient is a fake resourcegroupstaggingapi client
type mockResourceGroupsTaggingAPIClient struct {
	resourcegroupstaggingapiiface.ResourceGroupsTaggingAPIAPI
	t   *testing.T
	err error
}

func newmockResourceGroupsTaggingAPIClient(t *testing.T, err error) resourcegroupstaggingapiiface.ResourceGroupsTaggingAPIAPI {
	return &mockResourceGroupsTaggingAPIClient{
		t:   t,
		err: err,
	}
}

func TestNewSession(t *testing.T) {
	e := NewSession(common.Account{})
	to := reflect.TypeOf(e).String()
	if to != "resourcegroupstaggingapi.ResourceGroupsTaggingAPI" {
		t.Errorf("expected type to be 'resourcegroupstaggingapi.ResourceGroupsTaggingAPI', got %s", to)
	}
}

type tag struct {
	key   string
	value string
}

type testResource struct {
	resourceType string
	tags         []tag
	arn          string
}

var testResources = []testResource{
	{
		resourceType: "ec2:instance",
		tags: []tag{
			{
				key:   "spinup:org",
				value: "foobar",
			},
			{
				key:   "spinup:spaceid",
				value: "123",
			},
		},
		arn: "arn:aws:ec2:us-east-1:1234567890:instance/i-0987654321",
	},
	{
		resourceType: "elasticloadbalancing:targetgroup",
		tags: []tag{
			{
				key:   "spinup:org",
				value: "foobar",
			},
			{
				key:   "spinup:spaceid",
				value: "123",
			},
		},
		arn: "arn:aws:elasticloadbalancing:us-east-1:1234567890:targetgroup/testtg123/0987654321",
	},
	{
		resourceType: "elasticloadbalancing:targetgroup",
		tags: []tag{
			{
				key:   "spinup:org",
				value: "foobar",
			},
			{
				key:   "spinup:spaceid",
				value: "321",
			},
		},
		arn: "arn:aws:elasticloadbalancing:us-east-1:1234567890:targetgroup/testtg321/0987654321",
	},
}

func (m *mockResourceGroupsTaggingAPIClient) GetResourcesWithContext(ctx context.Context, input *resourcegroupstaggingapi.GetResourcesInput, opts ...request.Option) (*resourcegroupstaggingapi.GetResourcesOutput, error) {
	if m.err != nil {
		return nil, m.err
	}

	resourceList := []*resourcegroupstaggingapi.ResourceTagMapping{}
	for _, r := range testResources {
		if len(input.ResourceTypeFilters) > 0 {
			var typeMatch bool
			for _, t := range input.ResourceTypeFilters {
				if aws.StringValue(t) == r.resourceType {
					typeMatch = true
					break
				}
			}

			if !typeMatch {
				continue
			}
		}

		matches := true
		for _, filter := range input.TagFilters {
			innerMatch := func() bool {
				m.t.Logf("processing tagfilter %+v", filter)
				for _, rt := range r.tags {
					if aws.StringValue(filter.Key) == rt.key {
						m.t.Logf("tag keys match for %s (%s = %s)", r.arn, rt.key, aws.StringValue(filter.Key))
						if len(filter.Values) == 0 {
							m.t.Logf("appending %s to the list, keys match (%s = %s) and no value specified", r.arn, rt.key, aws.StringValue(filter.Key))
							return true
						}

						for _, value := range aws.StringValueSlice(filter.Values) {
							if value == rt.value {
								m.t.Logf("appending %s to the list, keys match (%s = %s) and value matches (%s = %s)", r.arn, rt.key, aws.StringValue(filter.Key), value, rt.value)
								return true
							}
						}
					}
				}
				m.t.Logf("returning false for %s", r.arn)
				return false
			}()

			if !innerMatch {
				matches = false
			}
		}

		if matches {
			m.t.Logf("resource %s matches", r.arn)
			resourceList = append(resourceList, &resourcegroupstaggingapi.ResourceTagMapping{
				ResourceARN: aws.String(r.arn),
			})
		}
	}

	return &resourcegroupstaggingapi.GetResourcesOutput{
		ResourceTagMappingList: resourceList,
	}, nil
}

func TestGetResourcesWithTags(t *testing.T) {
	r := ResourceGroupsTaggingAPI{Service: newmockResourceGroupsTaggingAPIClient(t, nil)}
	filters := []*TagFilter{
		{
			Key:   "spinup:org",
			Value: []string{"foobar"},
		},
		{
			Key:   "spinup:spaceid",
			Value: []string{"123"},
		},
	}
	out, err := r.GetResourcesWithTags(context.TODO(), []string{"elasticloadbalancing:targetgroup"}, filters)
	if err != nil {
		t.Errorf("expected nil error, got %s", err)
	}

	expected := []string{"arn:aws:elasticloadbalancing:us-east-1:1234567890:targetgroup/testtg123/0987654321"}
	if !reflect.DeepEqual(expected, out) {
		t.Errorf("expected %+v, got %+v", expected, out)
	}

	out, err = r.GetResourcesWithTags(context.TODO(), []string{}, filters)
	if err != nil {
		t.Errorf("expected nil error, got %s", err)
	}

	expected = []string{
		"arn:aws:ec2:us-east-1:1234567890:instance/i-0987654321",
		"arn:aws:elasticloadbalancing:us-east-1:1234567890:targetgroup/testtg123/0987654321",
	}
	if !reflect.DeepEqual(expected, out) {
		t.Errorf("expected %+v, got %+v", expected, out)
	}
}
