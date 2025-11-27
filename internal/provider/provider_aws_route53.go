//go:build aws_route53 || (!netcup && !aws_route53)
// +build aws_route53 !netcup,!aws_route53

package provider

import (
	"context"
	"fmt"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/route53"
	"github.com/aws/aws-sdk-go-v2/service/route53/types"
)

// Route53API defines the interface for the AWS Route53 API, enabling mocking.
type Route53API interface {
	ListHostedZones(ctx context.Context, params *route53.ListHostedZonesInput, optFns ...func(*route53.Options)) (*route53.ListHostedZonesOutput, error)
	ListResourceRecordSets(ctx context.Context, params *route53.ListResourceRecordSetsInput, optFns ...func(*route53.Options)) (*route53.ListResourceRecordSetsOutput, error)
	ChangeResourceRecordSets(ctx context.Context, params *route53.ChangeResourceRecordSetsInput, optFns ...func(*route53.Options)) (*route53.ChangeResourceRecordSetsOutput, error)
}

// AwsRoute53Client represents an AWS Route53 client
type AwsRoute53Client struct {
	client    Route53API
	zoneCache map[string]string // domain -> hostedZoneId cache
}

// NewAwsRoute53Client creates a new Route53 client
func NewAwsRoute53Client(ctx context.Context) (*AwsRoute53Client, error) {
	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		return nil, fmt.Errorf("load AWS config: %w", err)
	}

	return &AwsRoute53Client{
		client:    route53.NewFromConfig(cfg),
		zoneCache: make(map[string]string),
	}, nil
}

// NewAwsRoute53ClientWithConfig creates a new Route53 client with custom AWS config
func NewAwsRoute53ClientWithConfig(cfg aws.Config) *AwsRoute53Client {
	return &AwsRoute53Client{
		client:    route53.NewFromConfig(cfg),
		zoneCache: make(map[string]string),
	}
}

// NewAwsRoute53ClientWithMock creates a new Route53 client with a mock API for testing
func NewAwsRoute53ClientWithMock(mock Route53API) *AwsRoute53Client {
	return &AwsRoute53Client{
		client:    mock,
		zoneCache: make(map[string]string),
	}
}

// Name returns the provider name
func (c *AwsRoute53Client) Name() string {
	return "aws_route53"
}

// GetRecord retrieves a specific DNS record
func (c *AwsRoute53Client) GetRecord(ctx context.Context, domain, hostname, recordType string) (*DNSRecord, error) {
	// Get hosted zone ID
	zoneID, err := c.getHostedZoneID(ctx, domain)
	if err != nil {
		return nil, fmt.Errorf("get hosted zone: %w", err)
	}

	// Ensure hostname ends with a dot for Route53
	fqdn := c.ensureTrailingDot(hostname)

	// List resource record sets
	input := &route53.ListResourceRecordSetsInput{
		HostedZoneId:    aws.String(zoneID),
		StartRecordName: aws.String(fqdn),
		StartRecordType: types.RRType(recordType),
		MaxItems:        aws.Int32(1),
	}

	result, err := c.client.ListResourceRecordSets(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("list record sets: %w", err)
	}

	// Check if we found the record
	if len(result.ResourceRecordSets) == 0 {
		return nil, fmt.Errorf("record not found")
	}

	recordSet := result.ResourceRecordSets[0]
	if aws.ToString(recordSet.Name) != fqdn || string(recordSet.Type) != recordType {
		return nil, fmt.Errorf("record not found")
	}

	// Extract value
	var value string
	if len(recordSet.ResourceRecords) > 0 {
		value = aws.ToString(recordSet.ResourceRecords[0].Value)
	}

	return &DNSRecord{
		Name:  hostname,
		Type:  recordType,
		Value: value,
		TTL:   int(aws.ToInt64(recordSet.TTL)),
	}, nil
}

// UpdateRecord updates or creates a DNS record
func (c *AwsRoute53Client) UpdateRecord(ctx context.Context, domain string, record *DNSRecord) error {
	// Get hosted zone ID
	zoneID, err := c.getHostedZoneID(ctx, domain)
	if err != nil {
		return fmt.Errorf("get hosted zone: %w", err)
	}

	// Ensure hostname ends with a dot for Route53
	fqdn := c.ensureTrailingDot(record.Name)

	// Check if record exists and if it needs updating
	existing, err := c.GetRecord(ctx, domain, record.Name, record.Type)
	if err == nil && existing.Value == record.Value {
		// Record exists and is already up to date
		return nil
	}

	// Prepare the resource record
	resourceRecord := types.ResourceRecord{
		Value: aws.String(record.Value),
	}

	// Prepare the change batch
	change := types.Change{
		Action: types.ChangeActionUpsert,
		ResourceRecordSet: &types.ResourceRecordSet{
			Name: aws.String(fqdn),
			Type: types.RRType(record.Type),
			TTL:  aws.Int64(int64(record.TTL)),
			ResourceRecords: []types.ResourceRecord{
				resourceRecord,
			},
		},
	}

	// Apply the change
	input := &route53.ChangeResourceRecordSetsInput{
		HostedZoneId: aws.String(zoneID),
		ChangeBatch: &types.ChangeBatch{
			Changes: []types.Change{change},
		},
	}

	_, err = c.client.ChangeResourceRecordSets(ctx, input)
	if err != nil {
		return fmt.Errorf("change resource record sets: %w", err)
	}

	return nil
}

// Close cleans up resources (no-op for Route53)
func (c *AwsRoute53Client) Close(ctx context.Context) error {
	return nil
}

// getHostedZoneID retrieves the hosted zone ID for a domain
func (c *AwsRoute53Client) getHostedZoneID(ctx context.Context, domain string) (string, error) {
	// Check cache first
	if zoneID, exists := c.zoneCache[domain]; exists {
		return zoneID, nil
	}

	// Ensure domain ends with a dot
	domainWithDot := c.ensureTrailingDot(domain)

	// List hosted zones manually for mockability
	var marker *string
	for {
		input := &route53.ListHostedZonesInput{
			Marker: marker,
		}

		output, err := c.client.ListHostedZones(ctx, input)
		if err != nil {
			return "", fmt.Errorf("list hosted zones: %w", err)
		}

		for _, zone := range output.HostedZones {
			if aws.ToString(zone.Name) == domainWithDot {
				zoneID := aws.ToString(zone.Id)
				// Strip /hostedzone/ prefix if present
				zoneID = strings.TrimPrefix(zoneID, "/hostedzone/")
				c.zoneCache[domain] = zoneID
				return zoneID, nil
			}
		}

		if !output.IsTruncated {
			break
		}
		marker = output.NextMarker
	}

	return "", fmt.Errorf("hosted zone for domain %s not found", domain)
}

// ensureTrailingDot ensures the hostname ends with a dot
func (c *AwsRoute53Client) ensureTrailingDot(hostname string) string {
	if !strings.HasSuffix(hostname, ".") {
		return hostname + "."
	}
	return hostname
}

func init() {
	RegisterFactory("aws_route53", NewRoute53Provider)
}

// NewRoute53Provider creates a new AWS Route53 provider
// It uses the default AWS SDK configuration chain (env vars, ~/.aws/credentials, IAM roles, etc.)
func NewRoute53Provider(ctx context.Context, config interface{}) (Provider, error) {
	return NewAwsRoute53Client(ctx)
}
