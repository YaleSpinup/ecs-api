package orchestration

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"testing"
	"time"

	sm "github.com/YaleSpinup/ecs-api/secretsmanager"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/awsutil"
	"github.com/aws/aws-sdk-go/aws/request"
	"github.com/aws/aws-sdk-go/service/ecs"
	"github.com/aws/aws-sdk-go/service/secretsmanager"
)

var (
	goodContainerDefs = []*ecs.ContainerDefinition{
		{
			Name:  aws.String("webserver"),
			Image: aws.String("nginx:alpine"),
		},
		{
			Name:  aws.String("container1"),
			Image: aws.String("secretImage1"),
		},
		{
			Name:  aws.String("container2"),
			Image: aws.String("secretImage2"),
		},
	}

	tdInput = &ecs.RegisterTaskDefinitionInput{
		ContainerDefinitions: goodContainerDefs,
	}

	svcInput = &ecs.CreateServiceInput{
		Cluster: aws.String("getAClu1"),
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
		return nil, awserr.New(secretsmanager.ErrCodeInvalidRequestException, "secret string OR secretbinary is required", nil)
	}

	arn := fmt.Sprintf("arn:%s", aws.StringValue(input.Name))
	return &secretsmanager.CreateSecretOutput{
		ARN:       aws.String(arn),
		Name:      input.Name,
		VersionId: aws.String("v1"),
	}, nil
}

func TestProcessRepositoryCredentialsCreate(t *testing.T) {
	credentialsMapIn := map[string]*secretsmanager.CreateSecretInput{
		"container1": {
			Name:         aws.String("container1"),
			SecretString: aws.String("shhhhhhh"),
		},
		"container2": {
			Name:         aws.String("container2"),
			SecretString: aws.String("donttell"),
		},
	}

	credentialsMapOut := map[string]*secretsmanager.CreateSecretOutput{
		"container1": {
			ARN:       aws.String("arn:spinup/mock/getAClu1/container1"),
			Name:      aws.String("spinup/mock/getAClu1/container1"),
			VersionId: aws.String("v1"),
		},
		"container2": {
			ARN:       aws.String("arn:spinup/mock/getAClu1/container2"),
			Name:      aws.String("spinup/mock/getAClu1/container2"),
			VersionId: aws.String("v1"),
		},
	}

	o := Orchestrator{
		SecretsManager: sm.SecretsManager{Service: &mockSMClient{t: t}},
		Org:            "mock",
	}
	out, _, err := o.processRepositoryCredentialsCreate(context.TODO(), &ServiceOrchestrationInput{})
	if err != nil {
		t.Errorf("expected nil error for processRepositoryCredentials, got %s", err)
	}

	if out != nil {
		t.Errorf("expected nil output for empty repository credentials, got %+v", out)
	}

	out, _, err = o.processRepositoryCredentialsCreate(context.TODO(), &ServiceOrchestrationInput{
		Cluster:        &ecs.CreateClusterInput{ClusterName: aws.String("getAClu1")},
		TaskDefinition: tdInput,
		Credentials:    credentialsMapIn,
		Service:        svcInput,
	})
	if err != nil {
		t.Errorf("expected nil error for processRepositoryCredentials, got %s", err)
	}

	t.Log("got processRepositoryCredentials response", out)

	if !reflect.DeepEqual(credentialsMapOut, out) {
		t.Errorf("Expected %+v\nGot %+v", credentialsMapOut, out)
	}

	for _, c := range tdInput.ContainerDefinitions {
		cName := aws.StringValue(c.Name)
		if _, ok := credentialsMapIn[cName]; ok {
			s, ok := credentialsMapOut[cName]
			if !ok {
				t.Errorf("expected secret for container %s, not found", cName)
				continue
			}

			sArn := aws.StringValue(s.ARN)
			if c.RepositoryCredentials == nil {
				t.Errorf("expected secret arn to be %s on repository credentials for container %s, got nil", sArn, cName)
			} else if cp := aws.StringValue(c.RepositoryCredentials.CredentialsParameter); cp != sArn {
				t.Errorf("expected secret arn to be %s on repository credentials for container %s, got %s", sArn, cName, cp)
			}
		}
	}

}

