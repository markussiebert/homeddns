package handler

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/markussiebert/homeddns/internal/logger"
	"github.com/markussiebert/homeddns/internal/provider"
)

// Config represents the DynDNS handler configuration
type Config struct {
	Provider   provider.Provider
	DefaultTTL int
}

// DynDNSHandler handles DynDNS update requests
type DynDNSHandler struct {
	config Config
}

// NewDynDNSHandler creates a new DynDNS handler
func NewDynDNSHandler(config Config) *DynDNSHandler {
	if config.DefaultTTL == 0 {
		config.DefaultTTL = 60
	}
	return &DynDNSHandler{config: config}
}

// ServeHTTP handles HTTP requests
func (h *DynDNSHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	logger.Debug("Received %s request: %s from %s", r.Method, r.URL.Path, r.RemoteAddr)
	logger.Debug("Query params: %s", r.URL.RawQuery)

	// Only allow GET requests
	if r.Method != http.MethodGet {
		logger.Warn("Method not allowed: %s from %s", r.Method, r.RemoteAddr)
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Determine format based on path
	isStandardFormat := strings.HasPrefix(r.URL.Path, "/nic/update")
	logger.Debug("Using %s format", map[bool]string{true: "standard", false: "UniFi"}[isStandardFormat])

	// Extract hostname
	hostname := h.extractHostname(r)
	if hostname == "" {
		logger.Warn("No valid hostname found in request from %s", r.RemoteAddr)
		h.respond(w, "notfqdn", "", isStandardFormat)
		return
	}

	logger.Debug("Extracted hostname: %s", hostname)

	// Validate and parse hostname
	hostname = h.normalizeHostname(hostname)
	logger.Debug("Normalized hostname: %s", hostname)

	domain, subdomain := h.splitHostname(hostname)
	if domain == "" {
		logger.Error("Failed to split hostname '%s' into domain and subdomain", hostname)
		h.respond(w, "notfqdn", "", isStandardFormat)
		return
	}
	logger.Debug("Split hostname: domain=%s, subdomain=%s", domain, subdomain)

	// Get IP address
	ipAddress := h.extractIP(r)
	if ipAddress == "" {
		logger.Error("Failed to extract valid IP address from request")
		h.respond(w, "911", "", isStandardFormat)
		return
	}
	logger.Debug("Extracted IP address: %s", ipAddress)

	// Update DNS record
	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()

	logger.Debug("Updating DNS: hostname=%s, domain=%s, subdomain=%s, ip=%s", hostname, domain, subdomain, ipAddress)

	if err := h.updateDNS(ctx, domain, subdomain, ipAddress); err != nil {
		logger.Error("Error updating DNS for %s: %v", hostname, err)
		h.respond(w, "911", ipAddress, isStandardFormat)
		return
	}

	logger.Info("Successfully updated %s to %s", hostname, ipAddress)
	h.respond(w, "good", ipAddress, isStandardFormat)
}

// extractHostname extracts the hostname from the request
func (h *DynDNSHandler) extractHostname(r *http.Request) string {
	// Standard format: /nic/update?hostname=example.com
	if strings.HasPrefix(r.URL.Path, "/nic/update") {
		return r.URL.Query().Get("hostname")
	}

	// UniFi format: /example.com
	path := strings.TrimPrefix(r.URL.Path, "/")
	if path != "" {
		// URL decode in case of encoded wildcards
		if decoded, err := url.PathUnescape(path); err == nil {
			return decoded
		}
		return path
	}

	return ""
}

// normalizeHostname normalizes the hostname (handle wildcards)
func (h *DynDNSHandler) normalizeHostname(hostname string) string {
	// Replace URL-encoded asterisk
	hostname = strings.ReplaceAll(hostname, "%2a", "*")
	hostname = strings.ReplaceAll(hostname, "%2A", "*")
	return strings.ToLower(hostname)
}

// splitHostname splits a hostname into domain and subdomain
// Example: "test.example.com" -> domain="example.com", subdomain="test"
// Example: "*.example.com" -> domain="example.com", subdomain="*"
func (h *DynDNSHandler) splitHostname(hostname string) (domain, subdomain string) {
	parts := strings.Split(hostname, ".")
	if len(parts) < 2 {
		return "", ""
	}

	// Domain is the last two parts (e.g., "example.com")
	domain = strings.Join(parts[len(parts)-2:], ".")

	// Subdomain is everything before the domain
	if len(parts) > 2 {
		subdomain = strings.Join(parts[:len(parts)-2], ".")
	} else {
		// No subdomain, use "@" for apex
		subdomain = "@"
	}

	return domain, subdomain
}

// extractIP extracts the IP address from the request
func (h *DynDNSHandler) extractIP(r *http.Request) string {
	// Check query parameter first
	if ip := r.URL.Query().Get("myip"); ip != "" {
		if net.ParseIP(ip) != nil {
			return ip
		}
	}

	// Fall back to request source IP
	ip := r.RemoteAddr
	if host, _, err := net.SplitHostPort(ip); err == nil {
		ip = host
	}

	// Handle X-Forwarded-For header
	if forwarded := r.Header.Get("X-Forwarded-For"); forwarded != "" {
		ips := strings.Split(forwarded, ",")
		if len(ips) > 0 {
			ip = strings.TrimSpace(ips[0])
		}
	}

	if net.ParseIP(ip) != nil {
		return ip
	}

	return ""
}

// updateDNS updates the DNS record via the configured provider
func (h *DynDNSHandler) updateDNS(ctx context.Context, domain, subdomain, ipAddress string) error {
	// Determine record type
	recordType := "A"
	if strings.Contains(ipAddress, ":") {
		recordType = "AAAA"
	}

	// Build full hostname
	hostname := h.buildHostname(subdomain, domain)

	// Prepare DNS record
	record := &provider.DNSRecord{
		Name:  hostname,
		Type:  recordType,
		Value: ipAddress,
		TTL:   h.config.DefaultTTL,
	}

	// Update the record via provider
	if err := h.config.Provider.UpdateRecord(ctx, domain, record); err != nil {
		return fmt.Errorf("update DNS record: %w", err)
	}

	return nil
}

// buildHostname builds a full hostname from subdomain and domain
func (h *DynDNSHandler) buildHostname(subdomain, domain string) string {
	if subdomain == "@" || subdomain == "" {
		return domain
	}
	return subdomain + "." + domain
}

// respond sends a DynDNS response
func (h *DynDNSHandler) respond(w http.ResponseWriter, status, ip string, standardFormat bool) {
	w.Header().Set("Content-Type", "text/plain")

	if standardFormat {
		// Standard format includes IP
		if ip != "" {
			fmt.Fprintf(w, "%s %s\n", status, ip)
		} else {
			fmt.Fprintf(w, "%s\n", status)
		}
	} else {
		// Simple format is just the status
		fmt.Fprintf(w, "%s\n", status)
	}
}
