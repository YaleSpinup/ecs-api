package secretsmanager

import (
	"context"

	"github.com/YaleSpinup/s3-api/apierror"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/secretsmanager"
	log "github.com/sirupsen/logrus"
)

// CreateSecrets creates a secret in the secretsmanager
func (s *SecretsManager) CreateSecret(ctx context.Context, input *secretsmanager.CreateSecretInput) (*secretsmanager.CreateSecretOutput, error) {
	if input == nil {
		return nil, apierror.New(apierror.ErrBadRequest, "invalid input", nil)
	}

	if input.SecretBinary == nil && input.SecretString == nil {
		return nil, apierror.New(apierror.ErrBadRequest, "invalid input, one of secretstring or secretbinary are required", nil)
	}

	if input.SecretBinary != nil && input.SecretString != nil {
		return nil, apierror.New(apierror.ErrBadRequest, "invalid input, ONLY one of secretstring or secretbinary are allowed", nil)
	}

	log.Infof("creating secret %s", aws.StringValue(input.Name))

	out, err := s.Service.CreateSecretWithContext(ctx, input)
	if err != nil {
		return nil, ErrCode("failed to create secret", err)
	}

	return out, nil
}

// ListSecretsWithFIlter lists all of the secrets with a passed filter function
func (s *SecretsManager) ListSecretsWithFilter(ctx context.Context, filter func(*secretsmanager.SecretListEntry) bool) ([]*secretsmanager.SecretListEntry, error) {
	log.Infof("listing secretsmanager secrets")
	secrets := []*secretsmanager.SecretListEntry{}

	i := 0
	next := ""
	for i == 0 || next != "" {
		input := secretsmanager.ListSecretsInput{MaxResults: aws.Int64(100)}
		if next != "" {
			input.NextToken = aws.String(next)
		}

		out, err := s.Service.ListSecretsWithContext(ctx, &input)
		if err != nil {
			return secrets, ErrCode("failed to list secrets", err)
		}

		for _, secret := range out.SecretList {
			if filter(secret) {
				secrets = append(secrets, secret)
			}
		}
		next = aws.StringValue(out.NextToken)
		i++
	}

	return secrets, nil
}
