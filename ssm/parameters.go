package ssm

import (
	"context"
	"fmt"
	"strings"

	"github.com/YaleSpinup/ecs-api/apierror"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ssm"
	log "github.com/sirupsen/logrus"
)

// ListParametersByPath gets all of the parameters in a path recursively
func (s *SSM) ListParametersByPath(ctx context.Context, path string) ([]string, error) {
	if path == "" {
		return nil, apierror.New(apierror.ErrBadRequest, "invalid input", nil)
	}

	log.Infof("listing ssm parameter store params with path %s", path)

	params := []string{}
	i := 0
	next := ""
	for i == 0 || next != "" {
		input := ssm.GetParametersByPathInput{
			Path:       aws.String(path),
			Recursive:  aws.Bool(true),
			MaxResults: aws.Int64(10),
		}

		if next != "" {
			input.NextToken = aws.String(next)
		}

		out, err := s.Service.GetParametersByPathWithContext(ctx, &input)
		if err != nil {
			return params, ErrCode("failed to list parameters", err)
		}

		for _, p := range out.Parameters {
			log.Debugf("processing parameter in list %+v", p)
			params = append(params, strings.TrimPrefix(aws.StringValue(p.Name), path+"/"))
		}

		next = aws.StringValue(out.NextToken)
		i++
	}

	return params, nil
}

// GetParameterMetadata gets a parameters metadata
func (s *SSM) GetParameterMetadata(ctx context.Context, prefix, name string) (*ssm.ParameterMetadata, error) {
	if prefix == "" || name == "" {
		return nil, apierror.New(apierror.ErrBadRequest, "invalid input", nil)
	}

	path := fmt.Sprintf("%s/%s", prefix, name)

	log.Infof("describing ssm parameter store param with path %s", path)

	out, err := s.Service.DescribeParametersWithContext(ctx, &ssm.DescribeParametersInput{
		MaxResults: aws.Int64(1),
		ParameterFilters: []*ssm.ParameterStringFilter{
			&ssm.ParameterStringFilter{
				Key:    aws.String("Name"),
				Option: aws.String("Equals"),
				Values: []*string{aws.String(path)},
			},
		},
	})
	if err != nil {
		return nil, ErrCode("failed to get parameter", err)
	}

	if len(out.Parameters) == 0 || out.Parameters[0] == nil {
		return nil, apierror.New(apierror.ErrNotFound, "parameter not found", nil)
	}

	metadata := out.Parameters[0]
	metadata.Name = aws.String(name)

	return metadata, nil
}

// GetParameter gets the details of a parameter
func (s *SSM) GetParameter(ctx context.Context, prefix, name string) (*ssm.Parameter, error) {
	if prefix == "" || name == "" {
		return nil, apierror.New(apierror.ErrBadRequest, "invalid input", nil)
	}

	path := fmt.Sprintf("%s/%s", prefix, name)

	log.Infof("getting a ssm parameter store param with path %s", path)

	out, err := s.Service.GetParameterWithContext(ctx, &ssm.GetParameterInput{
		Name:           aws.String(path),
		WithDecryption: aws.Bool(false),
	})
	if err != nil {
		return nil, ErrCode("failed to get parameter", err)
	}

	return out.Parameter, nil
}

// CreateParameter creates a new parameter
func (s *SSM) CreateParameter(ctx context.Context, input *ssm.PutParameterInput) error {
	if input == nil {
		return apierror.New(apierror.ErrBadRequest, "invalid input", nil)
	}

	log.Infof("creating ssm parameter store params with name %s", aws.StringValue(input.Name))

	_, err := s.Service.PutParameterWithContext(ctx, input)
	if err != nil {
		return ErrCode("failed to create parameter", err)
	}

	return nil
}

// DeleteParameter deletes a parameter
func (s *SSM) DeleteParameter(ctx context.Context, name string) error {
	if name == "" {
		return apierror.New(apierror.ErrBadRequest, "invalid input", nil)
	}

	log.Infof("deleting ssm parameter store params with name %s", name)

	_, err := s.Service.DeleteParameterWithContext(ctx, &ssm.DeleteParameterInput{
		Name: aws.String(name),
	})
	if err != nil {
		return ErrCode("failed to delete parameter "+name, err)
	}

	return nil
}

// ListParameterTags gets a list of tags for the parameter
func (s *SSM) ListParameterTags(ctx context.Context, prefix, name string) ([]*ssm.Tag, error) {
	if prefix == "" || name == "" {
		return nil, apierror.New(apierror.ErrBadRequest, "invalid input", nil)
	}

	path := fmt.Sprintf("%s/%s", prefix, name)

	log.Infof("listing tags for ssm parameter %s", path)

	out, err := s.Service.ListTagsForResourceWithContext(ctx, &ssm.ListTagsForResourceInput{
		ResourceId:   aws.String(path),
		ResourceType: aws.String("Parameter"),
	})
	if err != nil {
		return []*ssm.Tag{}, ErrCode("failed to list parameter tags", err)
	}

	log.Debugf("returning %d tags for %s", len(out.TagList), path)

	return out.TagList, nil
}

// UpdateParameter creates a new parameter
func (s *SSM) UpdateParameter(ctx context.Context, input *ssm.PutParameterInput) error {
	if input == nil {
		return apierror.New(apierror.ErrBadRequest, "invalid input", nil)
	}

	log.Infof("creating ssm parameter store params with name %s", aws.StringValue(input.Name))

	_, err := s.Service.PutParameterWithContext(ctx, input)
	if err != nil {
		return ErrCode("failed to create parameter", err)
	}

	return nil
}

// UpdateParameterTags updates the tags for the parameter
func (s *SSM) UpdateParameterTags(ctx context.Context, id string, tags []*ssm.Tag) error {
	if len(tags) == 0 {
		return apierror.New(apierror.ErrBadRequest, "invalid input", nil)
	}

	log.Infof("updating tags for parameter %s", id)

	_, err := s.Service.AddTagsToResourceWithContext(ctx, &ssm.AddTagsToResourceInput{
		ResourceId:   aws.String(id),
		ResourceType: aws.String("Parameter"),
		Tags:         tags,
	})
	if err != nil {
		return ErrCode("failed to tag parameter", err)
	}

	return nil
}
