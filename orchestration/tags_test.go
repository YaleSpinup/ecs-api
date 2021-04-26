package orchestration

import (
	"reflect"
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/secretsmanager"
)

func TestSecretsmanagerTags(t *testing.T) {
	var tests = []struct {
		input  []*Tag
		output []*secretsmanager.Tag
	}{
		{
			input: []*Tag{
				{
					Key:   aws.String("foo"),
					Value: aws.String("bar"),
				},
			},
			output: []*secretsmanager.Tag{
				{
					Key:   aws.String("foo"),
					Value: aws.String("bar"),
				},
			},
		},
		{
			input: []*Tag{
				{
					Key:   aws.String("foo"),
					Value: aws.String("bar"),
				},
				{
					Key:   aws.String("spinup:org"),
					Value: aws.String("someOtherOrg"),
				},
			},
			output: []*secretsmanager.Tag{
				{
					Key:   aws.String("foo"),
					Value: aws.String("bar"),
				},
				{
					Key:   aws.String("spinup:org"),
					Value: aws.String("someOtherOrg"),
				},
			},
		},
	}

	for _, test := range tests {
		out := secretsmanagerTags(test.input)

		if !reflect.DeepEqual(test.output, out) {
			t.Errorf("expected %+v, got %+v", test.output, out)
		}

		for _, tag := range test.output {
			exists := false
			t.Logf("testing for test tag key: %v, value: %v", tag.Key, tag.Value)

			for _, otag := range out {
				t.Logf("testing output tag key: %v, value: %v", otag.Key, otag.Value)
				if aws.StringValue(otag.Key) == aws.StringValue(tag.Key) {
					value := aws.StringValue(tag.Value)
					ovalue := aws.StringValue(otag.Value)
					if value != ovalue {
						t.Errorf("expected tag %s value to be %s, got %s", aws.StringValue(tag.Key), value, ovalue)
					}
					exists = true
					break
				}
			}

			if !exists {
				t.Errorf("expected tag %+v to exist", tag)
			}
		}
	}

}
