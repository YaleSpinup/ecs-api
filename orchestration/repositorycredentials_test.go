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
		return nil, awserr.New(secretsmanager.ErrCodeInvalidRequestException, "secret string OR secretbinary is required", nil)
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

func TestProcessRepositoryCredentialsUpdate(t *testing.T) {
	o := Orchestrator{
		SecretsManager: sm.SecretsManager{Service: &mockSMClient{t: t}},
	}

	baseTdefInput := ecs.RegisterTaskDefinitionInput{
		Family: aws.String("tdef1"),
		Cpu:    aws.String("256"),
		Memory: aws.String("512"),
		ContainerDefinitions: []*ecs.ContainerDefinition{
			&ecs.ContainerDefinition{
				Name:  aws.String("nginx"),
				Image: aws.String("nginx:alpine"),
			},
			&ecs.ContainerDefinition{
				Name:  aws.String("privateapi"),
				Image: aws.String("privateapi:latest"),
			},
		},
	}

	var tests = []struct {
		desc           string
		tdinput        ecs.RegisterTaskDefinitionInput
		credentialsMap map[string]*secretsmanager.CreateSecretInput
		tdresult       ecs.RegisterTaskDefinitionInput
	}{
		{
			desc:           "empty everything",
			tdinput:        ecs.RegisterTaskDefinitionInput{},
			credentialsMap: map[string]*secretsmanager.CreateSecretInput{},
			tdresult:       ecs.RegisterTaskDefinitionInput{},
		},
		{
			desc:           "no creds map",
			tdinput:        baseTdefInput,
			credentialsMap: map[string]*secretsmanager.CreateSecretInput{},
			tdresult:       baseTdefInput,
		},
		{
			desc:    "new creds from map",
			tdinput: baseTdefInput,
			credentialsMap: map[string]*secretsmanager.CreateSecretInput{
				"privateapi": &secretsmanager.CreateSecretInput{
					Name:         aws.String("secret credentials"),
					SecretString: aws.String("ssssshhhh!"),
				},
			},
			tdresult: ecs.RegisterTaskDefinitionInput{
				Family: aws.String("tdef1"),
				Cpu:    aws.String("256"),
				Memory: aws.String("512"),
				ContainerDefinitions: []*ecs.ContainerDefinition{
					&ecs.ContainerDefinition{
						Name:  aws.String("nginx"),
						Image: aws.String("nginx:alpine"),
					},
					&ecs.ContainerDefinition{
						Name:  aws.String("privateapi"),
						Image: aws.String("privateapi:latest"),
						RepositoryCredentials: &ecs.RepositoryCredentials{
							CredentialsParameter: aws.String("arn:secret credentials"),
						},
					},
				},
			},
		},
		{
			desc: "update credentials",
			tdinput: ecs.RegisterTaskDefinitionInput{
				Family: aws.String("tdef1"),
				Cpu:    aws.String("256"),
				Memory: aws.String("512"),
				ContainerDefinitions: []*ecs.ContainerDefinition{
					&ecs.ContainerDefinition{
						Name:  aws.String("nginx"),
						Image: aws.String("nginx:alpine"),
					},
					&ecs.ContainerDefinition{
						Name:  aws.String("privateapi"),
						Image: aws.String("privateapi:latest"),
						RepositoryCredentials: &ecs.RepositoryCredentials{
							CredentialsParameter: aws.String("arn:aws:secretsmanager:us-east-1:12345678910:secret:test-cred-1"),
						},
					},
				},
			},
			credentialsMap: map[string]*secretsmanager.CreateSecretInput{
				"privateapi": &secretsmanager.CreateSecretInput{
					Name:         aws.String("secret credentials"),
					SecretString: aws.String("ssssshhhh!"),
				},
			},
			tdresult: ecs.RegisterTaskDefinitionInput{
				Family: aws.String("tdef1"),
				Cpu:    aws.String("256"),
				Memory: aws.String("512"),
				ContainerDefinitions: []*ecs.ContainerDefinition{
					&ecs.ContainerDefinition{
						Name:  aws.String("nginx"),
						Image: aws.String("nginx:alpine"),
					},
					&ecs.ContainerDefinition{
						Name:  aws.String("privateapi"),
						Image: aws.String("privateapi:latest"),
						RepositoryCredentials: &ecs.RepositoryCredentials{
							CredentialsParameter: aws.String("arn:aws:secretsmanager:us-east-1:12345678910:secret:test-cred-1"),
						},
					},
				},
			},
		},
	}

	out, err := o.processRepositoryCredentialsUpdate(context.TODO(), &ServiceOrchestrationUpdateInput{})
	if err != nil {
		t.Errorf("expected nil error for processRepositoryCredentialsUpdate, got %s", err)
	}

	if out != nil {
		t.Errorf("expected nil output for empty repository credentials, got %+v", out)
	}

	for _, test := range tests {
		t.Logf("testing %s", test.desc)

		tdef := test.tdinput
		out, err = o.processRepositoryCredentialsUpdate(context.TODO(), &ServiceOrchestrationUpdateInput{
			TaskDefinition: &tdef,
			Credentials:    test.credentialsMap,
		})
		if err != nil {
			t.Errorf("expected nil error for processRepositoryCredentialsUpdate, got %s", err)
		}

		t.Log("got processRepositoryCredentials response", out)
		if !reflect.DeepEqual(tdef, test.tdresult) {
			t.Fatalf("Expected %+v\nGot %+v", test.tdresult, tdef)
		}
	}
}

func TestProcessSecretsmanagerTags(t *testing.T) {
	o := Orchestrator{
		SecretsManager: sm.SecretsManager{Service: &mockSMClient{t: t}},
		Org:            "testOrg",
	}

	var tests = []struct {
		input  []*secretsmanager.Tag
		output []*secretsmanager.Tag
	}{
		{
			input: []*secretsmanager.Tag{
				&secretsmanager.Tag{
					Key:   aws.String("foo"),
					Value: aws.String("bar"),
				},
			},
			output: []*secretsmanager.Tag{
				&secretsmanager.Tag{
					Key:   aws.String("foo"),
					Value: aws.String("bar"),
				},
				&secretsmanager.Tag{
					Key:   aws.String("spinup:org"),
					Value: aws.String("testOrg"),
				},
			},
		},
		{
			input: []*secretsmanager.Tag{
				&secretsmanager.Tag{
					Key:   aws.String("foo"),
					Value: aws.String("bar"),
				},
				&secretsmanager.Tag{
					Key:   aws.String("spinup:org"),
					Value: aws.String("someOtherOrg"),
				},
				&secretsmanager.Tag{
					Key:   aws.String("yale:org"),
					Value: aws.String("someOtherOrg"),
				},
			},
			output: []*secretsmanager.Tag{
				&secretsmanager.Tag{
					Key:   aws.String("foo"),
					Value: aws.String("bar"),
				},
				&secretsmanager.Tag{
					Key:   aws.String("spinup:org"),
					Value: aws.String("testOrg"),
				},
			},
		},
	}

	for _, test := range tests {
		out := o.processSecretsmanagerTags(test.input)

		for _, tag := range test.output {
			exists := false
			for _, otag := range out {
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
