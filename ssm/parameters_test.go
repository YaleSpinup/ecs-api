package ssm

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"

	"github.com/YaleSpinup/ecs-api/apierror"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/request"
	"github.com/aws/aws-sdk-go/service/ssm"
)

type testParam struct {
	Param *ssm.Parameter
	Tags  []*ssm.Tag
}

var (
	now    = time.Now()
	org    = "test"
	prefix = "xochitl"

	testParam1 = testParam{
		Param: &ssm.Parameter{
			ARN:              aws.String("arn:aws:ssm:us-east-1:846761448161:parameter/" + org + "/" + prefix + "/newsecret1"),
			LastModifiedDate: aws.Time(now),
			Name:             aws.String("/newsecret1"),
			Type:             aws.String("SecureString"),
			Value:            aws.String("xxxxxxxx"),
			Version:          aws.Int64(3),
		},
		Tags: []*ssm.Tag{
			&ssm.Tag{
				Key:   aws.String("ice"),
				Value: aws.String("cream"),
			},
			&ssm.Tag{
				Key:   aws.String("mashed"),
				Value: aws.String("potatoes"),
			},
		},
	}

	testParam2 = testParam{
		Param: &ssm.Parameter{
			ARN:              aws.String("arn:aws:ssm:us-east-1:846761448161:parameter/" + org + "/" + prefix + "/newsecret2"),
			LastModifiedDate: aws.Time(now),
			Name:             aws.String("/newsecret2"),
			Type:             aws.String("SecureString"),
			Value:            aws.String("yyyyyyyyy"),
			Version:          aws.Int64(2),
		},
		Tags: []*ssm.Tag{
			&ssm.Tag{
				Key:   aws.String("peanut"),
				Value: aws.String("butter"),
			},
			&ssm.Tag{
				Key:   aws.String("meant"),
				Value: aws.String("foryou"),
			},
		},
	}

	testParam3 = testParam{
		Param: &ssm.Parameter{
			ARN:              aws.String("arn:aws:ssm:us-east-1:846761448161:parameter/" + org + "/" + prefix + "/newsecret3"),
			LastModifiedDate: aws.Time(now),
			Name:             aws.String("/newsecret3"),
			Type:             aws.String("SecureString"),
			Value:            aws.String("zzzzzzzzz"),
			Version:          aws.Int64(1),
		},
		Tags: []*ssm.Tag{
			&ssm.Tag{
				Key:   aws.String("ok"),
				Value: aws.String("yeah"),
			},
			&ssm.Tag{
				Key:   aws.String("hello"),
				Value: aws.String("nope"),
			},
		},
	}
)

func (m *mockSSMClient) GetParametersByPathWithContext(ctx context.Context, input *ssm.GetParametersByPathInput, opts ...request.Option) (*ssm.GetParametersByPathOutput, error) {
	if m.err != nil {
		return nil, m.err
	}
	out := &ssm.GetParametersByPathOutput{}
	if aws.StringValue(input.NextToken) == "" {
		out.Parameters = []*ssm.Parameter{testParam1.Param, testParam2.Param}
		out.NextToken = aws.String("2")
	} else if aws.StringValue(input.NextToken) == "2" {
		out.Parameters = []*ssm.Parameter{testParam3.Param}
	} else {
		return nil, nil
	}

	return out, nil
}

func (m *mockSSMClient) DescribeParametersWithContext(ctx context.Context, input *ssm.DescribeParametersInput, opts ...request.Option) (*ssm.DescribeParametersOutput, error) {
	if m.err != nil {
		return nil, m.err
	}

	for _, p := range []testParam{testParam1, testParam2, testParam3} {
		if org+"/"+prefix+"/"+aws.StringValue(p.Param.Name) == aws.StringValue(input.ParameterFilters[0].Values[0]) {
			return &ssm.DescribeParametersOutput{
				Parameters: []*ssm.ParameterMetadata{
					&ssm.ParameterMetadata{
						Name:             p.Param.Name,
						KeyId:            p.Param.ARN,
						LastModifiedDate: p.Param.LastModifiedDate,
					},
				},
			}, nil
		}
	}

	return &ssm.DescribeParametersOutput{}, awserr.New(ssm.ErrCodeParameterNotFound, "not found", nil)
}

