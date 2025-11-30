package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/markussiebert/homeddns/internal/logger"
	"github.com/markussiebert/homeddns/internal/util"
)

// Config represents the application configuration
type Config struct {
	Port       int
	Username   string
	Password   string
	Provider   string
	Domain     string
	DefaultTTL int
	SSL        bool
	CertFile   string
	KeyFile    string
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
		return logger.Errorf("failed to stat options file %s: %w", optionsPath, err)
	}
	if info.IsDir() {
		return logger.Errorf("options path %s is a directory", optionsPath)
	}

	logger.Debug("Reading options file (size: %d bytes, mode: %s, uid/gid: owner info)", info.Size(), info.Mode())
	data, err := os.ReadFile(optionsPath)
	if err != nil {
		if os.IsPermission(err) {
			logger.Info("Permission denied reading options file - Home Assistant addon may need 'hassio_api: true' or file permissions fix")
			logger.Info("Current process UID: %d, file mode: %s", os.Getuid(), info.Mode())
		}
		return logger.Errorf("failed to read options file %s: %w", optionsPath, err)
	}

	logger.Debug("Raw JSON content: %s", string(data))

	var raw map[string]any
	if err := json.Unmarshal(data, &raw); err != nil {
		logger.Debug("Invalid JSON content was: %s", string(data))
		return logger.Errorf("failed to parse options.json: %w", err)
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
		if util.IsSensitiveKey(key) {
			logValue = util.MaskSensitive(envValue)
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

func LoadConfig() (*Config, error) {
	logger.Debug("Loading application configuration")

	if err := LoadHomeAssistantConfig(); err != nil {
		logger.Warn("Failed to load Home Assistant config: %v", err)
	}

	config := &Config{
		Port:       8053,
		DefaultTTL: 60,
		Provider:   "netcup_ccp", // default provider
	}

	logger.Debug("Default config: port=%d, ttl=%d, provider=%s", config.Port, config.DefaultTTL, config.Provider)

	// Port
	if port := os.Getenv("PORT"); port != "" {
		logger.Debug("Reading PORT from env: %s", port)
		p, err := strconv.Atoi(port)
		if err != nil {
			return nil, logger.Errorf("invalid PORT: %w", err)
		}
		config.Port = p
		logger.Debug("Set port to: %d", p)
	}

	// Auth credentials
	config.Username = os.Getenv("AUTH_USERNAME")
	if config.Username == "" {
		return nil, logger.Errorf("AUTH_USERNAME is required")
	}
	logger.Debug("Auth username: %s", config.Username)

	config.Password = os.Getenv("AUTH_PASSWORD")
	if config.Password == "" {
		return nil, logger.Errorf("AUTH_PASSWORD is required")
	}
	logger.Debug("Password loaded (length: %d)", len(config.Password))

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
		return nil, logger.Errorf("DOMAIN is required")
	}
	logger.Debug("Domain: %s", config.Domain)

	// TTL
	if ttl := os.Getenv("DNS_TTL"); ttl != "" {
		logger.Debug("Reading DNS_TTL from env: %s", ttl)
		t, err := strconv.Atoi(ttl)
		if err != nil {
			return nil, logger.Errorf("invalid DNS_TTL: %w", err)
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
			return nil, logger.Errorf("SSL certificate file not found: %s", config.CertFile)
		}
		if _, err := os.Stat(config.KeyFile); err != nil {
			return nil, logger.Errorf("SSL key file not found: %s", config.KeyFile)
		}
		logger.Info("SSL certificate files validated successfully")
	}

	logger.Info("Configuration loaded successfully: provider=%s, domain=%s, port=%d, ttl=%d, ssl=%v",
		config.Provider, config.Domain, config.Port, config.DefaultTTL, config.SSL)

	return config, nil
}
