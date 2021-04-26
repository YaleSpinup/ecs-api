package orchestration

import (
	"errors"
	"fmt"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ecs"
	"github.com/aws/aws-sdk-go/service/secretsmanager"
)

// ecsTags takes a slice of tags and converts them to ECS tags
func ecsTags(input []*Tag) []*ecs.Tag {
	et := make([]*ecs.Tag, len(input))
	for i, t := range input {
		et[i] = &ecs.Tag{Key: t.Key, Value: t.Value}
	}
	return et
}

func secretsmanagerTags(tags []*Tag) []*secretsmanager.Tag {
	st := make([]*secretsmanager.Tag, len(tags))
	for i, t := range tags {
		st[i] = &secretsmanager.Tag{Key: t.Key, Value: t.Value}
	}
	return st
}

// cleanTags cleanses the tags input and ensures spinup:org and spinup:spaceid are set correctly
func cleanTags(org, spaceid string, tags []*Tag) ([]*Tag, error) {
	cleanTags := []*Tag{
		{
			Key:   aws.String("spinup:org"),
			Value: aws.String(org),
		},
		{
			Key:   aws.String("spinup:spaceid"),
			Value: aws.String(spaceid),
		},
	}

	for _, t := range tags {
		if aws.StringValue(t.Key) != "spinup:org" && aws.StringValue(t.Key) != "yale:org" && aws.StringValue(t.Key) != "spinup:spaceid" {
			cleanTags = append(cleanTags, &Tag{Key: t.Key, Value: t.Value})
			continue
		}

		if aws.StringValue(t.Key) == "spinup:org" || aws.StringValue(t.Key) == "yale:org" {
			if aws.StringValue(t.Value) != org {
				msg := fmt.Sprintf("not a part of our org (%s)", org)
				return nil, errors.New(msg)
			}
		}
	}

	return cleanTags, nil
}
