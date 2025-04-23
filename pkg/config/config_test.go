package config

import (
	"os"
	"sync"
	"testing"
)

func TestGetConfig(t *testing.T) {
	originalOpenAI := os.Getenv("OPENAI_API_KEY")
	originalGemini := os.Getenv("GEMINI_API_KEY")
	originalPort := os.Getenv("PORT")
	originalCacheEnabled := os.Getenv("CACHE_ENABLED")
	originalCacheTTL := os.Getenv("CACHE_TTL")
	
	defer func() {
		os.Setenv("OPENAI_API_KEY", originalOpenAI)
		os.Setenv("GEMINI_API_KEY", originalGemini)
		os.Setenv("PORT", originalPort)
		os.Setenv("CACHE_ENABLED", originalCacheEnabled)
		os.Setenv("CACHE_TTL", originalCacheTTL)
	}()
	
	os.Setenv("OPENAI_API_KEY", "test-openai-key")
	os.Setenv("GEMINI_API_KEY", "test-gemini-key")
	os.Setenv("PORT", "9090")
	os.Setenv("CACHE_ENABLED", "true")
	os.Setenv("CACHE_TTL", "600")
	
	config := GetConfig()
	
	if config.OpenAIAPIKey.Value != "test-openai-key" {
		t.Errorf("Expected OpenAIAPIKey.Value to be 'test-openai-key', got '%s'", config.OpenAIAPIKey.Value)
	}
	
	if config.GeminiAPIKey.Value != "test-gemini-key" {
		t.Errorf("Expected GeminiAPIKey.Value to be 'test-gemini-key', got '%s'", config.GeminiAPIKey.Value)
	}
	
	if config.Port != "9090" {
		t.Errorf("Expected Port to be '9090', got '%s'", config.Port)
	}
	
	if !config.CacheEnabled {
		t.Errorf("Expected CacheEnabled to be true, got %v", config.CacheEnabled)
	}
	
	if config.CacheTTL != 600 {
		t.Errorf("Expected CacheTTL to be 600, got %d", config.CacheTTL)
	}
}

func TestGetConfigDefaults(t *testing.T) {
	originalOpenAI := os.Getenv("OPENAI_API_KEY")
	originalGemini := os.Getenv("GEMINI_API_KEY")
	originalPort := os.Getenv("PORT")
	originalCacheEnabled := os.Getenv("CACHE_ENABLED")
	originalCacheTTL := os.Getenv("CACHE_TTL")
	
	defer func() {
		os.Setenv("OPENAI_API_KEY", originalOpenAI)
		os.Setenv("GEMINI_API_KEY", originalGemini)
		os.Setenv("PORT", originalPort)
		os.Setenv("CACHE_ENABLED", originalCacheEnabled)
		os.Setenv("CACHE_TTL", originalCacheTTL)
	}()
	
	config = nil
	configOnce = sync.Once{}
	
	os.Unsetenv("OPENAI_API_KEY")
	os.Unsetenv("GEMINI_API_KEY")
	os.Unsetenv("PORT")
	os.Unsetenv("CACHE_ENABLED")
	os.Unsetenv("CACHE_TTL")
	
	config := GetConfig()
	
	if config.Port != "8080" {
		t.Errorf("Expected default Port to be '8080', got '%s'", config.Port)
	}
	
	if !config.CacheEnabled {
		t.Errorf("Expected default CacheEnabled to be true, got %v", config.CacheEnabled)
	}
	
	if config.CacheTTL != 300 {
		t.Errorf("Expected default CacheTTL to be 300, got %d", config.CacheTTL)
	}
}

func TestGetEnvWithDefault(t *testing.T) {
	os.Setenv("TEST_ENV_VAR", "test-value")
	value := getEnvWithDefault("TEST_ENV_VAR", "default-value")
	if value != "test-value" {
		t.Errorf("Expected 'test-value', got '%s'", value)
	}
	
	os.Unsetenv("TEST_ENV_VAR")
	value = getEnvWithDefault("TEST_ENV_VAR", "default-value")
	if value != "default-value" {
		t.Errorf("Expected 'default-value', got '%s'", value)
	}
}

func TestGetEnvAsBool(t *testing.T) {
	testCases := []struct {
		envValue      string
		defaultValue  bool
		expectedValue bool
	}{
		{"true", false, true},
		{"TRUE", false, true},
		{"True", false, true},
		{"1", false, true},
		{"false", true, false},
		{"FALSE", true, false},
		{"False", true, false},
		{"0", true, false},
		{"invalid", true, true},
		{"", true, true},
	}
	
	for _, tc := range testCases {
		if tc.envValue == "" {
			os.Unsetenv("TEST_BOOL_VAR")
		} else {
			os.Setenv("TEST_BOOL_VAR", tc.envValue)
		}
		
		result := getEnvAsBool("TEST_BOOL_VAR", tc.defaultValue)
		if result != tc.expectedValue {
			t.Errorf("For env value '%s' and default %v, expected %v, got %v", 
				tc.envValue, tc.defaultValue, tc.expectedValue, result)
		}
	}
}

func TestGetEnvAsInt(t *testing.T) {
	testCases := []struct {
		envValue      string
		defaultValue  int
		expectedValue int
	}{
		{"123", 0, 123},
		{"-456", 0, -456},
		{"0", 42, 0},
		{"invalid", 42, 42},
		{"", 42, 42},
	}
	
	for _, tc := range testCases {
		if tc.envValue == "" {
			os.Unsetenv("TEST_INT_VAR")
		} else {
			os.Setenv("TEST_INT_VAR", tc.envValue)
		}
		
		result := getEnvAsInt("TEST_INT_VAR", tc.defaultValue)
		if result != tc.expectedValue {
			t.Errorf("For env value '%s' and default %d, expected %d, got %d", 
				tc.envValue, tc.defaultValue, tc.expectedValue, result)
		}
	}
}
