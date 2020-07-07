package secretsmanager

import (
	"context"
	"reflect"
	"strconv"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/request"
	"github.com/aws/aws-sdk-go/service/secretsmanager"

	"github.com/YaleSpinup/ecs-api/apierror"
)

var now = time.Now()

var secretMeta1 = &secretsmanager.DescribeSecretOutput{
	ARN:             aws.String("arn:aws:secretsmanager:us-east-1:00000000000:secret:Secret01-abcdefg"),
	Name:            aws.String("Secret01"),
	LastChangedDate: &now,
	Tags: []*secretsmanager.Tag{
		{
			Key:   aws.String("spinup:org"),
			Value: aws.String("test"),
		},
		{
			Key:   aws.String("Application"),
			Value: aws.String("Spinup"),
		},
		{
			Key:   aws.String("Foo"),
			Value: aws.String("Bar"),
		},
	},
	VersionIdsToStages: map[string][]*string{
		"00000000-1111-2222-3333-444444444444": {
			aws.String("AWSCURRENT"),
		},
	},
}

var secretMeta2 = &secretsmanager.DescribeSecretOutput{
	ARN:             aws.String("arn:aws:secretsmanager:us-east-1:00000000000:secret:Secret02-abcdefg"),
	Name:            aws.String("Secret02"),
	LastChangedDate: &now,
	Tags: []*secretsmanager.Tag{
		{
			Key:   aws.String("spinup:org"),
			Value: aws.String("test"),
		},
		{
			Key:   aws.String("Application"),
			Value: aws.String("Spinup"),
		},
	},
	VersionIdsToStages: map[string][]*string{
		"00000000-1111-2222-3333-444444444444": {
			aws.String("AWSCURRENT"),
		},
	},
}

var secretMeta3 = &secretsmanager.DescribeSecretOutput{
	ARN:             aws.String("arn:aws:secretsmanager:us-east-1:00000000000:secret:Secret03-abcdefg"),
	Name:            aws.String("Secret03"),
	LastChangedDate: &now,
	Tags: []*secretsmanager.Tag{
		{
			Key:   aws.String("spinup:org"),
			Value: aws.String("prod"),
		},
		{
			Key:   aws.String("Application"),
			Value: aws.String("Spinup"),
		},
	},
	VersionIdsToStages: map[string][]*string{
		"00000000-1111-2222-3333-444444444444": {
			aws.String("AWSCURRENT"),
		},
	},
}

var secretList1 = []*secretsmanager.SecretListEntry{
	{
		ARN:  aws.String("arn:aws:secretsmanager:us-east-1:00000000000:secret:Secret01-abcdefg"),
		Name: aws.String("Secret01"),
	},
	{
		ARN:  aws.String("arn:aws:secretsmanager:us-east-1:00000000000:secret:Secret02-abcdefg"),
		Name: aws.String("Secret02"),
	},
	{
		ARN:  aws.String("arn:aws:secretsmanager:us-east-1:00000000000:secret:Secret03-abcdefg"),
		Name: aws.String("Secret03"),
	},
}

var secretList2 = []*secretsmanager.SecretListEntry{
	{
		ARN:  aws.String("arn:aws:secretsmanager:us-east-1:00000000000:secret:Secret11-abcdefg"),
		Name: aws.String("Secret11"),
	},
	{
		ARN:  aws.String("arn:aws:secretsmanager:us-east-1:00000000000:secret:Secret12-abcdefg"),
		Name: aws.String("Secret12"),
	},
	{
		ARN:  aws.String("arn:aws:secretsmanager:us-east-1:00000000000:secret:Secret13-abcdefg"),
		Name: aws.String("Secret13"),
	},
}

var secretList3 = []*secretsmanager.SecretListEntry{
	{
		ARN:  aws.String("arn:aws:secretsmanager:us-east-1:00000000000:secret:Secret21-abcdefg"),
		Name: aws.String("Secret21"),
	},
	{
		ARN:  aws.String("arn:aws:secretsmanager:us-east-1:00000000000:secret:Secret22-abcdefg"),
		Name: aws.String("Secret22"),
	},
	{
		ARN:  aws.String("arn:aws:secretsmanager:us-east-1:00000000000:secret:Secret23-abcdefg"),
		Name: aws.String("Secret23"),
	},
}

var testSecretsList = []*secretsmanager.ListSecretsOutput{
	{
		SecretList: secretList1,
		NextToken:  aws.String("1"),
	},
	{
		SecretList: secretList2,
		NextToken:  aws.String("2"),
	},
	{
		SecretList: secretList3,
	},
}

