package orchestration

import (
	"reflect"
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/iam"
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

func Test_sharedResourceTags(t *testing.T) {
	type args struct {
		name string
		tags []*Tag
	}
	tests := []struct {
		name string
		args args
		want map[string]*string
	}{
		{
			name: "skip category",
			args: args{
				name: "get-a-clu",
				tags: []*Tag{
					{
						Key:   aws.String("spinup:category"),
						Value: aws.String("thingy"),
					},
				},
			},
			want: map[string]*string{"Name": aws.String("get-a-clu")},
		},
		{
			name: "override name",
			args: args{
				name: "get-a-clu",
				tags: []*Tag{
					{
						Key:   aws.String("Name"),
						Value: aws.String("thingy1"),
					},
					{
						Key:   aws.String("name"),
						Value: aws.String("thingy2"),
					},
				},
			},
			want: map[string]*string{"Name": aws.String("get-a-clu")},
		},
		{
			name: "tag list",
			args: args{
				name: "get-a-clu",
				tags: []*Tag{
					{
						Key:   aws.String("Name"),
						Value: aws.String("thingy1"),
					},
					{
						Key:   aws.String("some"),
						Value: aws.String("thingy2"),
					},
					{
						Key:   aws.String("other"),
						Value: aws.String("thingy3"),
					},
				},
			},
			want: map[string]*string{
				"Name":  aws.String("get-a-clu"),
				"some":  aws.String("thingy2"),
				"other": aws.String("thingy3"),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := sharedResourceTags(tt.args.name, tt.args.tags); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("sharedResourceTags() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_specificResourceTags(t *testing.T) {
	type args struct {
		tags []*Tag
	}
	tests := []struct {
		name string
		args args
		want map[string]*string
	}{
		{
			name: "category",
			args: args{
				tags: []*Tag{
					{
						Key:   aws.String("spinup:category"),
						Value: aws.String("thingy"),
					},
				},
			},
			want: map[string]*string{"spinup:category": aws.String("thingy")},
		},
		{
			name: "name",
			args: args{
				tags: []*Tag{
					{
						Key:   aws.String("Name"),
						Value: aws.String("thingy1"),
					},
				},
			},
			want: map[string]*string{"Name": aws.String("thingy1")},
		},
		{
			name: "tag list",
			args: args{
				tags: []*Tag{
					{
						Key:   aws.String("Name"),
						Value: aws.String("thingy1"),
					},
					{
						Key:   aws.String("some"),
						Value: aws.String("thingy2"),
					},
					{
						Key:   aws.String("other"),
						Value: aws.String("thingy3"),
					},
				},
			},
			want: map[string]*string{
				"Name":  aws.String("thingy1"),
				"some":  aws.String("thingy2"),
				"other": aws.String("thingy3"),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := specificResourceTags(tt.args.tags); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("specificResourceTags() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_roleTags(t *testing.T) {
	type args struct {
		name string
		tags []*Tag
	}
	tests := []struct {
		name string
		args args
		want []*iam.Tag
	}{
		{
			name: "skip category",
			args: args{
				name: "bestRole",
				tags: []*Tag{
					{
						Key:   aws.String("spinup:category"),
						Value: aws.String("thingy"),
					},
				},
			},
			want: []*iam.Tag{
				{
					Key:   aws.String("Name"),
					Value: aws.String("bestRole"),
				},
			},
		},
		{
			name: "override name",
			args: args{
				name: "bestRole",
				tags: []*Tag{
					{
						Key:   aws.String("Name"),
						Value: aws.String("thingy1"),
					},
				},
			},
			want: []*iam.Tag{
				{
					Key:   aws.String("Name"),
					Value: aws.String("bestRole"),
				},
			},
		},
		{
			name: "tag list",
			args: args{
				name: "bestRole",
				tags: []*Tag{
					{
						Key:   aws.String("Name"),
						Value: aws.String("thingy1"),
					},
					{
						Key:   aws.String("name"),
						Value: aws.String("littlethingy1"),
					},
					{
						Key:   aws.String("some"),
						Value: aws.String("thingy2"),
					},
					{
						Key:   aws.String("other"),
						Value: aws.String("thingy3"),
					},
				},
			},
			want: []*iam.Tag{
				{
					Key:   aws.String("Name"),
					Value: aws.String("bestRole"),
				},
				{
					Key:   aws.String("some"),
					Value: aws.String("thingy2"),
				},
				{
					Key:   aws.String("other"),
					Value: aws.String("thingy3"),
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := roleTags(tt.args.name, tt.args.tags); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("roleTags() = %v, want %v", got, tt.want)
			}
		})
	}
}
