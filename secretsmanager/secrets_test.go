package secretsmanager

import (
	"context"
	"reflect"
	"strconv"
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/request"
	"github.com/aws/aws-sdk-go/service/secretsmanager"

	"github.com/YaleSpinup/s3-api/apierror"
)

var secretList1 = []*secretsmanager.SecretListEntry{
	&secretsmanager.SecretListEntry{
		ARN:  aws.String("arn:aws:secretsmanager:us-east-1:00000000000:secret:Secret01-abcdefg"),
		Name: aws.String("Secret01"),
	},
	&secretsmanager.SecretListEntry{
		ARN:  aws.String("arn:aws:secretsmanager:us-east-1:00000000000:secret:Secret02-abcdefg"),
		Name: aws.String("Secret02"),
	},
	&secretsmanager.SecretListEntry{
		ARN:  aws.String("arn:aws:secretsmanager:us-east-1:00000000000:secret:Secret03-abcdefg"),
		Name: aws.String("Secret03"),
	},
}

var secretList2 = []*secretsmanager.SecretListEntry{
	&secretsmanager.SecretListEntry{
		ARN:  aws.String("arn:aws:secretsmanager:us-east-1:00000000000:secret:Secret11-abcdefg"),
		Name: aws.String("Secret11"),
	},
	&secretsmanager.SecretListEntry{
		ARN:  aws.String("arn:aws:secretsmanager:us-east-1:00000000000:secret:Secret12-abcdefg"),
		Name: aws.String("Secret12"),
	},
	&secretsmanager.SecretListEntry{
		ARN:  aws.String("arn:aws:secretsmanager:us-east-1:00000000000:secret:Secret13-abcdefg"),
		Name: aws.String("Secret13"),
	},
}

var secretList3 = []*secretsmanager.SecretListEntry{
	&secretsmanager.SecretListEntry{
		ARN:  aws.String("arn:aws:secretsmanager:us-east-1:00000000000:secret:Secret21-abcdefg"),
		Name: aws.String("Secret21"),
	},
	&secretsmanager.SecretListEntry{
		ARN:  aws.String("arn:aws:secretsmanager:us-east-1:00000000000:secret:Secret22-abcdefg"),
		Name: aws.String("Secret22"),
	},
	&secretsmanager.SecretListEntry{
		ARN:  aws.String("arn:aws:secretsmanager:us-east-1:00000000000:secret:Secret23-abcdefg"),
		Name: aws.String("Secret23"),
	},
}

var testSecretsList = []*secretsmanager.ListSecretsOutput{
	&secretsmanager.ListSecretsOutput{
		SecretList: secretList1,
		NextToken:  aws.String("1"),
	},
	&secretsmanager.ListSecretsOutput{
		SecretList: secretList2,
		NextToken:  aws.String("2"),
	},
	&secretsmanager.ListSecretsOutput{
		SecretList: secretList3,
	},
}

func (m *mockSecretsManagerClient) ListSecretsWithContext(ctx context.Context, input *secretsmanager.ListSecretsInput, opts ...request.Option) (*secretsmanager.ListSecretsOutput, error) {
	if m.err != nil {
		return nil, m.err
	}

	if aws.StringValue(input.NextToken) == "" {
		return testSecretsList[0], nil
	}

	next, err := strconv.Atoi(aws.StringValue(input.NextToken))
	if err != nil {
		return nil, err
	}

	if next <= len(testSecretsList) {
		return testSecretsList[next], nil
	}

	return nil, awserr.New(secretsmanager.ErrCodeInvalidNextTokenException, "invalid next token", nil)
}

func (m *mockSecretsManagerClient) CreateSecretWithContext(ctx context.Context, input *secretsmanager.CreateSecretInput, opts ...request.Option) (*secretsmanager.CreateSecretOutput, error) {
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

	return &secretsmanager.CreateSecretOutput{
		ARN:       aws.String("arn:foobar"),
		Name:      aws.String("foobar"),
		VersionId: aws.String("v1"),
	}, nil
}

func TestListSecretsWithFilter(t *testing.T) {
	s := SecretsManager{Service: newmockSecretsManagerClient(t, nil)}

	var expected []*string
	for _, list := range [][]*secretsmanager.SecretListEntry{secretList1, secretList2, secretList3} {
		for _, s := range list {
			expected = append(expected, s.ARN)
		}
	}

	out, err := s.ListSecretsWithFilter(context.TODO(), func(secret *secretsmanager.SecretListEntry) bool {
		return true
	})

	if err != nil {
		t.Errorf("unexpected error %s", err)
	}

	if !reflect.DeepEqual(out, expected) {
		t.Errorf("expected %+v, got %+v", expected, out)
	}

	expected = []*string{}
	out, err = s.ListSecretsWithFilter(context.TODO(), func(secret *secretsmanager.SecretListEntry) bool {
		return false
	})

	if err != nil {
		t.Errorf("unexpected error %s", err)
	}

	if !reflect.DeepEqual(out, expected) {
		t.Errorf("expected %+v, got %+v", expected, out)
	}

	// secretsmanager.ErrCodeInternalServiceError
	s.Service.(*mockSecretsManagerClient).err = awserr.New(secretsmanager.ErrCodeInternalServiceError, "Internal Error", nil)
	_, err = s.ListSecretsWithFilter(context.TODO(), func(secret *secretsmanager.SecretListEntry) bool {
		return true
	})
	if aerr, ok := err.(apierror.Error); ok {
		if aerr.Code != apierror.ErrInternalError {
			t.Errorf("expected error code %s, got: %s", apierror.ErrInternalError, aerr.Code)
		}
	} else {
		t.Errorf("expected apierror.Error, got: %s", reflect.TypeOf(err).String())
	}
}

func TestCreateSecrets(t *testing.T) {
	s := SecretsManager{Service: newmockSecretsManagerClient(t, nil)}
	expected := &secretsmanager.CreateSecretOutput{
		ARN:       aws.String("arn:foobar"),
		Name:      aws.String("foobar"),
		VersionId: aws.String("v1"),
	}

	out, err := s.CreateSecret(context.TODO(), &secretsmanager.CreateSecretInput{
		Name:         aws.String("foobar"),
		SecretString: aws.String("top sekrit"),
	})
	if err != nil {
		t.Errorf("expected nil error, got %s", err)
	}

	if !reflect.DeepEqual(out, expected) {
		t.Errorf("expected %+v, got %+v", expected, out)
	}

	if _, err = s.CreateSecret(context.TODO(), nil); err == nil {
		t.Error("expected error for nil input, got nil")
	}

	if _, err = s.CreateSecret(context.TODO(), &secretsmanager.CreateSecretInput{
		Name:         aws.String("foobar"),
		SecretString: aws.String("top sekrit"),
		SecretBinary: []byte("moar sekrit"),
	}); err == nil {
		t.Error("expected error for bad input, got nil")
	}

	// test an error from the api secretsmanager.ErrCodeInternalServiceError
	s.Service.(*mockSecretsManagerClient).err = awserr.New(secretsmanager.ErrCodeInternalServiceError, "Internal Error", nil)
	_, err = s.CreateSecret(context.TODO(), &secretsmanager.CreateSecretInput{
		Name:         aws.String("foobar"),
		SecretString: aws.String("top sekrit"),
	})
	if aerr, ok := err.(apierror.Error); ok {
		if aerr.Code != apierror.ErrInternalError {
			t.Errorf("expected error code %s, got: %s", apierror.ErrInternalError, aerr.Code)
		}
	} else {
		t.Errorf("expected apierror.Error, got: %s", reflect.TypeOf(err).String())
	}
}
