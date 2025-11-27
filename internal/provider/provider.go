package provider

import (
	"context"
	"fmt"
	"sort"
)

// DNSRecordSet represents a DNS zone's record set
type DNSRecordSet struct {
	DNSRecords []DNSRecord `json:"dnsrecords"`
}

// DNSRecord represents a DNS record.
type DNSRecord struct {
	Name     string
	Type     string
	Value    string
	TTL      int
	Priority int
}

// Provider defines the interface that all DNS providers must implement.
type Provider interface {
	Name() string
	GetRecord(ctx context.Context, domain, hostname, recordType string) (*DNSRecord, error)
	UpdateRecord(ctx context.Context, domain string, record *DNSRecord) error
	Close(ctx context.Context) error
}

var (
	// factories holds the registered provider factories.
	factories = make(map[string]func(ctx context.Context, config interface{}) (Provider, error))
)

// RegisterFactory registers a provider factory function.
// This is called by provider packages in their init() function.
func RegisterFactory(name string, factory func(ctx context.Context, config interface{}) (Provider, error)) {
	if _, exists := factories[name]; exists {
		panic(fmt.Sprintf("provider factory already registered: %s", name))
	}
	factories[name] = factory
}

// GetFactory retrieves a provider factory by name.
func GetFactory(name string) (func(ctx context.Context, config interface{}) (Provider, error), bool) {
	factory, ok := factories[name]
	return factory, ok
}

// List returns the names of all registered providers, sorted alphabetically.
func List() []string {
	names := make([]string, 0, len(factories))
	for name := range factories {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}
