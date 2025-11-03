package logging

import (
	"regexp"
	"strings"
)

type Sanitizer struct {
	apiKeyPatterns []*regexp.Regexp
	piiPatterns    []*regexp.Regexp
	redactedMarker string
}

func NewSanitizer() *Sanitizer {
	return &Sanitizer{
		apiKeyPatterns: []*regexp.Regexp{
			regexp.MustCompile(`sk-[a-zA-Z0-9]{48}`),
			regexp.MustCompile(`(?i)api[_-]?key["\s:=]+[a-zA-Z0-9_-]{20,}`),
			regexp.MustCompile(`(?i)bearer\s+[a-zA-Z0-9_-]{20,}`),
			regexp.MustCompile(`AKIA[0-9A-Z]{16}`),
			regexp.MustCompile(`(?i)secret["\s:=]+[a-zA-Z0-9_-]{20,}`),
			regexp.MustCompile(`(?i)authorization["\s:=]+[a-zA-Z0-9_-]{20,}`),
		},
		piiPatterns: []*regexp.Regexp{
			regexp.MustCompile(`[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}`),
			regexp.MustCompile(`\b\d{3}[-.]?\d{3}[-.]?\d{4}\b`),
			regexp.MustCompile(`\b\d{4}[\s-]?\d{4}[\s-]?\d{4}[\s-]?\d{4}\b`),
			regexp.MustCompile(`\b\d{3}-\d{2}-\d{4}\b`),
		},
		redactedMarker: "[REDACTED]",
	}
}

func (s *Sanitizer) SanitizeAPIKeys(message string) string {
	sanitized := message

	for _, pattern := range s.apiKeyPatterns {
		sanitized = pattern.ReplaceAllString(sanitized, s.redactedMarker)
	}

	return sanitized
}

func (s *Sanitizer) SanitizePII(message string) string {
	sanitized := message

	for _, pattern := range s.piiPatterns {
		sanitized = pattern.ReplaceAllString(sanitized, s.redactedMarker)
	}

	return sanitized
}

func (s *Sanitizer) Sanitize(message string) string {
	sanitized := s.SanitizeAPIKeys(message)
	sanitized = s.SanitizePII(sanitized)
	return sanitized
}

func (s *Sanitizer) SanitizeMap(data map[string]interface{}) map[string]interface{} {
	sanitized := make(map[string]interface{})

	for key, value := range data {
		switch v := value.(type) {
		case string:
			sanitized[key] = s.Sanitize(v)
		case map[string]interface{}:
			sanitized[key] = s.SanitizeMap(v)
		default:
			sanitized[key] = value
		}
	}

	return sanitized
}

func (s *Sanitizer) SanitizeFields(data map[string]interface{}) map[string]interface{} {
	sensitiveFields := []string{
		"api_key", "apikey", "api-key",
		"secret", "password", "token",
		"authorization", "auth",
		"access_key", "secret_key",
	}

	sanitized := make(map[string]interface{})

	for key, value := range data {
		lowerKey := strings.ToLower(key)
		isSensitive := false

		for _, sensitive := range sensitiveFields {
			if strings.Contains(lowerKey, sensitive) {
				isSensitive = true
				break
			}
		}

		if isSensitive {
			sanitized[key] = s.redactedMarker
		} else {
			switch v := value.(type) {
			case string:
				sanitized[key] = s.Sanitize(v)
			case map[string]interface{}:
				sanitized[key] = s.SanitizeFields(v)
			default:
				sanitized[key] = value
			}
		}
	}

	return sanitized
}

func (s *Sanitizer) IsAPIKey(value string) bool {
	for _, pattern := range s.apiKeyPatterns {
		if pattern.MatchString(value) {
			return true
		}
	}
	return false
}

func (s *Sanitizer) IsPII(value string) bool {
	for _, pattern := range s.piiPatterns {
		if pattern.MatchString(value) {
			return true
		}
	}
	return false
}