func TestProcessTaskRepositoryCredentialsCreate(t *testing.T) {
	credentialsMapIn := map[string]*secretsmanager.CreateSecretInput{
		"container1": {
			Name:         aws.String("container1"),
			SecretString: aws.String("shhhhhhh"),
		},
		"container2": {
			Name:         aws.String("container2"),
			SecretString: aws.String("donttell"),
		},
	}

	credentialsMapOut := map[string]*secretsmanager.CreateSecretOutput{
		"container1": {
			ARN:       aws.String("arn:spinup/mock/getAClu1/container1"),
			Name:      aws.String("spinup/mock/getAClu1/container1"),
			VersionId: aws.String("v1"),
		},
		"container2": {
			ARN:       aws.String("arn:spinup/mock/getAClu1/container2"),
			Name:      aws.String("spinup/mock/getAClu1/container2"),
			VersionId: aws.String("v1"),
		},
	}

	o := Orchestrator{
		SecretsManager: sm.SecretsManager{Service: &mockSMClient{t: t}},
		Org:            "mock",
	}
	out, _, err := o.processTaskRepositoryCredentialsCreate(context.TODO(), &TaskDefCreateOrchestrationInput{})
	if err != nil {
		t.Errorf("expected nil error for processTaskRepositoryCredentialsCreate, got %s", err)
	}

	if out != nil {
		t.Errorf("expected nil output for empty repository credentials, got %+v", out)
	}

	out, _, err = o.processTaskRepositoryCredentialsCreate(context.TODO(), &TaskDefCreateOrchestrationInput{
		Cluster:        &ecs.CreateClusterInput{ClusterName: aws.String("getAClu1")},
		TaskDefinition: tdInput,
		Credentials:    credentialsMapIn,
	})
	if err != nil {
		t.Errorf("expected nil error for processTaskRepositoryCredentialsCreate, got %s", err)
	}

	t.Log("got processTaskRepositoryCredentialsCreate response", out)

	if !reflect.DeepEqual(credentialsMapOut, out) {
		t.Errorf("Expected %+v\nGot %+v", credentialsMapOut, out)
	}
}

var testSecrets = []struct {
	ARN          string
	Name         string
	SecretString string
}{
	{
		ARN:          "arn:aws:secretsmanager:us-east-1:12345678910:secret:test-cred-1",
		Name:         "test-cred-1",
		SecretString: "ssshhh1",
	},
	{
		ARN:          "arn:aws:secretsmanager:us-east-1:12345678910:secret:test-cred-2",
		Name:         "test-cred-2",
		SecretString: "ssshhh2",
	},
	{
		ARN:          "arn:aws:secretsmanager:us-east-1:12345678910:secret:test-cred-3",
		Name:         "test-cred-3",
		SecretString: "ssshhh3",
	},
	{
		ARN:          "arn:aws:secretsmanager:us-east-1:12345678910:secret:test-cred-4",
		Name:         "test-cred-4",
		SecretString: "ssshhh4",
	},
}

func (m *mockSMClient) PutSecretValueWithContext(ctx context.Context, input *secretsmanager.PutSecretValueInput, opts ...request.Option) (*secretsmanager.PutSecretValueOutput, error) {
	if m.err != nil {
		return nil, m.err
	}

	if input == nil {
		return nil, awserr.New(secretsmanager.ErrCodeInvalidRequestException, "invalid input", nil)
	}

	if (input.SecretBinary == nil && input.SecretString == nil) || (input.SecretBinary != nil && input.SecretString != nil) {
		return nil, awserr.New(secretsmanager.ErrCodeInvalidRequestException, "secret string OR secretbinary is required", nil)
	}

	for _, secret := range testSecrets {
		if aws.StringValue(input.SecretId) == secret.ARN {
			return &secretsmanager.PutSecretValueOutput{
				ARN:       aws.String(secret.ARN),
				Name:      aws.String(secret.Name),
				VersionId: aws.String("AWSCURRENT"),
			}, nil
		}
	}

	return nil, awserr.New(secretsmanager.ErrCodeResourceNotFoundException, "secret doesn't exist", nil)
}

