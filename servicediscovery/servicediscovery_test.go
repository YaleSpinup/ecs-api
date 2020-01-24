package servicediscovery

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"testing"

	"github.com/YaleSpinup/ecs-api/common"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/request"
	"github.com/aws/aws-sdk-go/service/servicediscovery"
	"github.com/aws/aws-sdk-go/service/servicediscovery/servicediscoveryiface"
)

var (
	goodSd = &servicediscovery.Service{
		Name: aws.String("goodsd"),
		Arn:  aws.String("arn:aws:servicediscovery:us-east-1:1234567890:service/srv-goodsd"),
		Id:   aws.String("srv-goodsd"),
		DnsConfig: &servicediscovery.DnsConfig{
			DnsRecords: []*servicediscovery.DnsRecord{
				&servicediscovery.DnsRecord{
					TTL:  aws.Int64(30),
					Type: aws.String("A"),
				},
			},
			NamespaceId: aws.String("ns-p5g6iyxdh5c5h3dr"),
		},
	}
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

// mockSDClient is a fake service discovery client
type mockSDClient struct {
	servicediscoveryiface.ServiceDiscoveryAPI
	t   *testing.T
	err error
}

func newmockSDClient(t *testing.T, err error) servicediscoveryiface.ServiceDiscoveryAPI {
	return &mockSDClient{
		t:   t,
		err: err,
	}
}

func TestNewSession(t *testing.T) {
	sd := NewSession(common.Account{})
	to := reflect.TypeOf(sd).String()
	if to != "servicediscovery.ServiceDiscovery" {
		t.Errorf("expected type to be 'servicediscovery.ServiceDiscovery', got %s", to)
	}
}

func TestCreateServiceDiscovery(t *testing.T) {
	client := ServiceDiscovery{Service: &mockSDClient{t: t}}
	sd, err := client.CreateServiceDiscoveryService(context.TODO(), &servicediscovery.CreateServiceInput{
		Name: aws.String("goodsd"),
		DnsConfig: &servicediscovery.DnsConfig{
			DnsRecords: []*servicediscovery.DnsRecord{
				&servicediscovery.DnsRecord{
					TTL:  aws.Int64(30),
					Type: aws.String("A"),
				},
			},
			NamespaceId: aws.String("ns-p5g6iyxdh5c5h3dr"),
		},
	})
	if err != nil {
		t.Fatal("expected no error from create service discovery service, got", err)
	}
	t.Log("Got service discovery create service output", sd)
	if !reflect.DeepEqual(sd, goodSd) {
		t.Fatalf("expected: %+v\nGot:%+v", goodSd, sd)
	}

	sd, err = client.CreateServiceDiscoveryService(context.TODO(), &servicediscovery.CreateServiceInput{
		Name: aws.String("badsd"),
	})
	if err == nil {
		t.Fatalf("expected error from bad create service discovery service, got %+v", sd)
	}
	t.Log("Got expected error from bad service discovery create service", err)
}

func TestGetServiceDiscovery(t *testing.T) {
	client := ServiceDiscovery{Service: &mockSDClient{t: t}}
	sd, err := client.GetServiceDiscoveryService(context.TODO(), aws.String("srv-goodsd"))
	if err != nil {
		t.Fatal("expected no error from get service discovery service, got", err)
	}
	t.Log("Got service discovery get service output", sd)
	if !reflect.DeepEqual(sd, goodSd) {
		t.Fatalf("expected: %+v\n Got: %+v", goodSd, sd)
	}

	sd, err = client.GetServiceDiscoveryService(context.TODO(), aws.String("srv-badsd"))
	if err == nil {
		t.Fatalf("expected error from bad get service discovery service, got %+v", sd)
	}
	t.Log("Got expected error from bad service discovery service", err)
}