func (m *mockSecretsManagerClient) DescribeSecretWithContext(ctx context.Context, input *secretsmanager.DescribeSecretInput, opts ...request.Option) (*secretsmanager.DescribeSecretOutput, error) {
	if m.err != nil {
		return nil, m.err
	}

	for _, s := range []*secretsmanager.DescribeSecretOutput{secretMeta1, secretMeta2, secretMeta3} {
		if aws.StringValue(input.SecretId) == aws.StringValue(s.ARN) {
			return s, nil
		}
	}

	return nil, awserr.New(secretsmanager.ErrCodeResourceNotFoundException, "Secret not found", nil)
}

func (m *mockSecretsManagerClient) DeleteSecretWithContext(ctx context.Context, input *secretsmanager.DeleteSecretInput, opts ...request.Option) (*secretsmanager.DeleteSecretOutput, error) {
	if m.err != nil {
		return nil, m.err
	}

	deleteDate := now.Add(time.Duration(aws.Int64Value(input.RecoveryWindowInDays) * 24))
	for _, s := range []*secretsmanager.DescribeSecretOutput{secretMeta1, secretMeta2, secretMeta3} {
		if aws.StringValue(input.SecretId) == aws.StringValue(s.ARN) {
			return &secretsmanager.DeleteSecretOutput{
				ARN:          s.ARN,
				DeletionDate: &deleteDate,
				Name:         s.Name,
			}, nil
		}
	}

	return nil, awserr.New(secretsmanager.ErrCodeResourceNotFoundException, "Secret not found", nil)
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

func (m *mockSecretsManagerClient) PutSecretValueWithContext(ctx context.Context, input *secretsmanager.PutSecretValueInput, opts ...request.Option) (*secretsmanager.PutSecretValueOutput, error) {
	if m.err != nil {
		return nil, m.err
	}

	if input == nil {
		return nil, awserr.New(secretsmanager.ErrCodeInvalidRequestException, "invalid input", nil)
	}

	if aws.StringValue(input.SecretId) == "" {
		return nil, awserr.New(secretsmanager.ErrCodeInvalidRequestException, "SecretId is required", nil)
	}

	if (input.SecretBinary == nil && input.SecretString == nil) || (input.SecretBinary != nil && input.SecretString != nil) {
		return nil, awserr.New(secretsmanager.ErrCodeInvalidRequestException, "secret string OR secretbinary is required", nil)
	}

	return &secretsmanager.PutSecretValueOutput{
		ARN:       aws.String("arn:foobar"),
		Name:      aws.String("foobar"),
		VersionId: aws.String("v1"),
		VersionStages: []*string{
			aws.String("AWSCURRENT"),
		},
	}, nil
}

func (m *mockSecretsManagerClient) TagResourceWithContext(ctx context.Context, input *secretsmanager.TagResourceInput, opts ...request.Option) (*secretsmanager.TagResourceOutput, error) {
	if m.err != nil {
		return nil, m.err
	}

	if input == nil {
		return nil, awserr.New(secretsmanager.ErrCodeInvalidRequestException, "invalid input", nil)
	}

	if aws.StringValue(input.SecretId) == "" {
		return nil, awserr.New(secretsmanager.ErrCodeInvalidRequestException, "SecretId is required", nil)
	}

	return &secretsmanager.TagResourceOutput{}, nil
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

func TestGetSecretMetaDataWithFilter(t *testing.T) {
	s := SecretsManager{Service: newmockSecretsManagerClient(t, nil)}

	out, err := s.GetSecretMetaDataWithFilter(context.TODO(), aws.StringValue(secretMeta1.ARN), func(filter *secretsmanager.DescribeSecretOutput) bool { return true })
	if err != nil {
		t.Errorf("unexpected error: %s", err)
	}

	if !reflect.DeepEqual(out, secretMeta1) {
		t.Errorf("expected %+v, got %+v", secretMeta1, out)
	}

	_, err = s.GetSecretMetaDataWithFilter(context.TODO(), aws.StringValue(secretMeta1.ARN), func(filter *secretsmanager.DescribeSecretOutput) bool { return false })
	if err == nil {
		t.Error("expected error returned when no matching secret")
	}

	if aerr, ok := err.(apierror.Error); !ok || aerr.Code != apierror.ErrNotFound {
		t.Errorf("expected apierr not found, got %s", err)
	}

	_, err = s.GetSecretMetaDataWithFilter(context.TODO(), "", func(filter *secretsmanager.DescribeSecretOutput) bool { return true })
	if err == nil {
		t.Error("expected error returned when id is empty string")
	}

	// test an error from the api secretsmanager.ErrCodeInternalServiceError
	s.Service.(*mockSecretsManagerClient).err = awserr.New(secretsmanager.ErrCodeInternalServiceError, "Internal Error", nil)
	_, err = s.GetSecretMetaDataWithFilter(context.TODO(), aws.StringValue(secretMeta1.ARN), func(filter *secretsmanager.DescribeSecretOutput) bool { return true })
	if aerr, ok := err.(apierror.Error); ok {
		if aerr.Code != apierror.ErrInternalError {
			t.Errorf("expected error code %s, got: %s", apierror.ErrInternalError, aerr.Code)
		}
	} else {
		t.Errorf("expected apierror.Error, got: %s", reflect.TypeOf(err).String())
	}
}

func TestDeleteSecret(t *testing.T) {
	s := SecretsManager{Service: newmockSecretsManagerClient(t, nil)}
	expected := &secretsmanager.DeleteSecretOutput{
		ARN:          secretMeta1.ARN,
		DeletionDate: &now,
		Name:         secretMeta1.Name,
	}

	out, err := s.DeleteSecret(context.TODO(), aws.StringValue(secretMeta1.ARN), int64(0))
	if err != nil {
		t.Errorf("unexpected error: %s", err)
	}

	if !reflect.DeepEqual(out, expected) {
		t.Errorf("expected %+v, got %+v", expected, out)
	}

	_, err = s.DeleteSecret(context.TODO(), aws.StringValue(secretMeta1.ARN), int64(1))
	if err == nil {
		t.Error("expected error returned when no window is not 0 or between 7 and 30")
	}

	if aerr, ok := err.(apierror.Error); !ok || aerr.Code != apierror.ErrBadRequest {
		t.Errorf("expected apierr bad request, got %s", err)
	}

	_, err = s.DeleteSecret(context.TODO(), "", int64(0))
	if err == nil {
		t.Error("expected error returned when id is empty string")
	}

	// test an error from the api secretsmanager.ErrCodeInternalServiceError
	s.Service.(*mockSecretsManagerClient).err = awserr.New(secretsmanager.ErrCodeInternalServiceError, "Internal Error", nil)
	_, err = s.DeleteSecret(context.TODO(), aws.StringValue(secretMeta1.ARN), int64(0))
	if aerr, ok := err.(apierror.Error); ok {
		if aerr.Code != apierror.ErrInternalError {
			t.Errorf("expected error code %s, got: %s", apierror.ErrInternalError, aerr.Code)
		}
	} else {
		t.Errorf("expected apierror.Error, got: %s", reflect.TypeOf(err).String())
	}
}

func TestUpdateSecrets(t *testing.T) {
	s := SecretsManager{Service: newmockSecretsManagerClient(t, nil)}
	expected := &secretsmanager.PutSecretValueOutput{
		ARN:       aws.String("arn:foobar"),
		Name:      aws.String("foobar"),
		VersionId: aws.String("v1"),
		VersionStages: []*string{
			aws.String("AWSCURRENT"),
		},
	}

	out, err := s.UpdateSecret(context.TODO(), &secretsmanager.PutSecretValueInput{
		SecretId:     aws.String("arn:foobar"),
		SecretString: aws.String("top sekrit"),
	})
	if err != nil {
		t.Errorf("expected nil error, got %s", err)
	}

	if !reflect.DeepEqual(out, expected) {
		t.Errorf("expected %+v, got %+v", expected, out)
	}

	if _, err = s.UpdateSecret(context.TODO(), nil); err == nil {
		t.Error("expected error for nil input, got nil")
	}

	if _, err = s.UpdateSecret(context.TODO(), &secretsmanager.PutSecretValueInput{
		SecretId:     aws.String("arn:foobar"),
		SecretString: aws.String("top sekrit"),
		SecretBinary: []byte("moar sekrit"),
	}); err == nil {
		t.Error("expected error for bad input, got nil")
	}

	// test an error from the api secretsmanager.ErrCodeInternalServiceError
	s.Service.(*mockSecretsManagerClient).err = awserr.New(secretsmanager.ErrCodeInternalServiceError, "Internal Error", nil)
	_, err = s.UpdateSecret(context.TODO(), &secretsmanager.PutSecretValueInput{
		SecretId:     aws.String("arn:foobar"),
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

func TestUpdateSecretTags(t *testing.T) {
	s := SecretsManager{Service: newmockSecretsManagerClient(t, nil)}

	tags := []*secretsmanager.Tag{
		{
			Key:   aws.String("foo"),
			Value: aws.String("bar"),
		},
		{
			Key:   aws.String("baz"),
			Value: aws.String("biz"),
		},
	}

	if err := s.UpdateSecretTags(context.TODO(), "arn:foobar", tags); err != nil {
		t.Errorf("expected nil error, got %s", err)
	}
}
