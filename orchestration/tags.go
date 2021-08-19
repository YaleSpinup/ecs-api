package orchestration

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/arn"
	"github.com/aws/aws-sdk-go/aws/awsutil"
	"github.com/aws/aws-sdk-go/service/ecs"
	"github.com/aws/aws-sdk-go/service/iam"
	"github.com/aws/aws-sdk-go/service/secretsmanager"
	log "github.com/sirupsen/logrus"
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
func cleanTags(org, spaceid, stype, flavor string, tags []*Tag) ([]*Tag, error) {
	cleanTags := []*Tag{
		{
			Key:   aws.String("spinup:org"),
			Value: aws.String(org),
		},
		{
			Key:   aws.String("spinup:spaceid"),
			Value: aws.String(spaceid),
		},
		{
			Key:   aws.String("spinup:type"),
			Value: aws.String(stype),
		},
		{
			Key:   aws.String("spinup:flavor"),
			Value: aws.String(flavor),
		},
	}

	for _, t := range tags {
		switch aws.StringValue(t.Key) {
		case "spinup:org", "yale:org":
			if aws.StringValue(t.Value) != org {
				msg := fmt.Sprintf("not a part of our org (%s)", org)
				return nil, errors.New(msg)
			}
		case "spinup:spaceid", "spinup:type", "spinup:flavor":
			log.Debugf("skipping api controlled tag %s", aws.StringValue(t.Key))
		default:
			cleanTags = append(cleanTags, &Tag{Key: t.Key, Value: t.Value})
		}
	}

	return cleanTags, nil
}

// sharedResourceTags generates a taglist for resources that are shared (clusters, roles, etc)
func sharedResourceTags(name string, tags []*Tag) map[string]*string {
	output := map[string]*string{}

	if name != "" {
		output["Name"] = aws.String(name)
	}

	for _, t := range tags {
		key := aws.StringValue(t.Key)
		if strings.ToLower(key) != "spinup:category" && strings.ToLower(key) != "name" {
			output[key] = t.Value
		}
	}

	return output
}

// specificResourceTags generates a taglist for resources that are specific to a resource in spinup
func specificResourceTags(tags []*Tag) map[string]*string {
	output := map[string]*string{}

	for _, t := range tags {
		output[aws.StringValue(t.Key)] = t.Value
	}

	return output
}

// roleTags generates a taglist for IAM roles which are still unique and special snowflakes (and dont't
// support the resourcegroupstaggingapi)
func roleTags(name string, tags []*Tag) []*iam.Tag {
	output := []*iam.Tag{
		{
			Key:   aws.String("Name"),
			Value: aws.String(name),
		},
	}

	for _, t := range tags {
		key := aws.StringValue(t.Key)
		if strings.ToLower(key) != "spinup:category" && strings.ToLower(key) != "name" {
			output = append(output, &iam.Tag{
				Key:   t.Key,
				Value: t.Value,
			})
		}
	}

	return output
}

