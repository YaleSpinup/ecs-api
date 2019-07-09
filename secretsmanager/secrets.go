package secretsmanager

import (
	"context"
	"fmt"

	"github.com/YaleSpinup/s3-api/apierror"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/secretsmanager"
	log "github.com/sirupsen/logrus"
)

// CreateSecret creates a secret in the secretsmanager
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

// ListSecretsWithFilter lists all of the secrets with a passed filter function
func (s *SecretsManager) ListSecretsWithFilter(ctx context.Context, filter func(*secretsmanager.SecretListEntry) bool) ([]*string, error) {
	log.Info("listing secretsmanager secrets")
	secrets := []*string{}

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
				secrets = append(secrets, secret.ARN)
			}
		}
		next = aws.StringValue(out.NextToken)
		i++
	}

	return secrets, nil
}

// GetSecretMetaDataWithFilter describes a secret (doesn't return the actual secret) and requires a filter function to be passed.  This function
// can be used (for example) to ensure the returned secret has certain tags or was encrypted with a specific CMK
func (s *SecretsManager) GetSecretMetaDataWithFilter(ctx context.Context, id string, filter func(*secretsmanager.DescribeSecretOutput) bool) (*secretsmanager.DescribeSecretOutput, error) {
	if id == "" {
		return nil, apierror.New(apierror.ErrBadRequest, "invalid input", nil)
	}

	log.Infof("describing secretsmanager secret %s", id)

	out, err := s.Service.DescribeSecretWithContext(ctx, &secretsmanager.DescribeSecretInput{SecretId: aws.String(id)})
	if err != nil {
		return nil, ErrCode("failed to describe secret", err)
	}

	if filter(out) {
		return out, nil
	}

	return nil, apierror.New(apierror.ErrNotFound, "no secret matching filter", nil)
}

// DeleteSecret marks a secret for deletion. Optionally, the secret can be forcefully deleted.
func (s *SecretsManager) DeleteSecret(ctx context.Context, id string, window int64) (*secretsmanager.DeleteSecretOutput, error) {
	if id == "" {
		return nil, apierror.New(apierror.ErrBadRequest, "invalid input", nil)
	}

	input := secretsmanager.DeleteSecretInput{SecretId: aws.String(id)}
	if window == 0 {
		input.ForceDeleteWithoutRecovery = aws.Bool(true)
	} else {
		if window < 7 || window > 30 {
			return nil, apierror.New(apierror.ErrBadRequest, "recovery window must be between 7 and 30 days", nil)
		}
		input.RecoveryWindowInDays = aws.Int64(window)
	}

	out, err := s.Service.DeleteSecretWithContext(ctx, &input)
	if err != nil {
		msg := fmt.Sprintf("failed to delete secret with id %s", id)
		return nil, ErrCode(msg, err)
	}

	return out, nil
}