func (m *mockSSMClient) GetParameterWithContext(ctx context.Context, input *ssm.GetParameterInput, opts ...request.Option) (*ssm.GetParameterOutput, error) {
	if m.err != nil {
		return nil, m.err
	}

	for _, p := range []testParam{testParam1, testParam2, testParam3} {
		if org+"/"+prefix+"/"+aws.StringValue(p.Param.Name) == aws.StringValue(input.Name) {
			return &ssm.GetParameterOutput{
				Parameter: p.Param,
			}, nil
		}
	}

	return &ssm.GetParameterOutput{}, awserr.New(ssm.ErrCodeParameterNotFound, "not found", nil)
}

func (m *mockSSMClient) PutParameterWithContext(ctx context.Context, input *ssm.PutParameterInput, opts ...request.Option) (*ssm.PutParameterOutput, error) {
	if m.err != nil {
		return nil, m.err
	}

	return &ssm.PutParameterOutput{}, nil
}

func (m *mockSSMClient) DeleteParameterWithContext(ctx context.Context, input *ssm.DeleteParameterInput, opts ...request.Option) (*ssm.DeleteParameterOutput, error) {
	if m.err != nil {
		return nil, m.err
	}

	return &ssm.DeleteParameterOutput{}, nil
}

func (m *mockSSMClient) ListTagsForResourceWithContext(ctx context.Context, input *ssm.ListTagsForResourceInput, opts ...request.Option) (*ssm.ListTagsForResourceOutput, error) {
	if m.err != nil {
		return nil, m.err
	}

	if aws.StringValue(input.ResourceType) != "Parameter" {
		return &ssm.ListTagsForResourceOutput{}, errors.New("bad request")
	}

	for _, p := range []testParam{testParam1, testParam2, testParam3} {
		if org+"/"+prefix+"/"+aws.StringValue(p.Param.Name) == aws.StringValue(input.ResourceId) {
			return &ssm.ListTagsForResourceOutput{
				TagList: p.Tags,
			}, nil
		}
	}

	return &ssm.ListTagsForResourceOutput{}, awserr.New(ssm.ErrCodeParameterNotFound, "not found", nil)
}

func (m *mockSSMClient) AddTagsToResourceWithContext(ctx context.Context, input *ssm.AddTagsToResourceInput, opts ...request.Option) (*ssm.AddTagsToResourceOutput, error) {
	if m.err != nil {
		return nil, m.err
	}

	if aws.StringValue(input.ResourceType) != "Parameter" {
		return &ssm.AddTagsToResourceOutput{}, errors.New("bad request")
	}

	for _, p := range []testParam{testParam1, testParam2, testParam3} {
		if aws.StringValue(p.Param.Name) == aws.StringValue(input.ResourceId) {
			return &ssm.AddTagsToResourceOutput{}, nil
		}
	}

	return &ssm.AddTagsToResourceOutput{}, awserr.New(ssm.ErrCodeParameterNotFound, "not found", nil)
}

func TestListParametersByPath(t *testing.T) {
	p := SSM{Service: newmockSSMClient(t, nil)}
	path := "/" + org + "/" + prefix

	expected := []string{}
	for _, t := range []*ssm.Parameter{testParam1.Param, testParam2.Param, testParam3.Param} {
		expected = append(expected, aws.StringValue(t.Name))
	}

	out, err := p.ListParametersByPath(context.TODO(), path)
	if err != nil {
		t.Errorf("unexpected error %s", err)
	}

	if !reflect.DeepEqual(expected, out) {
		t.Errorf("expected %+v, got %+v", expected, out)
	}

	// test empty path
	_, err = p.ListParametersByPath(context.TODO(), "")
	if err == nil {
		t.Error("expected error for empty path, got nil")
	}

	// ssm.ErrCodeInternalServiceError
	p.Service.(*mockSSMClient).err = awserr.New(ssm.ErrCodeInternalServerError, "Internal Error", nil)
	_, err = p.ListParametersByPath(context.TODO(), path)
	if aerr, ok := err.(apierror.Error); ok {
		if aerr.Code != apierror.ErrInternalError {
			t.Errorf("expected error code %s, got: %s", apierror.ErrInternalError, aerr.Code)
		}
	} else {
		t.Errorf("expected apierror.Error, got: %s", reflect.TypeOf(err).String())
	}
}

