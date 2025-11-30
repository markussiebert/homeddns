package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/markussiebert/homeddns/internal/logger"
)

// Config represents the application configuration
type Config struct {
	Port         int
	Username     string
	PasswordHash string
	Provider     string
	Domain       string
	DefaultTTL   int
	SSL          bool
	CertFile     string
	KeyFile      string
}

func LoadHomeAssistantConfig() error {
	optionsPath := os.Getenv("ADDON_OPTIONS_PATH")
	if optionsPath == "" {
		logger.Debug("ADDON_OPTIONS_PATH not set, skipping Home Assistant config loading")
		return nil
	}

	logger.Info("Loading Home Assistant configuration from: %s", optionsPath)

	info, err := os.Stat(optionsPath)
	if err != nil {
		logger.Error("Failed to stat options file %s: %v", optionsPath, err)
		return fmt.Errorf("failed to stat options file %s: %w", optionsPath, err)
	}
	if info.IsDir() {
		logger.Error("Options path %s is a directory, expected a file", optionsPath)
		return fmt.Errorf("options path %s is a directory", optionsPath)
	}

	logger.Debug("Reading options file (size: %d bytes, mode: %s, uid/gid: owner info)", info.Size(), info.Mode())
	data, err := os.ReadFile(optionsPath)
	if err != nil {
		logger.Error("Failed to read options file %s: %v", optionsPath, err)
		if os.IsPermission(err) {
			logger.Error("Permission denied reading options file - Home Assistant addon may need 'hassio_api: true' or file permissions fix")
			logger.Info("Current process UID: %d, file mode: %s", os.Getuid(), info.Mode())
		}
		return fmt.Errorf("failed to read options file %s: %w", optionsPath, err)
	}

	logger.Debug("Raw JSON content: %s", string(data))

	var raw map[string]any
	if err := json.Unmarshal(data, &raw); err != nil {
		logger.Error("Failed to parse options.json: %v", err)
		logger.Debug("Invalid JSON content was: %s", string(data))
		return fmt.Errorf("failed to parse options.json: %w", err)
	}

	logger.Info("Successfully parsed options.json with %d keys", len(raw))
	logger.Debug("Parsed JSON keys: %v", getKeys(raw))

	envVarsSet := 0
	for key, value := range raw {
		envKey := strings.ToUpper(strings.ReplaceAll(key, " ", "_"))
		if envKey == "" {
			logger.Warn("Skipping empty key in options.json")
			continue
		}
		envValue := fmt.Sprintf("%v", value)

		// Mask sensitive values in logs
		logValue := envValue
		if isSensitiveKey(key) {
			logValue = maskSensitive(envValue)
		}

		logger.Debug("Setting env var: %s=%s (from JSON key: %s)", envKey, logValue, key)
		os.Setenv(envKey, envValue)
		envVarsSet++
	}

	logger.Info("Set %d environment variables from Home Assistant config", envVarsSet)
	return nil
}

// getKeys returns the keys of a map
func getKeys(m map[string]any) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}

// isSensitiveKey checks if a key contains sensitive information
func isSensitiveKey(key string) bool {
	lowerKey := strings.ToLower(key)
	sensitivePatterns := []string{
		"password", "secret", "key", "token", "hash",
		"api_key", "api_password", "access_key", "customer_number",
	}
	for _, pattern := range sensitivePatterns {
		if strings.Contains(lowerKey, pattern) {
			return true
		}
	}
	return false
}

// maskSensitive masks sensitive values for logging
func maskSensitive(value string) string {
	if len(value) == 0 {
		return "<empty>"
	}
	if len(value) <= 8 {
		return "***"
	}
	// Show first 4 and last 4 characters
	return value[:4] + "..." + value[len(value)-4:]
}

