package cmd

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

// Config represents the application configuration
type Config struct {
	Port         int
	Username     string
	PasswordHash string
	Provider     string
	Domain       string
	DefaultTTL   int
}

func LoadConfig() (*Config, error) {
	config := &Config{
		Port:       8080,
		DefaultTTL: 60,
		Provider:   "netcup_ccp", // default provider
	}

	// Port
	if port := os.Getenv("PORT"); port != "" {
		p, err := strconv.Atoi(port)
		if err != nil {
			return nil, fmt.Errorf("invalid PORT: %w", err)
		}
		config.Port = p
	}

	// Auth credentials
	config.Username = os.Getenv("AUTH_USERNAME")
	if config.Username == "" {
		return nil, fmt.Errorf("AUTH_USERNAME is required")
	}

	config.PasswordHash = os.Getenv("AUTH_PASSWORD_HASH")
	if config.PasswordHash == "" {
		return nil, fmt.Errorf("AUTH_PASSWORD_HASH is required")
	}

	// Provider selection
	if provider := os.Getenv("DNS_PROVIDER"); provider != "" {
		config.Provider = strings.ToLower(provider)
	}

	// Domain
	config.Domain = os.Getenv("DOMAIN")
	if config.Domain == "" {
		return nil, fmt.Errorf("DOMAIN is required")
	}

	// TTL
	if ttl := os.Getenv("DNS_TTL"); ttl != "" {
		t, err := strconv.Atoi(ttl)
		if err != nil {
			return nil, fmt.Errorf("invalid DNS_TTL: %w", err)
		}
		config.DefaultTTL = t
	}

	return config, nil
}