func (m *mockSMClient) DeleteSecretWithContext(ctx context.Context, input *secretsmanager.DeleteSecretInput, opts ...request.Option) (*secretsmanager.DeleteSecretOutput, error) {
	if m.err != nil {
		return nil, m.err
	}

	if input == nil {
		return nil, awserr.New(secretsmanager.ErrCodeInvalidRequestException, "invalid input", nil)
	}

	if input.SecretId == nil {
		return nil, awserr.New(secretsmanager.ErrCodeInvalidRequestException, "invalid input", nil)
	}

	for _, secret := range testSecrets {
		if aws.StringValue(input.SecretId) == secret.ARN {
			return &secretsmanager.DeleteSecretOutput{
				ARN:          aws.String(secret.ARN),
				Name:         aws.String(secret.Name),
				DeletionDate: aws.Time(time.Now()),
			}, nil
		}
	}

	return nil, awserr.New(secretsmanager.ErrCodeResourceNotFoundException, "secret doesn't exist", nil)
}

func TestProcessRepositoryCredentialsUpdate(t *testing.T) {
	o := Orchestrator{
		SecretsManager: sm.SecretsManager{Service: &mockSMClient{t: t}},
		Org:            "mock",
	}

	// test empty input
	if err := o.processRepositoryCredentialsUpdate(context.TODO(), &ServiceOrchestrationUpdateInput{}, &ServiceOrchestrationUpdateOutput{}); err != nil {
		if e := err.Error(); e != "cannot process nil active input" {
			t.Errorf("Expected error 'cannot process nil active input' for empty input, got '%s'", e)
		}
	} else {
		t.Error("expected error for empty input, not nil")
	}

	type test struct {
		desc     string
		tdinput  ServiceOrchestrationUpdateInput
		active   *ecs.TaskDefinition
		tdresult *ecs.RegisterTaskDefinitionInput
		inputerr error
		err      error
	}
	tests := []test{}

	// If the active container definition HAS repostory credentials set
	// ...AND the input has Credentials defined for the container definition
	// ...AND the input has repository credentials set for the container definition
	// ...THEN update the secret at the (active) ARN in place with the given Credentials and apply to the container definition
	tests = append(tests, test{
		desc: "with active repo creds AND input creds AND input repo creds",
		tdinput: ServiceOrchestrationUpdateInput{
			TaskDefinition: &ecs.RegisterTaskDefinitionInput{
				ContainerDefinitions: []*ecs.ContainerDefinition{
					{
						Name: aws.String("nginx"),
					},
					{
						Name: aws.String("privateapi"),
						RepositoryCredentials: &ecs.RepositoryCredentials{
							CredentialsParameter: aws.String("arn:spinup/mock/someOtherCredentials"),
						},
					},
				},
			},
			Credentials: map[string]*secretsmanager.CreateSecretInput{
				"privateapi": {
					Name:         aws.String("secretCredentials"),
					SecretString: aws.String("ssssshhhh!"),
				},
			},
		},
		active: &ecs.TaskDefinition{
			ContainerDefinitions: []*ecs.ContainerDefinition{
				{
					Name: aws.String("nginx"),
				},
				{
					Name: aws.String("privateapi"),
					RepositoryCredentials: &ecs.RepositoryCredentials{
						CredentialsParameter: aws.String("arn:aws:secretsmanager:us-east-1:12345678910:secret:test-cred-1"),
					},
				},
			},
		},
		tdresult: &ecs.RegisterTaskDefinitionInput{
			ContainerDefinitions: []*ecs.ContainerDefinition{
				{
					Name: aws.String("nginx"),
				},
				{
					Name: aws.String("privateapi"),
					RepositoryCredentials: &ecs.RepositoryCredentials{
						CredentialsParameter: aws.String("arn:aws:secretsmanager:us-east-1:12345678910:secret:test-cred-1"),
					},
				},
			},
		},
	})

	// If the active container definition HAS repository credentials set
	// ...AND the input has Credentials defined for the container definition
	// ...AND the input doesn't have repository credentials set for the container definition
	// ...THEN set the input repository credentials to the ARN for the active container definition repository credentials
	// ...AND update the secret at the ARN in place with the given Credentials
	tests = append(tests, test{
		desc: "with active repo creds AND input creds AND NO input repo creds",
		tdinput: ServiceOrchestrationUpdateInput{
			TaskDefinition: &ecs.RegisterTaskDefinitionInput{
				ContainerDefinitions: []*ecs.ContainerDefinition{
					{
						Name: aws.String("nginx"),
					},
					{
						Name: aws.String("privateapi"),
					},
				},
			},
			Credentials: map[string]*secretsmanager.CreateSecretInput{
				"privateapi": {
					Name:         aws.String("secretCredentials"),
					SecretString: aws.String("ssssshhhh!"),
				},
			},
		},
		active: &ecs.TaskDefinition{
			ContainerDefinitions: []*ecs.ContainerDefinition{
				{
					Name: aws.String("nginx"),
				},
				{
					Name: aws.String("privateapi"),
					RepositoryCredentials: &ecs.RepositoryCredentials{
						CredentialsParameter: aws.String("arn:aws:secretsmanager:us-east-1:12345678910:secret:test-cred-1"),
					},
				},
			},
		},
		tdresult: &ecs.RegisterTaskDefinitionInput{
			ContainerDefinitions: []*ecs.ContainerDefinition{
				{
					Name: aws.String("nginx"),
				},
				{
					Name: aws.String("privateapi"),
					RepositoryCredentials: &ecs.RepositoryCredentials{
						CredentialsParameter: aws.String("arn:aws:secretsmanager:us-east-1:12345678910:secret:test-cred-1"),
					},
				},
			},
		},
	})

	// If the active container definition HAS repository credentials set
	// ...AND the input doesn't have Credentials defined for the container definition
	// ...AND the input has repository credentials set for the container definition
	// ...THEN override the repository credentials with the ARN of the active repository credentials
	tests = append(tests, test{
		desc: "with active repo creds AND NO input creds AND input repo creds",
		tdinput: ServiceOrchestrationUpdateInput{
			TaskDefinition: &ecs.RegisterTaskDefinitionInput{
				ContainerDefinitions: []*ecs.ContainerDefinition{
					{
						Name: aws.String("nginx"),
					},
					{
						Name: aws.String("privateapi"),
						RepositoryCredentials: &ecs.RepositoryCredentials{
							CredentialsParameter: aws.String("arn:aws:secretsmanager:us-east-1:12345678910:secret:test-cred-1"),
						},
					},
				},
			},
		},
		active: &ecs.TaskDefinition{
			ContainerDefinitions: []*ecs.ContainerDefinition{
				{
					Name: aws.String("nginx"),
				},
				{
					Name: aws.String("privateapi"),
					RepositoryCredentials: &ecs.RepositoryCredentials{
						CredentialsParameter: aws.String("arn:aws:secretsmanager:us-east-1:12345678910:secret:test-cred-1"),
					},
				},
			},
		},
		tdresult: &ecs.RegisterTaskDefinitionInput{
			ContainerDefinitions: []*ecs.ContainerDefinition{
				{
					Name: aws.String("nginx"),
				},
				{
					Name: aws.String("privateapi"),
					RepositoryCredentials: &ecs.RepositoryCredentials{
						CredentialsParameter: aws.String("arn:aws:secretsmanager:us-east-1:12345678910:secret:test-cred-1"),
					},
				},
			},
		},
	})

	// If the active container definition HAS repository credentials set
	// ...AND the input doesn't have repository credentials set
	// ...AND the input doesn't have Credentials defined for the container definition
	// ...THEN delete the secret at the ARN for the active container definition
	tests = append(tests, test{
		desc: "with active repo creds AND NO input creds AND NO input repo creds",
		tdinput: ServiceOrchestrationUpdateInput{
			TaskDefinition: &ecs.RegisterTaskDefinitionInput{
				ContainerDefinitions: []*ecs.ContainerDefinition{
					{
						Name: aws.String("nginx"),
					},
					{
						Name: aws.String("privateapi"),
					},
				},
			},
		},
		active: &ecs.TaskDefinition{
			ContainerDefinitions: []*ecs.ContainerDefinition{
				{
					Name: aws.String("nginx"),
				},
				{
					Name: aws.String("privateapi"),
					RepositoryCredentials: &ecs.RepositoryCredentials{
						CredentialsParameter: aws.String("arn:aws:secretsmanager:us-east-1:12345678910:secret:test-cred-1"),
					},
				},
			},
		},
		tdresult: &ecs.RegisterTaskDefinitionInput{
			ContainerDefinitions: []*ecs.ContainerDefinition{
				{
					Name: aws.String("nginx"),
				},
				{
					Name: aws.String("privateapi"),
				},
			},
		},
	})

	// If the active container definition doesn't exist or doesn't have repostitory credentials set
	// ...AND the input has Credentials defined for the container definition
	// ...AND the input has repository credentials defined for the container definition
	// ...THEN update the secret in place or fail if it doesn't exist
	// Note: (this case shouldn't happen)
	tests = append(tests, test{
		desc: "without active container def AND input creds AND input repo creds",
		tdinput: ServiceOrchestrationUpdateInput{
			TaskDefinition: &ecs.RegisterTaskDefinitionInput{
				ContainerDefinitions: []*ecs.ContainerDefinition{
					{
						Name: aws.String("nginx"),
					},
					{
						Name: aws.String("privateapi"),
						RepositoryCredentials: &ecs.RepositoryCredentials{
							CredentialsParameter: aws.String("arn:aws:secretsmanager:us-east-1:12345678910:secret:test-cred-1"),
						},
					},
				},
			},
			Credentials: map[string]*secretsmanager.CreateSecretInput{
				"privateapi": {
					Name:         aws.String("secretCredentials"),
					SecretString: aws.String("ssssshhhh!"),
				},
			},
		},
		active: &ecs.TaskDefinition{
			ContainerDefinitions: []*ecs.ContainerDefinition{
				{
					Name: aws.String("nginx"),
				},
			},
		},
		tdresult: &ecs.RegisterTaskDefinitionInput{
			ContainerDefinitions: []*ecs.ContainerDefinition{
				{
					Name: aws.String("nginx"),
				},
				{
					Name: aws.String("privateapi"),
					RepositoryCredentials: &ecs.RepositoryCredentials{
						CredentialsParameter: aws.String("arn:aws:secretsmanager:us-east-1:12345678910:secret:test-cred-1"),
					},
				},
			},
		},
	})

	tests = append(tests, test{
		desc: "without active repo creds AND input creds AND input repo creds",
		tdinput: ServiceOrchestrationUpdateInput{
			TaskDefinition: &ecs.RegisterTaskDefinitionInput{
				ContainerDefinitions: []*ecs.ContainerDefinition{
					{
						Name: aws.String("nginx"),
					},
					{
						Name: aws.String("privateapi"),
						RepositoryCredentials: &ecs.RepositoryCredentials{
							CredentialsParameter: aws.String("arn:aws:secretsmanager:us-east-1:12345678910:secret:test-cred-1"),
						},
					},
				},
			},
			Credentials: map[string]*secretsmanager.CreateSecretInput{
				"privateapi": {
					Name:         aws.String("secretCredentials"),
					SecretString: aws.String("ssssshhhh!"),
				},
			},
		},
		active: &ecs.TaskDefinition{
			ContainerDefinitions: []*ecs.ContainerDefinition{
				{
					Name: aws.String("nginx"),
				},
				{
					Name: aws.String("privateapi"),
				},
			},
		},
		tdresult: &ecs.RegisterTaskDefinitionInput{
			ContainerDefinitions: []*ecs.ContainerDefinition{
				{
					Name: aws.String("nginx"),
				},
				{
					Name: aws.String("privateapi"),
					RepositoryCredentials: &ecs.RepositoryCredentials{
						CredentialsParameter: aws.String("arn:aws:secretsmanager:us-east-1:12345678910:secret:test-cred-1"),
					},
				},
			},
		},
	})

	// If the active container doesn't exist or doesn't have repository credentials set
	// ...AND the input has Credentials defined for the container definition
	// ...THEN create a new secret and apply the resulting ARN to the repsitory credentials for the input
	tests = append(tests, test{
		desc: "without active container def AND input creds",
		tdinput: ServiceOrchestrationUpdateInput{
			TaskDefinition: &ecs.RegisterTaskDefinitionInput{
				ContainerDefinitions: []*ecs.ContainerDefinition{
					{
						Name: aws.String("nginx"),
					},
					{
						Name: aws.String("privateapi"),
					},
				},
			},
			Credentials: map[string]*secretsmanager.CreateSecretInput{
				"privateapi": {
					Name:         aws.String("secretCredentials"),
					SecretString: aws.String("ssssshhhh!"),
				},
			},
		},
		active: &ecs.TaskDefinition{
			ContainerDefinitions: []*ecs.ContainerDefinition{
				{
					Name: aws.String("nginx"),
				},
			},
		},
		tdresult: &ecs.RegisterTaskDefinitionInput{
			ContainerDefinitions: []*ecs.ContainerDefinition{
				{
					Name: aws.String("nginx"),
				},
				{
					Name: aws.String("privateapi"),
					RepositoryCredentials: &ecs.RepositoryCredentials{
						CredentialsParameter: aws.String("arn:spinup/mock/secretCredentials"),
					},
				},
			},
		},
	})

	tests = append(tests, test{
		desc: "without active repo creds AND input creds",
		tdinput: ServiceOrchestrationUpdateInput{
			TaskDefinition: &ecs.RegisterTaskDefinitionInput{
				ContainerDefinitions: []*ecs.ContainerDefinition{
					{
						Name: aws.String("nginx"),
					},
					{
						Name: aws.String("privateapi"),
					},
				},
			},
			Credentials: map[string]*secretsmanager.CreateSecretInput{
				"privateapi": {
					Name:         aws.String("secretCredentials"),
					SecretString: aws.String("ssssshhhh!"),
				},
			},
		},
		active: &ecs.TaskDefinition{
			ContainerDefinitions: []*ecs.ContainerDefinition{
				{
					Name: aws.String("nginx"),
				},
				{
					Name: aws.String("privateapi"),
				},
			},
		},
		tdresult: &ecs.RegisterTaskDefinitionInput{
			ContainerDefinitions: []*ecs.ContainerDefinition{
				{
					Name: aws.String("nginx"),
				},
				{
					Name: aws.String("privateapi"),
					RepositoryCredentials: &ecs.RepositoryCredentials{
						CredentialsParameter: aws.String("arn:spinup/mock/secretCredentials"),
					},
				},
			},
		},
	})

	// If the active container doesn't exist or doesn't have repository credentials set
	// ...AND the input doesn't have Credentials defined for the container definition
	// ...THEN assume public image, no secrets are created, no repository credentials are applied
	tests = append(tests, test{
		desc: "without active container def AND NO input creds",
		tdinput: ServiceOrchestrationUpdateInput{
			TaskDefinition: &ecs.RegisterTaskDefinitionInput{
				ContainerDefinitions: []*ecs.ContainerDefinition{
					{
						Name: aws.String("nginx"),
					},
					{
						Name: aws.String("notsoprivateapi"),
					},
				},
			},
		},
		active: &ecs.TaskDefinition{
			ContainerDefinitions: []*ecs.ContainerDefinition{
				{
					Name: aws.String("nginx"),
				},
			},
		},
		tdresult: &ecs.RegisterTaskDefinitionInput{
			ContainerDefinitions: []*ecs.ContainerDefinition{
				{
					Name: aws.String("nginx"),
				},
				{
					Name: aws.String("notsoprivateapi"),
				},
			},
		},
	})

	tests = append(tests, test{
		desc: "without active repo creds AND NO input creds",
		tdinput: ServiceOrchestrationUpdateInput{
			TaskDefinition: &ecs.RegisterTaskDefinitionInput{
				ContainerDefinitions: []*ecs.ContainerDefinition{
					{
						Name: aws.String("nginx"),
					},
					{
						Name: aws.String("notsoprivateapi"),
					},
				},
			},
		},
		active: &ecs.TaskDefinition{
			ContainerDefinitions: []*ecs.ContainerDefinition{
				{
					Name: aws.String("nginx"),
				},
				{
					Name: aws.String("notsoprivateapi"),
				},
			},
		},
		tdresult: &ecs.RegisterTaskDefinitionInput{
			ContainerDefinitions: []*ecs.ContainerDefinition{
				{
					Name: aws.String("nginx"),
				},
				{
					Name: aws.String("notsoprivateapi"),
				},
			},
		},
	})

	// error creating secret
	tests = append(tests, test{
		inputerr: errors.New("boom"),
		err:      errors.New("InternalError: failed to create secret (boom)"),
		desc:     "error creating secret without active container def AND input creds",
		tdinput: ServiceOrchestrationUpdateInput{
			TaskDefinition: &ecs.RegisterTaskDefinitionInput{
				ContainerDefinitions: []*ecs.ContainerDefinition{
					{
						Name: aws.String("nginx"),
					},
					{
						Name: aws.String("privateapi"),
					},
				},
			},
			Credentials: map[string]*secretsmanager.CreateSecretInput{
				"privateapi": {
					Name:         aws.String("secretCredentials"),
					SecretString: aws.String("ssssshhhh!"),
				},
			},
		},
		active: &ecs.TaskDefinition{
			ContainerDefinitions: []*ecs.ContainerDefinition{
				{
					Name: aws.String("nginx"),
				},
			},
		},
	})

	// error updating secret
	tests = append(tests, test{
		inputerr: errors.New("boom"),
		err:      errors.New("InternalError: failed to update secret (boom)"),
		desc:     "error updating secret with active repo creds AND input creds AND NO input repo creds",
		tdinput: ServiceOrchestrationUpdateInput{
			TaskDefinition: &ecs.RegisterTaskDefinitionInput{
				ContainerDefinitions: []*ecs.ContainerDefinition{
					{
						Name: aws.String("nginx"),
					},
					{
						Name: aws.String("privateapi"),
					},
				},
			},
			Credentials: map[string]*secretsmanager.CreateSecretInput{
				"privateapi": {
					Name:         aws.String("secretCredentials"),
					SecretString: aws.String("ssssshhhh!"),
				},
			},
		},
		active: &ecs.TaskDefinition{
			ContainerDefinitions: []*ecs.ContainerDefinition{
				{
					Name: aws.String("nginx"),
				},
				{
					Name: aws.String("privateapi"),
					RepositoryCredentials: &ecs.RepositoryCredentials{
						CredentialsParameter: aws.String("arn:aws:secretsmanager:us-east-1:12345678910:secret:test-cred-1"),
					},
				},
			},
		},
	})

	// error deleting secret
	tests = append(tests, test{
		inputerr: errors.New("boom"),
		err:      errors.New("InternalError: failed to delete secret with id arn:aws:secretsmanager:us-east-1:12345678910:secret:test-cred-1 (boom)"),
		desc:     "error deleting secret with active repo creds AND NO input creds AND NO input repo creds",
		tdinput: ServiceOrchestrationUpdateInput{
			TaskDefinition: &ecs.RegisterTaskDefinitionInput{
				ContainerDefinitions: []*ecs.ContainerDefinition{
					{
						Name: aws.String("nginx"),
					},
					{
						Name: aws.String("privateapi"),
					},
				},
			},
		},
		active: &ecs.TaskDefinition{
			ContainerDefinitions: []*ecs.ContainerDefinition{
				{
					Name: aws.String("nginx"),
				},
				{
					Name: aws.String("privateapi"),
					RepositoryCredentials: &ecs.RepositoryCredentials{
						CredentialsParameter: aws.String("arn:aws:secretsmanager:us-east-1:12345678910:secret:test-cred-1"),
					},
				},
			},
		},
	})

	for _, test := range tests {
		o = Orchestrator{
			SecretsManager: sm.SecretsManager{Service: &mockSMClient{t: t, err: test.inputerr}},
			Org:            "mock",
		}

		t.Logf("testing %s", test.desc)

		err := o.processRepositoryCredentialsUpdate(context.TODO(), &test.tdinput, &ServiceOrchestrationUpdateOutput{
			TaskDefinition: test.active,
		})

		if test.err == nil && err != nil {
			t.Errorf("expected nil error, got %s", err)
		} else if test.err != nil && err == nil {
			t.Errorf("expected error '%s', got nil", test.err)
		} else if test.err != nil && err != nil {
			if test.err.Error() != err.Error() {
				t.Errorf("expected error '%s', got '%s''", test.err, err)
			}
		} else {
			if !awsutil.DeepEqual(test.tdinput.TaskDefinition.ContainerDefinitions, test.tdresult.ContainerDefinitions) {
				t.Errorf("expected container defs %s, got %s", awsutil.Prettify(test.tdresult.ContainerDefinitions), awsutil.Prettify(test.tdinput.TaskDefinition.ContainerDefinitions))
			}
		}
	}
}