func TestGetParameterMetadata(t *testing.T) {
	p := SSM{Service: newmockSSMClient(t, nil)}
	expected := &ssm.ParameterMetadata{
		Name:             testParam1.Param.Name,
		KeyId:            testParam1.Param.ARN,
		LastModifiedDate: testParam1.Param.LastModifiedDate,
	}

	out, err := p.GetParameterMetadata(context.TODO(), org+"/"+prefix, aws.StringValue(testParam1.Param.Name))
	if err != nil {
		t.Errorf("unexpected error %s", err)
	}

	if !reflect.DeepEqual(expected, out) {
		t.Errorf("expected %+v, got %+v", expected, out)
	}

	// test param that doesn't exist
	_, err = p.GetParameterMetadata(context.TODO(), org+"/"+prefix, "foobar")
	if err == nil {
		t.Error("expected error for not found name, got nil")
	}

	// test empty prefix
	if _, err = p.GetParameterMetadata(context.TODO(), "", "foobar"); err == nil {
		t.Error("expected error for empty name, got nil")
	}

	// test empty name
	if _, err = p.GetParameterMetadata(context.TODO(), org+"/"+prefix, ""); err == nil {
		t.Error("expected error for empty name, got nil")
	}

	p.Service.(*mockSSMClient).err = awserr.New(ssm.ErrCodeInternalServerError, "Internal Error", nil)
	_, err = p.GetParameterMetadata(context.TODO(), org+"/"+prefix, "foobar")
	if err == nil {
		t.Error("expected error for empty path, got nil")
	}
}

func TestGetParameter(t *testing.T) {
	p := SSM{Service: newmockSSMClient(t, nil)}
	expected := testParam1.Param

	out, err := p.GetParameter(context.TODO(), org+"/"+prefix, aws.StringValue(testParam1.Param.Name))
	if err != nil {
		t.Errorf("unexpected error %s", err)
	}

	if !reflect.DeepEqual(expected, out) {
		t.Errorf("expected %+v, got %+v", expected, out)
	}

	// test param that doesn't exist
	_, err = p.GetParameter(context.TODO(), org+"/"+prefix, "foobar")
	if err == nil {
		t.Error("expected error for not found name, got nil")
	}

	// test empty prefix
	if _, err = p.GetParameter(context.TODO(), "", "foobar"); err == nil {
		t.Error("expected error for empty name, got nil")
	}

	// test empty name
	if _, err = p.GetParameter(context.TODO(), org+"/"+prefix, ""); err == nil {
		t.Error("expected error for empty name, got nil")
	}

	p.Service.(*mockSSMClient).err = awserr.New(ssm.ErrCodeInternalServerError, "Internal Error", nil)
	_, err = p.GetParameter(context.TODO(), org+"/"+prefix, "foobar")
	if err == nil {
		t.Error("expected error for empty path, got nil")
	}
}

func TestCreateParameter(t *testing.T) {
	p := SSM{Service: newmockSSMClient(t, nil)}

	if err := p.CreateParameter(context.TODO(), &ssm.PutParameterInput{}); err != nil {
		t.Errorf("expected nil error, not %s", err)
	}

	// test nil input
	if err := p.CreateParameter(context.TODO(), nil); err == nil {
		t.Error("expected error for nil input, got nil")
	}

	p.Service.(*mockSSMClient).err = awserr.New(ssm.ErrCodeInternalServerError, "Internal Error", nil)
	err := p.CreateParameter(context.TODO(), &ssm.PutParameterInput{})
	if aerr, ok := err.(apierror.Error); ok {
		if aerr.Code != apierror.ErrInternalError {
			t.Errorf("expected error code %s, got: %s", apierror.ErrInternalError, aerr.Code)
		}
	} else {
		t.Errorf("expected apierror.Error, got: %s", reflect.TypeOf(err).String())
	}
}

func TestDeleteParameter(t *testing.T) {
	p := SSM{Service: newmockSSMClient(t, nil)}

	if err := p.DeleteParameter(context.TODO(), "testParam1"); err != nil {
		t.Errorf("expected nil error, not %s", err)
	}

	// test empty path
	if err := p.DeleteParameter(context.TODO(), ""); err == nil {
		t.Error("expected error for empty path, got nil")
	}

	p.Service.(*mockSSMClient).err = awserr.New(ssm.ErrCodeInternalServerError, "Internal Error", nil)
	err := p.DeleteParameter(context.TODO(), "testParam1")
	if aerr, ok := err.(apierror.Error); ok {
		if aerr.Code != apierror.ErrInternalError {
			t.Errorf("expected error code %s, got: %s", apierror.ErrInternalError, aerr.Code)
		}
	} else {
		t.Errorf("expected apierror.Error, got: %s", reflect.TypeOf(err).String())
	}
}

