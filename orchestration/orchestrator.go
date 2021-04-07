// Package orchestration brings together the other components of the API into a
// single orchestration interface for creating and deleting ecs services
package orchestration

import (
	"context"
	"time"

	"github.com/YaleSpinup/ecs-api/cloudwatchlogs"
	"github.com/YaleSpinup/ecs-api/ecs"
	"github.com/YaleSpinup/ecs-api/iam"
	"github.com/YaleSpinup/ecs-api/secretsmanager"
	"github.com/YaleSpinup/ecs-api/servicediscovery"
	"github.com/aws/aws-sdk-go/aws"

	log "github.com/sirupsen/logrus"
)

var (
	// DefaultCompatabilities sets the default task definition compatabilities to
	// Fargate.  By default, we won't support standard ECS.
	// https://docs.aws.amazon.com/AmazonECS/latest/APIReference/API_TaskDefinition.html
	DefaultCompatabilities = []*string{
		aws.String("FARGATE"),
	}
	// DefaultNetworkMode sets the default networking more for task definitions created
	// by the api.  Currently, Fargate only supports vpc networking.
	// https://docs.aws.amazon.com/AmazonECS/latest/developerguide/task-networking.html
	DefaultNetworkMode = aws.String("awsvpc")
	// DefaultLaunchType sets the default launch type to Fargate
	DefaultLaunchType = aws.String("FARGATE")
	// DefaultCloudwatchLogsRetention sets the detfault retention (in days) for logs in cloudwatch
	DefaultCloudwatchLogsRetention = aws.Int64(int64(365))
)

// Orchestrator holds the service discovery client, iam client, ecs client, secretsmanager client, input, and output
type Orchestrator struct {
	CloudWatchLogs cloudwatchlogs.CloudWatchLogs
	// https://docs.aws.amazon.com/sdk-for-go/api/service/ecs/#ECS
	ECS ecs.ECS
	// https://docs.aws.amazon.com/sdk-for-go/api/service/iam/#IAM
	IAM iam.IAM
	// https://docs.aws.amazon.com/sdk-for-go/api/service/secretsmanager/#SecretsManager
	SecretsManager secretsmanager.SecretsManager
	// https://docs.aws.amazon.com/sdk-for-go/api/service/servicediscovery/#ServiceDiscovery
	ServiceDiscovery servicediscovery.ServiceDiscovery
	// Token is a uniqueness token for calls to AWS
	Token string
	// DefaultPublic disables the setting of public IPs on ENIs by default
	DefaultPublic string
	// DefaultSubnets sets a list of default subnets to attach ENIs
	DefaultSubnets []string
	// DefaultSecurityGroups sets a list of default sgs to attach to ENIs
	DefaultSecurityGroups []string
	// Org is the organization where this orchestration runs
	Org string
}

type rollbackFunc func(ctx context.Context) error

// rollBack executes functions from a stack of rollback functions
func rollBack(t *[]rollbackFunc) {
	if t == nil {
		return
	}

	timeout, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	done := make(chan string, 1)
	go func() {
		tasks := *t
		log.Errorf("executing rollback of %d tasks", len(tasks))
		for i := len(tasks) - 1; i >= 0; i-- {
			f := tasks[i]
			if funcerr := f(timeout); funcerr != nil {
				log.Errorf("rollback task error: %s, continuing rollback", funcerr)
			}
			log.Infof("executed rollback task %d of %d", len(tasks)-i, len(tasks))
		}
		done <- "success"
	}()

	// wait for a done context
	select {
	case <-timeout.Done():
		log.Error("timeout waiting for successful rollback")
	case <-done:
		log.Info("successfully rolled back")
	}
}

func defaultRbfunc(name string) rollbackFunc {
	return func(_ context.Context) error {
		log.Infof("%s rollback, nothing to do", name)
		return nil
	}
}