func TestContainerDefinitionCredsMap(t *testing.T) {
	t.Log("TODO")
}

func TestOrchestrator_createRepostitoryCredentials(t *testing.T) {
	type fields struct {
		SecretsManager sm.SecretsManager
		Org            string
	}
	type args struct {
		ctx    context.Context
		prefix string
		input  map[string]*secretsmanager.CreateSecretInput
		tags   []*Tag
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    map[string]*secretsmanager.CreateSecretOutput
		wantErr bool
	}{
		{
			name: "nil input",
			fields: fields{
				SecretsManager: sm.SecretsManager{Service: &mockSMClient{t: t}},
				Org:            "mock",
			},
			args: args{
				ctx:    context.TODO(),
				prefix: "/foo/bar",
			},
			want: map[string]*secretsmanager.CreateSecretOutput{},
		},
		{
			name: "empty input",
			fields: fields{
				SecretsManager: sm.SecretsManager{Service: &mockSMClient{t: t}},
				Org:            "mock",
			},
			args: args{
				ctx:    context.TODO(),
				prefix: "/foo/bar",
				input:  map[string]*secretsmanager.CreateSecretInput{},
			},
			want: map[string]*secretsmanager.CreateSecretOutput{},
		},
		{
			name: "create secret aws error",
			fields: fields{
				SecretsManager: sm.SecretsManager{
					Service: newMockSMClient(t, awserr.New(secretsmanager.ErrCodeInvalidRequestException, "invalid input", nil)),
				},
				Org: "mock",
			},
			args: args{
				ctx:    context.TODO(),
				prefix: "/foo/bar",
				input: map[string]*secretsmanager.CreateSecretInput{
					"container1": {
						Description:  aws.String("secret for container1"),
						Name:         aws.String("container1-secret"),
						SecretString: aws.String("shhhhh"),
					},
				},
			},
			wantErr: true,
		},
		{
			name: "create secret random error",
			fields: fields{
				SecretsManager: sm.SecretsManager{
					Service: newMockSMClient(t, errors.New("boom!")),
				},
				Org: "mock",
			},
			args: args{
				ctx:    context.TODO(),
				prefix: "/foo/bar",
				input: map[string]*secretsmanager.CreateSecretInput{
					"container1": {
						Description:  aws.String("secret for container1"),
						Name:         aws.String("container1-secret"),
						SecretString: aws.String("shhhhh"),
					},
				},
			},
			wantErr: true,
		},
		{
			name: "good input 1 secret",
			fields: fields{
				SecretsManager: sm.SecretsManager{Service: &mockSMClient{t: t}},
				Org:            "mock",
			},
			args: args{
				ctx:    context.TODO(),
				prefix: "/foo/bar",
				input: map[string]*secretsmanager.CreateSecretInput{
					"container1": {
						Description:  aws.String("secret for container1"),
						Name:         aws.String("container1-secret"),
						SecretString: aws.String("shhhhh"),
					},
				},
			},
			want: map[string]*secretsmanager.CreateSecretOutput{
				"container1": {
					ARN:       aws.String("arn:/foo/bar/container1-secret"),
					Name:      aws.String("/foo/bar/container1-secret"),
					VersionId: aws.String("v1"),
				},
			},
		},
		{
			name: "good input 3 secret",
			fields: fields{
				SecretsManager: sm.SecretsManager{Service: &mockSMClient{t: t}},
				Org:            "mock",
			},
			args: args{
				ctx:    context.TODO(),
				prefix: "/foo/bar",
				input: map[string]*secretsmanager.CreateSecretInput{
					"container1": {
						Description:  aws.String("secret for container1"),
						Name:         aws.String("container1-secret"),
						SecretString: aws.String("shhhhh"),
					},
					"container2": {
						Description:  aws.String("secret for container2"),
						Name:         aws.String("container2-secret"),
						SecretString: aws.String("shhhhh"),
					},
					"container3": {
						Description:  aws.String("secret for container3"),
						Name:         aws.String("container3-secret"),
						SecretString: aws.String("shhhhh"),
					},
				},
			},
			want: map[string]*secretsmanager.CreateSecretOutput{
				"container1": {
					ARN:       aws.String("arn:/foo/bar/container1-secret"),
					Name:      aws.String("/foo/bar/container1-secret"),
					VersionId: aws.String("v1"),
				},
				"container2": {
					ARN:       aws.String("arn:/foo/bar/container2-secret"),
					Name:      aws.String("/foo/bar/container2-secret"),
					VersionId: aws.String("v1"),
				},
				"container3": {
					ARN:       aws.String("arn:/foo/bar/container3-secret"),
					Name:      aws.String("/foo/bar/container3-secret"),
					VersionId: aws.String("v1"),
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			o := &Orchestrator{
				SecretsManager: tt.fields.SecretsManager,
				Org:            tt.fields.Org,
			}

			got, err := o.createRepostitoryCredentials(tt.args.ctx, tt.args.prefix, tt.args.input, tt.args.tags)
			if (err != nil) != tt.wantErr {
				t.Errorf("Orchestrator.createRepostitoryCredentials() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Orchestrator.createRepostitoryCredentials() = %v, want %v", got, tt.want)
			}
		})
	}
}
