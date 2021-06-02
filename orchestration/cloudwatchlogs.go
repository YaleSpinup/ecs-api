package orchestration

import (
	"context"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ecs"
	log "github.com/sirupsen/logrus"
)

// cloudwatchLogGroups collects all of the log group arns for the passed container definitions
func (o *Orchestrator) cloudwatchLogGroups(ctx context.Context, containerDefs []*ecs.ContainerDefinition) ([]string, error) {
	logGroupArns := []string{}
	logGroupNames := map[string]struct{}{}

	for _, cd := range containerDefs {
		if cd.LogConfiguration != nil {
			logGroupName, ok := cd.LogConfiguration.Options["awslogs-group"]
			if !ok {
				continue
			}

			lgn := aws.StringValue(logGroupName)

			// if the log group has already been fetched, skip it
			if _, ok := logGroupNames[lgn]; ok {
				continue
			}
			logGroupNames[lgn] = struct{}{}

			lg, err := o.CloudWatchLogs.GetLogGroup(ctx, lgn)
			if err != nil {
				log.Errorf("failed to get details about log group")
				continue
			}

			// log group ARNs are returned with a :* on the end, remove it if it exists
			cleanArn := strings.TrimSuffix(aws.StringValue(lg.Arn), ":*")
			logGroupArns = append(logGroupArns, cleanArn)
		}
	}

	return logGroupArns, nil
}
