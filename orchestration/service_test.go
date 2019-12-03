package orchestration

import (
	"errors"
	"fmt"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/request"
	"github.com/aws/aws-sdk-go/service/servicediscovery"
)

func (m *mockSDClient) CreateServiceWithContext(ctx aws.Context, input *servicediscovery.CreateServiceInput, opts ...request.Option) (*servicediscovery.CreateServiceOutput, error) {
	if aws.StringValue(input.Name) == "goodsd" {
		return &servicediscovery.CreateServiceOutput{
			Service: goodSd,
		}, nil
	}
	msg := fmt.Sprintf("Failed to get mock service discovery service %s", aws.StringValue(input.Name))
	return nil, errors.New(msg)
}

func (m *mockSDClient) GetServiceWithContext(ctx aws.Context, input *servicediscovery.GetServiceInput, opts ...request.Option) (*servicediscovery.GetServiceOutput, error) {
	if aws.StringValue(input.Id) == "srv-goodsd" {
		return &servicediscovery.GetServiceOutput{
			Service: goodSd,
		}, nil
	}
	msg := fmt.Sprintf("Failed to get mock service discovery service %s", aws.StringValue(input.Id))
	return nil, errors.New(msg)
}
