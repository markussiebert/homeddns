package cmd

import (
	"context"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"

	"github.com/markussiebert/homeddns/internal/provider"
)

func RunUpdate(hostname, recordType string, config *Config) error {
	publicIP, err := getPublicIP()
	if err != nil {
		return fmt.Errorf("failed to get public IP: %w", err)
	}

	log.Printf("Current public IP: %s", publicIP)

	prov, ok := provider.GetFactory(config.Provider)
	if !ok {
		return fmt.Errorf("provider factory not found: %s", config.Provider)
	}

	// Provider handles its own credential loading
	p, err := prov(context.Background(), nil)
	if err != nil {
		return fmt.Errorf("failed to create provider: %w", err)
	}

	record := &provider.DNSRecord{
		Name:  hostname,
		Type:  recordType,
		Value: publicIP,
		TTL:   config.DefaultTTL,
	}

	if err := p.UpdateRecord(context.Background(), config.Domain, record); err != nil {
		return fmt.Errorf("failed to update DNS record: %w", err)
	}

	log.Printf("Successfully updated %s record for %s to %s", recordType, hostname, publicIP)
	return nil
}

func getPublicIP() (string, error) {
	resp, err := http.Get("https://api.ipify.org")
	if err != nil {
		return "", fmt.Errorf("HTTP request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response body: %w", err)
	}

	ip := string(body)
	if net.ParseIP(ip) == nil {
		return "", fmt.Errorf("invalid IP address received: %s", ip)
	}

	return ip, nil
}
