package config

import (
	"fmt"
	"os"
	"sync"

	"github.com/joho/godotenv"
	"github.com/sirupsen/logrus"
)

var (
	config     *Config
	configOnce sync.Once
)

type Config struct {
	OpenAIAPIKey  string
	GeminiAPIKey  string
	MistralAPIKey string
	ClaudeAPIKey  string
	Port          string
	CacheEnabled  bool
	CacheTTL      int // Time to live in seconds
}

func GetConfig() *Config {
	configOnce.Do(func() {
		godotenv.Load()

		config = &Config{
			OpenAIAPIKey:  os.Getenv("OPENAI_API_KEY"),
			GeminiAPIKey:  os.Getenv("GEMINI_API_KEY"),
			MistralAPIKey: os.Getenv("MISTRAL_API_KEY"),
			ClaudeAPIKey:  os.Getenv("CLAUDE_API_KEY"),
			Port:          getEnvWithDefault("PORT", "8080"),
			CacheEnabled:  getEnvAsBool("CACHE_ENABLED", true),
			CacheTTL:      getEnvAsInt("CACHE_TTL", 300),
		}

		logrus.Info("Configuration loaded")
	})

	return config
}

func getEnvWithDefault(key, defaultValue string) string {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	return value
}

func getEnvAsBool(key string, defaultValue bool) bool {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	return value == "true" || value == "1" || value == "yes"
}

func getEnvAsInt(key string, defaultValue int) int {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	intValue := 0
	_, err := fmt.Sscanf(value, "%d", &intValue)
	if err != nil {
		return defaultValue
	}
	return intValue
}
