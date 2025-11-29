package cmd

import (
	"encoding/json"
	"fmt"
	"os"
)

// HomeAssistantOptions represents the Home Assistant add-on options.json structure
type HomeAssistantOptions struct {
	AuthUsername       string `json:"auth_username"`
	AuthPasswordHash   string `json:"auth_password_hash"`
	DNSProvider        string `json:"dns_provider"`
	Domain             string `json:"domain"`
	DNSTTL             int    `json:"dns_ttl"`
	Port               int    `json:"port"`
	NetcupCustomerNum  string `json:"netcup_customer_number"`
	NetcupAPIKey       string `json:"netcup_api_key"`
	NetcupAPIPassword  string `json:"netcup_api_password"`
	AWSAccessKeyID     string `json:"aws_access_key_id"`
	AWSSecretAccessKey string `json:"aws_secret_access_key"`
	AWSRegion          string `json:"aws_region"`
}

// LoadHomeAssistantConfig loads configuration from the add-on options file when running under Supervisor
func LoadHomeAssistantConfig() (*Config, error) {
	optionsPath := os.Getenv("ADDON_OPTIONS_PATH")
	if optionsPath == "" {
		optionsPath = "/data/options.json"
	}

	info, err := os.Stat(optionsPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil // Not running as Home Assistant add-on
		}
		return nil, fmt.Errorf("failed to stat options file %s: %w", optionsPath, err)
	}
	if info.IsDir() {
		return nil, fmt.Errorf("options path %s is a directory", optionsPath)
	}

	data, err := os.ReadFile(optionsPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read options.json: %w", err)
	}

	var opts HomeAssistantOptions
	if err := json.Unmarshal(data, &opts); err != nil {
		return nil, fmt.Errorf("failed to parse options.json: %w", err)
	}

	// Map Home Assistant options to environment variables
	// This allows the rest of the code to work unchanged
	if opts.AuthUsername != "" {
		os.Setenv("AUTH_USERNAME", opts.AuthUsername)
	}
	if opts.AuthPasswordHash != "" {
		os.Setenv("AUTH_PASSWORD_HASH", opts.AuthPasswordHash)
	}
	if opts.DNSProvider != "" {
		os.Setenv("DNS_PROVIDER", opts.DNSProvider)
	}
	if opts.Domain != "" {
		os.Setenv("DOMAIN", opts.Domain)
	}
	if opts.DNSTTL > 0 {
		os.Setenv("DNS_TTL", fmt.Sprintf("%d", opts.DNSTTL))
	}
	if opts.Port > 0 {
		os.Setenv("PORT", fmt.Sprintf("%d", opts.Port))
	}

	// Provider-specific credentials
	if opts.NetcupCustomerNum != "" {
		os.Setenv("NETCUP_CUSTOMER_NUMBER", opts.NetcupCustomerNum)
	}
	if opts.NetcupAPIKey != "" {
		os.Setenv("NETCUP_API_KEY", opts.NetcupAPIKey)
	}
	if opts.NetcupAPIPassword != "" {
		os.Setenv("NETCUP_API_PASSWORD", opts.NetcupAPIPassword)
	}
	if opts.AWSAccessKeyID != "" {
		os.Setenv("AWS_ACCESS_KEY_ID", opts.AWSAccessKeyID)
	}
	if opts.AWSSecretAccessKey != "" {
		os.Setenv("AWS_SECRET_ACCESS_KEY", opts.AWSSecretAccessKey)
	}
	if opts.AWSRegion != "" {
		os.Setenv("AWS_REGION", opts.AWSRegion)
	}

	// Now load config normally from environment variables
	return LoadConfig()
}