func LoadConfig() (*Config, error) {
	logger.Debug("Loading application configuration")

	if err := LoadHomeAssistantConfig(); err != nil {
		logger.Warn("Failed to load Home Assistant config: %v", err)
	}

	config := &Config{
		Port:       8080,
		DefaultTTL: 60,
		Provider:   "netcup_ccp", // default provider
	}

	logger.Debug("Default config: port=%d, ttl=%d, provider=%s", config.Port, config.DefaultTTL, config.Provider)

	// Port
	if port := os.Getenv("PORT"); port != "" {
		logger.Debug("Reading PORT from env: %s", port)
		p, err := strconv.Atoi(port)
		if err != nil {
			logger.Error("Invalid PORT value '%s': %v", port, err)
			return nil, fmt.Errorf("invalid PORT: %w", err)
		}
		config.Port = p
		logger.Debug("Set port to: %d", p)
	}

	// Auth credentials
	config.Username = os.Getenv("AUTH_USERNAME")
	if config.Username == "" {
		logger.Error("AUTH_USERNAME environment variable is not set")
		return nil, fmt.Errorf("AUTH_USERNAME is required")
	}
	logger.Debug("Auth username: %s", config.Username)

	config.PasswordHash = os.Getenv("AUTH_PASSWORD_HASH")
	if config.PasswordHash == "" {
		logger.Error("AUTH_PASSWORD_HASH environment variable is not set")
		return nil, fmt.Errorf("AUTH_PASSWORD_HASH is required")
	}
	logger.Debug("Password hash loaded (length: %d)", len(config.PasswordHash))

	// Provider selection
	if provider := os.Getenv("DNS_PROVIDER"); provider != "" {
		logger.Debug("Reading DNS_PROVIDER from env: %s", provider)
		config.Provider = strings.ToLower(provider)
		logger.Debug("Set DNS provider to: %s", config.Provider)
	} else {
		logger.Debug("DNS_PROVIDER not set, using default: %s", config.Provider)
	}

	// Domain
	config.Domain = os.Getenv("DOMAIN")
	if config.Domain == "" {
		logger.Error("DOMAIN environment variable is not set")
		return nil, fmt.Errorf("DOMAIN is required")
	}
	logger.Debug("Domain: %s", config.Domain)

	// TTL
	if ttl := os.Getenv("DNS_TTL"); ttl != "" {
		logger.Debug("Reading DNS_TTL from env: %s", ttl)
		t, err := strconv.Atoi(ttl)
		if err != nil {
			logger.Error("Invalid DNS_TTL value '%s': %v", ttl, err)
			return nil, fmt.Errorf("invalid DNS_TTL: %w", err)
		}
		config.DefaultTTL = t
		logger.Debug("Set TTL to: %d", t)
	} else {
		logger.Debug("DNS_TTL not set, using default: %d", config.DefaultTTL)
	}

	// SSL Configuration
	if ssl := os.Getenv("SSL"); ssl != "" {
		logger.Debug("Reading SSL from env: %s", ssl)
		config.SSL = ssl == "true" || ssl == "1"
		logger.Debug("Set SSL enabled to: %v", config.SSL)
	}

	// Certificate files (only relevant if SSL is enabled)
	if certFile := os.Getenv("CERTFILE"); certFile != "" {
		logger.Debug("Reading CERTFILE from env: %s", certFile)
		// Prepend /ssl/ if path is relative
		if !strings.HasPrefix(certFile, "/") {
			config.CertFile = "/ssl/" + certFile
			logger.Debug("Converted relative path to: %s", config.CertFile)
		} else {
			config.CertFile = certFile
		}
	} else {
		config.CertFile = "/ssl/fullchain.pem" // Home Assistant default
	}

	if keyFile := os.Getenv("KEYFILE"); keyFile != "" {
		logger.Debug("Reading KEYFILE from env: %s", keyFile)
		// Prepend /ssl/ if path is relative
		if !strings.HasPrefix(keyFile, "/") {
			config.KeyFile = "/ssl/" + keyFile
			logger.Debug("Converted relative path to: %s", config.KeyFile)
		} else {
			config.KeyFile = keyFile
		}
	} else {
		config.KeyFile = "/ssl/privkey.pem" // Home Assistant default
	}

	logger.Debug("SSL config: enabled=%v, certfile=%s, keyfile=%s", config.SSL, config.CertFile, config.KeyFile)

	// Validate SSL certificate files exist if SSL is enabled
	if config.SSL {
		if _, err := os.Stat(config.CertFile); err != nil {
			logger.Error("SSL certificate file not found: %s (error: %v)", config.CertFile, err)
			return nil, fmt.Errorf("SSL certificate file not found: %s", config.CertFile)
		}
		if _, err := os.Stat(config.KeyFile); err != nil {
			logger.Error("SSL key file not found: %s (error: %v)", config.KeyFile, err)
			return nil, fmt.Errorf("SSL key file not found: %s", config.KeyFile)
		}
		logger.Info("SSL certificate files validated successfully")
	}

	logger.Info("Configuration loaded successfully: provider=%s, domain=%s, port=%d, ttl=%d, ssl=%v",
		config.Provider, config.Domain, config.Port, config.DefaultTTL, config.SSL)

	return config, nil
}