func (o *Orchestrator) processServiceTagsUpdate(ctx context.Context, active *ServiceOrchestrationUpdateOutput, tags []*Tag) error {
	log.Debugf("processing tags update with tags list %s", awsutil.Prettify(tags))

	// resources with the spaceid as their name tag (clusters, cloudwatchlogs loggroups, etc)
	spaceIdNameTags := sharedResourceTags(aws.StringValue(active.Cluster.ClusterName), tags)
	spaceIdNameArns := []*string{active.Cluster.ClusterArn}
	spaceIdNameArns = append(spaceIdNameArns, aws.StringSlice(active.CloudwatchLogGroups)...)
	if err := o.ResourceGroupsTaggingAPI.TagResource(ctx, spaceIdNameArns, spaceIdNameTags); err != nil {
		return err
	}

	// get the ecs task execution role arn from the active task definition
	ecsTaskExecutionRoleArn, err := arn.Parse(aws.StringValue(active.TaskDefinition.ExecutionRoleArn))
	if err != nil {
		return err
	}

	// determine the ecs task execution role name from the arn
	ecsTaskExecutionRoleName := ecsTaskExecutionRoleArn.Resource[strings.LastIndex(ecsTaskExecutionRoleArn.Resource, "/")+1:]

	// TODO roles don't currently support the resourcegroupstaggingapi
	// roleTags := sharedResourceTags(ecsTaskExecutionRoleName, tags)
	// if err := o.ResourceGroupsTaggingAPI.TagResource(ctx, []*string{
	// 	active.TaskDefinition.ExecutionRoleArn,
	// }, roleTags); err != nil {
	// 	return err
	// }

	roleTags := roleTags(ecsTaskExecutionRoleName, tags)
	if err := o.IAM.TagRole(ctx, ecsTaskExecutionRoleName, roleTags); err != nil {
		return err
	}

	commonTags := specificResourceTags(tags)

	commonArns := []*string{
		active.Service.ServiceArn,
		active.TaskDefinition.TaskDefinitionArn,
	}

	// collect secretsmanager ARNs
	for _, containerDef := range active.TaskDefinition.ContainerDefinitions {
		repositoryCredentials := containerDef.RepositoryCredentials
		if repositoryCredentials != nil && repositoryCredentials.CredentialsParameter != nil {
			commonArns = append(commonArns, repositoryCredentials.CredentialsParameter)
		}
	}

	if err := o.ResourceGroupsTaggingAPI.TagResource(ctx, commonArns, commonTags); err != nil {
		return err
	}

	// set the active tags for output
	active.Tags = tags

	return err
}

func (o *Orchestrator) processTaskDefTagsUpdate(ctx context.Context, active *TaskDefUpdateOrchestrationOutput, tags []*Tag) error {
	log.Debugf("processing tags update with tags list %s", awsutil.Prettify(tags))

	// resources with the spaceid as their name tag (clusters, cloudwatchlogs loggroups, etc)
	spaceIdNameTags := sharedResourceTags(aws.StringValue(active.Cluster.ClusterName), tags)
	spaceIdNameArns := []*string{active.Cluster.ClusterArn}
	spaceIdNameArns = append(spaceIdNameArns, aws.StringSlice(active.CloudwatchLogGroups)...)
	if err := o.ResourceGroupsTaggingAPI.TagResource(ctx, spaceIdNameArns, spaceIdNameTags); err != nil {
		return err
	}

	// get the ecs task execution role arn from the active task definition
	ecsTaskExecutionRoleArn, err := arn.Parse(aws.StringValue(active.TaskDefinition.ExecutionRoleArn))
	if err != nil {
		return err
	}

	// determine the ecs task execution role name from the arn
	ecsTaskExecutionRoleName := ecsTaskExecutionRoleArn.Resource[strings.LastIndex(ecsTaskExecutionRoleArn.Resource, "/")+1:]

	// TODO roles don't currently support the resourcegroupstaggingapi
	// roleTags := sharedResourceTags(ecsTaskExecutionRoleName, tags)
	// if err := o.ResourceGroupsTaggingAPI.TagResource(ctx, []*string{
	// 	active.TaskDefinition.ExecutionRoleArn,
	// }, roleTags); err != nil {
	// 	return err
	// }

	roleTags := roleTags(ecsTaskExecutionRoleName, tags)
	if err := o.IAM.TagRole(ctx, ecsTaskExecutionRoleName, roleTags); err != nil {
		return err
	}

	commonTags := specificResourceTags(tags)

	commonArns := []*string{
		active.TaskDefinition.TaskDefinitionArn,
	}

	// collect secretsmanager ARNs
	for _, containerDef := range active.TaskDefinition.ContainerDefinitions {
		repositoryCredentials := containerDef.RepositoryCredentials
		if repositoryCredentials != nil && repositoryCredentials.CredentialsParameter != nil {
			commonArns = append(commonArns, repositoryCredentials.CredentialsParameter)
		}
	}

	if err := o.ResourceGroupsTaggingAPI.TagResource(ctx, commonArns, commonTags); err != nil {
		return err
	}

	// set the active tags for output
	active.Tags = tags

	return err
}
