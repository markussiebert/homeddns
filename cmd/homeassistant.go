package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

// LoadHomeAssistantConfig reads Supervisor's add-on options and converts them into environment variables.
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
		return nil, fmt.Errorf("failed to read options file %s: %w", optionsPath, err)
	}

	var raw map[string]any
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("failed to parse options.json: %w", err)
	}

	for key, value := range raw {
		envKey := strings.ToUpper(strings.ReplaceAll(key, " ", "_"))
		if envKey == "" {
			continue
		}
		var envValue string
		switch v := value.(type) {
		case string:
			envValue = v
		case float64:
			envValue = fmt.Sprintf("%v", v)
		case bool:
			envValue = fmt.Sprintf("%t", v)
		default:
			envValue = fmt.Sprintf("%v", v)
		}
		if envValue == "" {
			continue
		}
		os.Setenv(envKey, envValue)
	}

	// Fallback to zero when no file was written
	return LoadConfig()
}
