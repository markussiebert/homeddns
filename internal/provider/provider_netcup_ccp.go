//go:build netcup_ccp || (!netcup_ccp && !aws_route53)
// +build netcup_ccp !netcup_ccp,!aws_route53

package provider

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

const (
	DefaultEndpoint    = "https://ccp.netcup.net/run/webservice/servers/endpoint.php?JSON"
	SessionTimeout     = 15 * time.Minute
	SessionRefreshTime = 10 * time.Minute // Refresh before timeout
	netcupCredDir      = ".homeddns"
	netcupCredFile     = "netcup_credentials"
)

// LoadNetcupConfig loads Netcup credentials from environment variables or credential file
func LoadNetcupConfig() (*NetcupConfig, error) {
	config := &NetcupConfig{}

	// First, try environment variables
	config.CustomerNumber = os.Getenv("NETCUP_CUSTOMER_NUMBER")
	config.ApiKey = os.Getenv("NETCUP_API_KEY")
	config.ApiPassword = os.Getenv("NETCUP_API_PASSWORD")

	if config.CustomerNumber != "" && config.ApiKey != "" && config.ApiPassword != "" {
		return config, nil
	}

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get home dir: %w", err)
	}

	credFile := filepath.Join(homeDir, netcupCredDir, netcupCredFile)
	file, err := os.Open(credFile)
	if os.IsNotExist(err) {
		return nil, fmt.Errorf("netcup credentials not found. Please set NETCUP_* environment variables or create %s", credFile)
	} else if err != nil {
		return nil, fmt.Errorf("failed to open credentials file %s: %w", credFile, err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}
		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])

		switch key {
		case "customer_number":
			if config.CustomerNumber == "" {
				config.CustomerNumber = value
			}
		case "api_key":
			if config.ApiKey == "" {
				config.ApiKey = value
			}
		case "api_password":
			if config.ApiPassword == "" {
				config.ApiPassword = value
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("failed to read credentials file: %w", err)
	}

	if config.CustomerNumber == "" || config.ApiKey == "" || config.ApiPassword == "" {
		return nil, fmt.Errorf("incomplete netcup credentials in env vars and file")
	}

	return config, nil
}

// netcupDNSRecord represents a DNS record in the netcup API.
type netcupDNSRecord struct {
	ID          string `json:"id,omitempty"`
	Hostname    string `json:"hostname"`
	Type        string `json:"type"`
	Priority    string `json:"priority,omitempty"`
	Destination string `json:"destination"`
	Delete      bool   `json:"deleterecord,omitempty"`
	State       string `json:"state,omitempty"`
}

// netcupDNSRecordSet represents a set of DNS records for the netcup API.
type netcupDNSRecordSet struct {
	DNSRecords []netcupDNSRecord `json:"dnsrecords"`
}

// NetcupClient represents a netcup CCP API client
type NetcupClient struct {
	endpoint         string
	customerNumber   string
	apiKey           string
	apiPassword      string
	httpNetcupClient *http.Client

	sessionID     string
	sessionExpiry time.Time
	sessionMu     sync.RWMutex
}

// APIRequest represents a generic API request
type APIRequest struct {
	Action string      `json:"action"`
	Param  interface{} `json:"param"`
}

// APIResponse represents a generic API response
type APIResponse struct {
	ServerRequestID       string          `json:"serverrequestid"`
	NetcupClientRequestID string          `json:"clientrequestid,omitempty"`
	Action                string          `json:"action"`
	Status                string          `json:"status"`
	StatusCode            int             `json:"statuscode"`
	ShortMessage          string          `json:"shortmessage"`
	LongMessage           string          `json:"longmessage"`
	ResponseData          json.RawMessage `json:"responsedata,omitempty"`
}

// LoginParams represents login parameters
type LoginParams struct {
	CustomerNumber string `json:"customernumber"`
	APIKey         string `json:"apikey"`
	APIPassword    string `json:"apipassword"`
}

// SessionParams represents session parameters
type SessionParams struct {
	CustomerNumber string `json:"customernumber"`
	APIKey         string `json:"apikey"`
	APISessionID   string `json:"apisessionid"`
}

// LoginResponse represents the login response data
type LoginResponse struct {
	APISessionID string `json:"apisessionid"`
}

// InfoDNSRecordsParams represents parameters for infoDnsRecords
type InfoDNSRecordsParams struct {
	DomainName string `json:"domainname"`
	SessionParams
}

