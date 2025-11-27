package provider

import (
	"context"
	"fmt"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/route53"
	"github.com/aws/aws-sdk-go-v2/service/route53/types"
)

// mockRoute53API satisfies the Route53API interface and allows controlling the responses.
type mockRoute53API struct {
	ListHostedZonesFunc          func(ctx context.Context, params *route53.ListHostedZonesInput, optFns ...func(*route53.Options)) (*route53.ListHostedZonesOutput, error)
	ListResourceRecordSetsFunc   func(ctx context.Context, params *route53.ListResourceRecordSetsInput, optFns ...func(*route53.Options)) (*route53.ListResourceRecordSetsOutput, error)
	ChangeResourceRecordSetsFunc func(ctx context.Context, params *route53.ChangeResourceRecordSetsInput, optFns ...func(*route53.Options)) (*route53.ChangeResourceRecordSetsOutput, error)
}

func (m *mockRoute53API) ListHostedZones(ctx context.Context, params *route53.ListHostedZonesInput, optFns ...func(*route53.Options)) (*route53.ListHostedZonesOutput, error) {
	if m.ListHostedZonesFunc != nil {
		return m.ListHostedZonesFunc(ctx, params, optFns...)
	}
	return nil, fmt.Errorf("ListHostedZonesFunc is not implemented")
}

func (m *mockRoute53API) ListResourceRecordSets(ctx context.Context, params *route53.ListResourceRecordSetsInput, optFns ...func(*route53.Options)) (*route53.ListResourceRecordSetsOutput, error) {
	if m.ListResourceRecordSetsFunc != nil {
		return m.ListResourceRecordSetsFunc(ctx, params, optFns...)
	}
	return nil, fmt.Errorf("ListResourceRecordSetsFunc is not implemented")
}

func (m *mockRoute53API) ChangeResourceRecordSets(ctx context.Context, params *route53.ChangeResourceRecordSetsInput, optFns ...func(*route53.Options)) (*route53.ChangeResourceRecordSetsOutput, error) {
	if m.ChangeResourceRecordSetsFunc != nil {
		return m.ChangeResourceRecordSetsFunc(ctx, params, optFns...)
	}
	return nil, fmt.Errorf("ChangeResourceRecordSetsFunc is not implemented")
}

func TestAwsRoute53Client_GetRecord(t *testing.T) {
	domain := "example.com"
	hostname := "test.example.com"
	recordType := "A"
	expectedValue := "192.0.2.1"

	mockAPI := &mockRoute53API{
		ListHostedZonesFunc: func(ctx context.Context, params *route53.ListHostedZonesInput, optFns ...func(*route53.Options)) (*route53.ListHostedZonesOutput, error) {
			return &route53.ListHostedZonesOutput{
				HostedZones: []types.HostedZone{{
					Id:   aws.String("/hostedzone/ZONE123"),
					Name: aws.String("example.com."),
				}},
				IsTruncated: false,
			}, nil
		},
		ListResourceRecordSetsFunc: func(ctx context.Context, params *route53.ListResourceRecordSetsInput, optFns ...func(*route53.Options)) (*route53.ListResourceRecordSetsOutput, error) {
			zoneID := aws.ToString(params.HostedZoneId)
			if zoneID != "ZONE123" {
				t.Fatalf("expected hosted zone %s, got %s", "ZONE123", zoneID)
			}
			if aws.ToString(params.StartRecordName) != "test.example.com." {
				t.Fatalf("unexpected start record name: %s", aws.ToString(params.StartRecordName))
			}
			if params.StartRecordType != types.RRTypeA {
				t.Fatalf("unexpected record type: %s", params.StartRecordType)
			}

			return &route53.ListResourceRecordSetsOutput{
				ResourceRecordSets: []types.ResourceRecordSet{{
					Name:            aws.String("test.example.com."),
					Type:            types.RRTypeA,
					TTL:             aws.Int64(300),
					ResourceRecords: []types.ResourceRecord{{Value: aws.String(expectedValue)}},
				}},
			}, nil
		},
	}

	client := NewAwsRoute53ClientWithMock(mockAPI)
	record, err := client.GetRecord(context.Background(), domain, hostname, recordType)

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if record == nil {
		t.Fatalf("expected record")
	}
	if record.Name != hostname {
		t.Fatalf("unexpected name: %s", record.Name)
	}
	if record.Type != recordType {
		t.Fatalf("unexpected type: %s", record.Type)
	}
	if record.Value != expectedValue {
		t.Fatalf("unexpected value: %s", record.Value)
	}
	if record.TTL != 300 {
		t.Fatalf("unexpected ttl: %d", record.TTL)
	}
}

func TestAwsRoute53Client_UpdateRecord(t *testing.T) {
	domain := "example.com"
	record := &DNSRecord{
		Name:  "test.example.com",
		Type:  "A",
		Value: "192.0.2.2",
		TTL:   250,
	}

	mockAPI := &mockRoute53API{}
	var changeCalls int

	mockAPI.ListHostedZonesFunc = func(ctx context.Context, params *route53.ListHostedZonesInput, optFns ...func(*route53.Options)) (*route53.ListHostedZonesOutput, error) {
		return &route53.ListHostedZonesOutput{HostedZones: []types.HostedZone{{
			Id:   aws.String("/hostedzone/ZONE123"),
			Name: aws.String("example.com."),
		}}}, nil
	}

	mockAPI.ListResourceRecordSetsFunc = func(ctx context.Context, params *route53.ListResourceRecordSetsInput, optFns ...func(*route53.Options)) (*route53.ListResourceRecordSetsOutput, error) {
		return &route53.ListResourceRecordSetsOutput{ResourceRecordSets: []types.ResourceRecordSet{}}, nil
	}

	mockAPI.ChangeResourceRecordSetsFunc = func(ctx context.Context, params *route53.ChangeResourceRecordSetsInput, optFns ...func(*route53.Options)) (*route53.ChangeResourceRecordSetsOutput, error) {
		changeCalls++
		if aws.ToString(params.HostedZoneId) != "ZONE123" {
			t.Fatalf("unexpected hosted zone: %s", aws.ToString(params.HostedZoneId))
		}
		if len(params.ChangeBatch.Changes) != 1 {
			t.Fatalf("expected a single change, got %d", len(params.ChangeBatch.Changes))
		}
		change := params.ChangeBatch.Changes[0]
		if change.Action != types.ChangeActionUpsert {
			t.Fatalf("unexpected action: %s", change.Action)
		}
		if aws.ToString(change.ResourceRecordSet.Name) != "test.example.com." {
			t.Fatalf("unexpected record name: %s", aws.ToString(change.ResourceRecordSet.Name))
		}
		if change.ResourceRecordSet.Type != types.RRTypeA {
			t.Fatalf("unexpected record type: %s", change.ResourceRecordSet.Type)
		}
		if aws.ToInt64(change.ResourceRecordSet.TTL) != int64(record.TTL) {
			t.Fatalf("unexpected ttl %d", aws.ToInt64(change.ResourceRecordSet.TTL))
		}
		if aws.ToString(change.ResourceRecordSet.ResourceRecords[0].Value) != record.Value {
			t.Fatalf("unexpected value: %s", aws.ToString(change.ResourceRecordSet.ResourceRecords[0].Value))
		}
		return &route53.ChangeResourceRecordSetsOutput{}, nil
	}

	client := NewAwsRoute53ClientWithMock(mockAPI)
	err := client.UpdateRecord(context.Background(), domain, record)

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if changeCalls != 1 {
		t.Fatalf("expected 1 change call, got %d", changeCalls)
	}
}
