// Package orchestration brings together the other components of the API into a
// single orchestration interface for creating and deleting ecs services
package orchestration

import (
	"github.com/YaleSpinup/ecs-api/ecs"
	"github.com/YaleSpinup/ecs-api/iam"
	"github.com/YaleSpinup/ecs-api/secretsmanager"
	"github.com/YaleSpinup/ecs-api/servicediscovery"
	"github.com/aws/aws-sdk-go/aws"
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
	// DefaultPublic disables the setting of public IPs on ENIs by default
	DefaultPublic = aws.String("DISABLED")
	// DefaultSubnets sets a list of default subnets to attach ENIs
	DefaultSubnets = []*string{}
	// DefaultSecurityGroups sets a list of default sgs to attach to ENIs
	DefaultSecurityGroups = []*string{}
)

// Orchestrator holds the service discovery client, iam client, ecs client, secretsmanager client, input, and output
type Orchestrator struct {
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
	// Org is the organization where this orchestration runs
	Org string
}