// UpdateDNSRecordsParams represents parameters for updateDnsRecords
type UpdateDNSRecordsParams struct {
	DomainName   string             `json:"domainname"`
	DNSRecordSet netcupDNSRecordSet `json:"dnsrecordset"`
	SessionParams
}

// NewNetcupClient creates a new netcup CCP API client
func NewNetcupClient(customerNumber, apiKey, apiPassword string) *NetcupClient {
	return &NetcupClient{
		endpoint:       DefaultEndpoint,
		customerNumber: customerNumber,
		apiKey:         apiKey,
		apiPassword:    apiPassword,
		httpNetcupClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// WithEndpoint sets a custom endpoint
func (c *NetcupClient) WithEndpoint(endpoint string) *NetcupClient {
	c.endpoint = endpoint
	return c
}

// doRequest performs an API request
func (c *NetcupClient) doRequest(ctx context.Context, req *APIRequest) (*APIResponse, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", c.endpoint, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.httpNetcupClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("do request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d, body: %s", resp.StatusCode, respBody)
	}

	var apiResp APIResponse
	if err := json.Unmarshal(respBody, &apiResp); err != nil {
		return nil, fmt.Errorf("unmarshal response: %w", err)
	}

	if apiResp.Status != "success" {
		return nil, fmt.Errorf("API error: %s - %s (code: %d)", apiResp.ShortMessage, apiResp.LongMessage, apiResp.StatusCode)
	}

	return &apiResp, nil
}

// login performs a login and stores the session ID
func (c *NetcupClient) login(ctx context.Context) error {
	req := &APIRequest{
		Action: "login",
		Param: LoginParams{
			CustomerNumber: c.customerNumber,
			APIKey:         c.apiKey,
			APIPassword:    c.apiPassword,
		},
	}

	resp, err := c.doRequest(ctx, req)
	if err != nil {
		return fmt.Errorf("login request: %w", err)
	}

	var loginResp LoginResponse
	if err := json.Unmarshal(resp.ResponseData, &loginResp); err != nil {
		return fmt.Errorf("unmarshal login response: %w", err)
	}

	c.sessionMu.Lock()
	c.sessionID = loginResp.APISessionID
	c.sessionExpiry = time.Now().Add(SessionRefreshTime)
	c.sessionMu.Unlock()

	return nil
}

// ensureSession ensures we have a valid session
func (c *NetcupClient) ensureSession(ctx context.Context) error {
	c.sessionMu.RLock()
	needsRefresh := c.sessionID == "" || time.Now().After(c.sessionExpiry)
	c.sessionMu.RUnlock()

	if needsRefresh {
		return c.login(ctx)
	}

	return nil
}

// getSessionParams returns session parameters
func (c *NetcupClient) getSessionParams() SessionParams {
	c.sessionMu.RLock()
	defer c.sessionMu.RUnlock()

	return SessionParams{
		CustomerNumber: c.customerNumber,
		APIKey:         c.apiKey,
		APISessionID:   c.sessionID,
	}
}

// InfoDNSRecords retrieves all DNS records for a domain
func (c *NetcupClient) InfoDNSRecords(ctx context.Context, domainName string) ([]netcupDNSRecord, error) {
	if err := c.ensureSession(ctx); err != nil {
		return nil, fmt.Errorf("ensure session: %w", err)
	}

	req := &APIRequest{
		Action: "infoDnsRecords",
		Param: InfoDNSRecordsParams{
			DomainName:    domainName,
			SessionParams: c.getSessionParams(),
		},
	}

	resp, err := c.doRequest(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("infoDnsRecords request: %w", err)
	}

	var recordSet netcupDNSRecordSet
	if err := json.Unmarshal(resp.ResponseData, &recordSet); err != nil {
		return nil, fmt.Errorf("unmarshal DNS records: %w", err)
	}

	return recordSet.DNSRecords, nil
}

// UpdateDNSRecords updates DNS records for a domain
func (c *NetcupClient) UpdateDNSRecords(ctx context.Context, domainName string, records []netcupDNSRecord) error {
	if err := c.ensureSession(ctx); err != nil {
		return fmt.Errorf("ensure session: %w", err)
	}

	req := &APIRequest{
		Action: "updateDnsRecords",
		Param: UpdateDNSRecordsParams{
			DomainName: domainName,
			DNSRecordSet: netcupDNSRecordSet{
				DNSRecords: records,
			},
			SessionParams: c.getSessionParams(),
		},
	}

	_, err := c.doRequest(ctx, req)
	if err != nil {
		return fmt.Errorf("updateDnsRecords request: %w", err)
	}

	return nil
}

// Logout logs out and invalidates the session
func (c *NetcupClient) Logout(ctx context.Context) error {
	c.sessionMu.RLock()
	if c.sessionID == "" {
		c.sessionMu.RUnlock()
		return nil
	}
	c.sessionMu.RUnlock()

	req := &APIRequest{
		Action: "logout",
		Param:  c.getSessionParams(),
	}

	_, err := c.doRequest(ctx, req)
	if err != nil {
		return fmt.Errorf("logout request: %w", err)
	}

	c.sessionMu.Lock()
	c.sessionID = ""
	c.sessionExpiry = time.Time{}
	c.sessionMu.Unlock()

	return nil
}

func init() {
	RegisterFactory("netcup_ccp", NewNetcupProvider)
}

// NewNetcupProvider creates a new Netcup provider
func NewNetcupProvider(ctx context.Context, config interface{}) (Provider, error) {
	// Load Netcup credentials from environment or file
	cfg, err := LoadNetcupConfig()
	if err != nil {
		return nil, fmt.Errorf("load netcup config: %w", err)
	}
	return NewNetcupClient(cfg.CustomerNumber, cfg.ApiKey, cfg.ApiPassword), nil
}

// Provider interface implementation

// Name returns the provider name
func (c *NetcupClient) Name() string {
	return "netcup_ccp"
}

// GetRecord retrieves a specific DNS record
func (c *NetcupClient) GetRecord(ctx context.Context, domain, hostname, recordType string) (*DNSRecord, error) {
	// Extract subdomain from hostname
	subdomain := c.extractSubdomain(hostname, domain)

	// Get all DNS records for the domain
	records, err := c.InfoDNSRecords(ctx, domain)
	if err != nil {
		return nil, fmt.Errorf("get DNS records: %w", err)
	}

	// Find matching record
	for _, record := range records {
		if record.Hostname == subdomain && record.Type == recordType {
			return &DNSRecord{
				Name:  hostname,
				Type:  record.Type,
				Value: record.Destination,
				TTL:   60, // Netcup doesn't expose TTL via API, default to 60
			}, nil
		}
	}

	return nil, fmt.Errorf("record not found")
}

// UpdateRecord updates or creates a DNS record
func (c *NetcupClient) UpdateRecord(ctx context.Context, domain string, record *DNSRecord) error {
	// Extract subdomain from hostname
	subdomain := c.extractSubdomain(record.Name, domain)

	// Get all existing records
	records, err := c.InfoDNSRecords(ctx, domain)
	if err != nil {
		return fmt.Errorf("get DNS records: %w", err)
	}

	// Find existing record
	var existingRecord *netcupDNSRecord
	var otherRecords []netcupDNSRecord
	for _, r := range records {
		if r.Hostname == subdomain && r.Type == record.Type {
			existingRecord = &r
		} else {
			otherRecords = append(otherRecords, r)
		}
	}

	// Check if update is needed
	if existingRecord != nil && existingRecord.Destination == record.Value {
		// Already up to date
		return nil
	}

	// Prepare record for update
	recordToUpdate := netcupDNSRecord{
		Hostname:    subdomain,
		Type:        record.Type,
		Destination: record.Value,
	}
	if existingRecord != nil {
		recordToUpdate.ID = existingRecord.ID
	}

	// Create the new list of records
	updatedRecords := append(otherRecords, recordToUpdate)

	// Update the record
	if err := c.UpdateDNSRecords(ctx, domain, updatedRecords); err != nil {
		return fmt.Errorf("update DNS record: %w", err)
	}

	return nil
}

// Close logs out and cleans up resources
func (c *NetcupClient) Close(ctx context.Context) error {
	return c.Logout(ctx)
}

// extractSubdomain extracts the subdomain from a full hostname
// Example: "test.example.com" with domain "example.com" returns "test"
// Example: "*.example.com" with domain "example.com" returns "*"
// Example: "example.com" with domain "example.com" returns "@"
func (c *NetcupClient) extractSubdomain(hostname, domain string) string {
	hostname = strings.ToLower(hostname)
	domain = strings.ToLower(domain)

	if hostname == domain {
		return "@"
	}

	suffix := "." + domain
	if strings.HasSuffix(hostname, suffix) {
		return strings.TrimSuffix(hostname, suffix)
	}

	return hostname
}