func TestListParameterTags(t *testing.T) {
	p := SSM{Service: newmockSSMClient(t, nil)}

	for _, param := range []testParam{testParam1, testParam2, testParam3} {
		expected := param.Tags
		out, err := p.ListParameterTags(context.TODO(), org+"/"+prefix, aws.StringValue(param.Param.Name))
		if err != nil {
			t.Errorf("expected nil error, got %s", err)
		}

		if !reflect.DeepEqual(out, expected) {
			t.Errorf("expected %+v, got %+v", expected, out)
		}
	}

	// test empty prefix
	if _, err := p.ListParameterTags(context.TODO(), "", "foobar"); err == nil {
		t.Error("expected error for empty id, got nil")
	}

	// test empty name
	if _, err := p.ListParameterTags(context.TODO(), org+"/"+prefix, ""); err == nil {
		t.Error("expected error for empty id, got nil")
	}

	// missing
	if _, err := p.ListParameterTags(context.TODO(), org+"/"+prefix, "foobar"); err == nil {
		t.Error("expected error for empty id, got nil")
	}

	p.Service.(*mockSSMClient).err = awserr.New(ssm.ErrCodeInternalServerError, "Internal Error", nil)
	_, err := p.ListParameterTags(context.TODO(), org+"/"+prefix, aws.StringValue(testParam1.Param.Name))
	if aerr, ok := err.(apierror.Error); ok {
		if aerr.Code != apierror.ErrInternalError {
			t.Errorf("expected error code %s, got: %s", apierror.ErrInternalError, aerr.Code)
		}
	} else {
		t.Errorf("expected apierror.Error, got: %s", reflect.TypeOf(err).String())
	}

}

func TestUpdateParameter(t *testing.T) {
	p := SSM{Service: newmockSSMClient(t, nil)}

	if err := p.UpdateParameter(context.TODO(), &ssm.PutParameterInput{}); err != nil {
		t.Errorf("expected nil error, not %s", err)
	}

	// test nil input
	if err := p.UpdateParameter(context.TODO(), nil); err == nil {
		t.Error("expected error for nil input, got nil")
	}

	p.Service.(*mockSSMClient).err = awserr.New(ssm.ErrCodeInternalServerError, "Internal Error", nil)
	err := p.UpdateParameter(context.TODO(), &ssm.PutParameterInput{})
	if aerr, ok := err.(apierror.Error); ok {
		if aerr.Code != apierror.ErrInternalError {
			t.Errorf("expected error code %s, got: %s", apierror.ErrInternalError, aerr.Code)
		}
	} else {
		t.Errorf("expected apierror.Error, got: %s", reflect.TypeOf(err).String())
	}
}

func TestUpdateParameterTags(t *testing.T) {
	p := SSM{Service: newmockSSMClient(t, nil)}

	for _, param := range []testParam{testParam1, testParam2, testParam3} {
		if err := p.UpdateParameterTags(context.TODO(), aws.StringValue(param.Param.Name), []*ssm.Tag{&ssm.Tag{Key: aws.String("foo"), Value: aws.String("bar")}}); err != nil {
			t.Errorf("expected nil error, got %s", err)
		}
	}

	// test empty id
	if err := p.UpdateParameterTags(context.TODO(), "", []*ssm.Tag{&ssm.Tag{Key: aws.String("foo"), Value: aws.String("bar")}}); err == nil {
		t.Error("expected error, got nil")
	}

	// test empty tags slice
	if err := p.UpdateParameterTags(context.TODO(), "foobar", []*ssm.Tag{}); err == nil {
		t.Error("expected error for nil input, got nil")
	}

	p.Service.(*mockSSMClient).err = awserr.New(ssm.ErrCodeInternalServerError, "Internal Error", nil)
	err := p.UpdateParameterTags(context.TODO(), "foobar", []*ssm.Tag{&ssm.Tag{Key: aws.String("foo"), Value: aws.String("bar")}})
	if aerr, ok := err.(apierror.Error); ok {
		if aerr.Code != apierror.ErrInternalError {
			t.Errorf("expected error code %s, got: %s", apierror.ErrInternalError, aerr.Code)
		}
	} else {
		t.Errorf("expected apierror.Error, got: %s", reflect.TypeOf(err).String())
	}
}
