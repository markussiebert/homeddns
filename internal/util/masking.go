package util

import "strings"

const (
	// MaskingThresholdShort is the length below which values are fully masked
	MaskingThresholdShort = 4
	// MaskingThresholdLong is the length below which values use short masking
	MaskingThresholdLong = 8
)

// MaskValue masks a value for logging (shows first 2 and last 2 characters)
// Used for API keys, tokens, and other sensitive values
func MaskValue(value string) string {
	if len(value) == 0 {
		return "<empty>"
	}
	if len(value) <= MaskingThresholdShort {
		return "***"
	}
	// Show first 2 and last 2 characters
	return value[:2] + "..." + value[len(value)-2:]
}

// MaskSensitive masks sensitive values for logging (shows first 4 and last 4 characters)
// Used for longer sensitive values like password hashes
func MaskSensitive(value string) string {
	if len(value) == 0 {
		return "<empty>"
	}
	if len(value) <= MaskingThresholdLong {
		return "***"
	}
	// Show first 4 and last 4 characters
	return value[:4] + "..." + value[len(value)-4:]
}

// IsSensitiveKey checks if a key name suggests sensitive information
func IsSensitiveKey(key string) bool {
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
