package orchestration

import (
	"context"
	"fmt"
	"reflect"
	"testing"

	sm "github.com/YaleSpinup/ecs-api/secretsmanager"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/request"
	"github.com/aws/aws-sdk-go/service/ecs"
	"github.com/aws/aws-sdk-go/service/secretsmanager"
)

var (
	goodContainerDefs = []*ecs.ContainerDefinition{
		&ecs.ContainerDefinition{
			Name:  aws.String("webserver"),
			Image: aws.String("nginx:alpine"),
		},
		&ecs.ContainerDefinition{
			Name:  aws.String("testDef1"),
			Image: aws.String("secretImage1"),
		},
		&ecs.ContainerDefinition{
			Name:  aws.String("testDef2"),
			Image: aws.String("secretImage2"),
		},
	}

	tdInput = &ecs.RegisterTaskDefinitionInput{
		ContainerDefinitions: goodContainerDefs,
	}
)

func (m *mockSMClient) CreateSecretWithContext(ctx context.Context, input *secretsmanager.CreateSecretInput, opts ...request.Option) (*secretsmanager.CreateSecretOutput, error) {
	if m.err != nil {
		return nil, m.err
	}

	if input == nil {
		return nil, awserr.New(secretsmanager.ErrCodeInvalidRequestException, "invalid input", nil)
	}

	if aws.StringValue(input.Name) == "" {
		return nil, awserr.New(secretsmanager.ErrCodeInvalidRequestException, "Name is required", nil)
	}

	if (input.SecretBinary == nil && input.SecretString == nil) || (input.SecretBinary != nil && input.SecretString != nil) {
		return nil, awserr.
			New(secretsmanager.ErrCodeInvalidRequestException, "secret string OR secretbinary is required", nil)
	}

	arn := fmt.Sprintf("arn:%s", aws.StringValue(input.Name))
	return &secretsmanager.CreateSecretOutput{
		ARN:       aws.String(arn),
		Name:      input.Name,
		VersionId: aws.String("v1"),
	}, nil
}

func TestProcessRepositoryCredentials(t *testing.T) {
	o := Orchestrator{
		SecretsManager: sm.SecretsManager{Service: &mockSMClient{t: t}},
	}
	out, err := o.processRepositoryCredentials(context.TODO(), &ServiceOrchestrationInput{})
	if err != nil {
		t.Errorf("expected nil error for processRepositoryCredentials, got %s", err)
	}

	if out != nil {
		t.Errorf("expected nil output for empty repository credentials, got %+v", out)
	}

	out, err = o.processRepositoryCredentials(context.TODO(), &ServiceOrchestrationInput{
		TaskDefinition: tdInput,
		Credentials:    credentialsMapIn,
	})
	if err != nil {
		t.Errorf("expected nil error for processRepositoryCredentials, got %s", err)
	}

	t.Log("got processRepositoryCredentials response", out)
	if !reflect.DeepEqual(credentialsMapOut, out) {
		t.Fatalf("Expected %+v\nGot %+v", credentialsMapOut, out)
	}
}
