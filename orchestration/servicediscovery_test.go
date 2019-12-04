package orchestration

import (
	"context"
	"reflect"
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/servicediscovery"
)

func TestCreateServiceDiscovery(t *testing.T) {
	client := &mockSDClient{}
	sd, err := createServiceDiscoveryService(context.TODO(), client, &servicediscovery.CreateServiceInput{
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

	sd, err = createServiceDiscoveryService(context.TODO(), client, &servicediscovery.CreateServiceInput{
		Name: aws.String("badsd"),
	})
	if err == nil {
		t.Fatalf("expected error from bad create service discovery service, got %+v", sd)
	}
	t.Log("Got expected error from bad service discovery create service", err)
}

func TestGetServiceDiscovery(t *testing.T) {
	client := &mockSDClient{}
	sd, err := getServiceDiscoveryService(context.TODO(), client, aws.String("srv-goodsd"))
	if err != nil {
		t.Fatal("expected no error from get service discovery service, got", err)
	}
	t.Log("Got service discovery get service output", sd)
	if !reflect.DeepEqual(sd, goodSd) {
		t.Fatalf("expected: %+v\n Got: %+v", goodSd, sd)
	}

	sd, err = getServiceDiscoveryService(context.TODO(), client, aws.String("srv-badsd"))
	if err == nil {
		t.Fatalf("expected error from bad get service discovery service, got %+v", sd)
	}
	t.Log("Got expected error from bad service discovery service", err)
}
