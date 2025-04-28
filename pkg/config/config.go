package config

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"os"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/joho/godotenv"
	"github.com/sirupsen/logrus"
)

const (
	defaultKeyRotationInterval = 24

	minAPIKeyLength = 8

	encryptionKeyEnvVar = "LLM_PROXY_ENCRYPTION_KEY"
)

var (
	ErrInvalidAPIKey      = errors.New("invalid API key format")
	ErrEncryptionKeyMissing = errors.New("encryption key not set")
	ErrDecryptionFailed   = errors.New("failed to decrypt API key")
)

var (
	config     *Config
	configOnce sync.Once
	
	openAIKeyPattern  = regexp.MustCompile(`^[a-zA-Z0-9_-]{8,}$`)
	geminiKeyPattern  = regexp.MustCompile(`^[a-zA-Z0-9_-]{8,}$`)
	mistralKeyPattern = regexp.MustCompile(`^[a-zA-Z0-9_-]{8,}$`)
	claudeKeyPattern  = regexp.MustCompile(`^[a-zA-Z0-9_-]{8,}$`)
	
	testKeyPattern = regexp.MustCompile(`^test_[a-zA-Z0-9_-]{8,}$`)
)

type APIKey struct {
	Value       string    // Encrypted value
	Provider    string    // Provider name (openai, gemini, etc.)
	Version     int       // Key version for rotation
	LastRotated time.Time // When the key was last rotated
	Encrypted   bool      // Whether the key is encrypted
}

func (k APIKey) String() string {
	if k.Value == "" {
		return "[not set]"
	}
	
	if !k.Encrypted {
		if len(k.Value) <= 8 {
			return "****"
		}
		return k.Value[:4] + "..." + k.Value[len(k.Value)-4:]
	}
	
	return fmt.Sprintf("[encrypted:%s:v%d]", k.Provider, k.Version)
}

type Config struct {
	OpenAIAPIKey      APIKey
	GeminiAPIKey      APIKey
	MistralAPIKey     APIKey
	ClaudeAPIKey      APIKey
	Port              string
	CacheEnabled      bool
	CacheTTL          int  // Time to live in seconds
	KeyRotationHours  int  // Hours between key rotations
	HTTPTimeout       int  // HTTP client timeout in seconds
	MaxIdleConns      int  // Maximum number of idle connections
	MaxIdleConnsPerHost int // Maximum number of idle connections per host
	IdleConnTimeout   int  // Idle connection timeout in seconds
	lastKeyCheck      time.Time
	encryptionKey     []byte
	mutex             sync.RWMutex
}

func GetConfig() *Config {
	configOnce.Do(func() {
		godotenv.Load()

		encryptionKey := os.Getenv(encryptionKeyEnvVar)
		
		config = &Config{
			OpenAIAPIKey: APIKey{
				Value:       os.Getenv("OPENAI_API_KEY"),
				Provider:    "openai",
				Version:     1,
				LastRotated: time.Now(),
				Encrypted:   false,
			},
			GeminiAPIKey: APIKey{
				Value:       os.Getenv("GEMINI_API_KEY"),
				Provider:    "gemini",
				Version:     1,
				LastRotated: time.Now(),
				Encrypted:   false,
			},
			MistralAPIKey: APIKey{
				Value:       os.Getenv("MISTRAL_API_KEY"),
				Provider:    "mistral",
				Version:     1,
				LastRotated: time.Now(),
				Encrypted:   false,
			},
			ClaudeAPIKey: APIKey{
				Value:       os.Getenv("CLAUDE_API_KEY"),
				Provider:    "claude",
				Version:     1,
				LastRotated: time.Now(),
				Encrypted:   false,
			},
			Port:               getEnvWithDefault("PORT", "8080"),
			CacheEnabled:       getEnvAsBool("CACHE_ENABLED", true),
			CacheTTL:           getEnvAsInt("CACHE_TTL", 300),
			KeyRotationHours:   getEnvAsInt("KEY_ROTATION_HOURS", defaultKeyRotationInterval),
			HTTPTimeout:        getEnvAsInt("HTTP_TIMEOUT", 30),
			MaxIdleConns:       getEnvAsInt("MAX_IDLE_CONNS", 100),
			MaxIdleConnsPerHost: getEnvAsInt("MAX_IDLE_CONNS_PER_HOST", 20),
			IdleConnTimeout:    getEnvAsInt("IDLE_CONN_TIMEOUT", 90),
			lastKeyCheck:       time.Now(),
		}
		
		if encryptionKey != "" {
			config.encryptionKey = []byte(encryptionKey)
			config.encryptAPIKeys()
		} else {
			logrus.Warn("No encryption key set. API keys will not be encrypted.")
		}
		
		config.validateAPIKeys()

		logrus.WithFields(logrus.Fields{
			"openai_key":  config.OpenAIAPIKey.String(),
			"gemini_key":  config.GeminiAPIKey.String(),
			"mistral_key": config.MistralAPIKey.String(),
			"claude_key":  config.ClaudeAPIKey.String(),
		}).Info("Configuration loaded")
	})

	if config != nil && config.KeyRotationHours > 0 {
		config.mutex.Lock()
		defer config.mutex.Unlock()
		
		if time.Since(config.lastKeyCheck).Hours() >= float64(config.KeyRotationHours) {
			config.checkForKeyRotation()
			config.lastKeyCheck = time.Now()
		}
	}

	return config
}

func (c *Config) GetAPIKey(provider string) (string, error) {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	
	var apiKey APIKey
	
	switch strings.ToLower(provider) {
	case "openai":
		apiKey = c.OpenAIAPIKey
	case "gemini":
		apiKey = c.GeminiAPIKey
	case "mistral":
		apiKey = c.MistralAPIKey
	case "claude":
		apiKey = c.ClaudeAPIKey
	default:
		return "", fmt.Errorf("unknown provider: %s", provider)
	}
	
	if !apiKey.Encrypted {
		return apiKey.Value, nil
	}
	
	if c.encryptionKey == nil || len(c.encryptionKey) == 0 {
		return "", ErrEncryptionKeyMissing
	}
	
	decrypted, err := decrypt(apiKey.Value, c.encryptionKey)
	if err != nil {
		return "", fmt.Errorf("%w: %v", ErrDecryptionFailed, err)
	}
	
	return decrypted, nil
}

func (c *Config) SetAPIKey(provider, value string) error {
	if err := c.validateAPIKeyFormat(provider, value); err != nil {
		return err
	}
	
	c.mutex.Lock()
	defer c.mutex.Unlock()
	
	apiKey := APIKey{
		Value:       value,
		Provider:    provider,
		Version:     1,
		LastRotated: time.Now(),
		Encrypted:   false,
	}
	
	if c.encryptionKey != nil && len(c.encryptionKey) > 0 {
		encrypted, err := encrypt(value, c.encryptionKey)
		if err != nil {
			return fmt.Errorf("failed to encrypt API key: %v", err)
		}
		apiKey.Value = encrypted
		apiKey.Encrypted = true
	}
	
	switch strings.ToLower(provider) {
	case "openai":
		c.OpenAIAPIKey = apiKey
	case "gemini":
		c.GeminiAPIKey = apiKey
	case "mistral":
		c.MistralAPIKey = apiKey
	case "claude":
		c.ClaudeAPIKey = apiKey
	default:
		return fmt.Errorf("unknown provider: %s", provider)
	}
	
	return nil
}

func (c *Config) RotateAPIKey(provider, newValue string) error {
	if err := c.validateAPIKeyFormat(provider, newValue); err != nil {
		return err
	}
	
	c.mutex.Lock()
	defer c.mutex.Unlock()
	
	var currentKey *APIKey
	
	switch strings.ToLower(provider) {
	case "openai":
		currentKey = &c.OpenAIAPIKey
	case "gemini":
		currentKey = &c.GeminiAPIKey
	case "mistral":
		currentKey = &c.MistralAPIKey
	case "claude":
		currentKey = &c.ClaudeAPIKey
	default:
		return fmt.Errorf("unknown provider: %s", provider)
	}
	
	currentKey.Version++
	currentKey.LastRotated = time.Now()
	currentKey.Value = newValue
	currentKey.Encrypted = false
	
	if c.encryptionKey != nil && len(c.encryptionKey) > 0 {
		encrypted, err := encrypt(newValue, c.encryptionKey)
		if err != nil {
			return fmt.Errorf("failed to encrypt API key: %v", err)
		}
		currentKey.Value = encrypted
		currentKey.Encrypted = true
	}
	
	logrus.WithFields(logrus.Fields{
		"provider": provider,
		"version":  currentKey.Version,
	}).Info("API key rotated")
	
	return nil
}

func (c *Config) encryptAPIKeys() {
	if c.encryptionKey == nil || len(c.encryptionKey) == 0 {
		return
	}
	
	if c.OpenAIAPIKey.Value != "" && !c.OpenAIAPIKey.Encrypted {
		if encrypted, err := encrypt(c.OpenAIAPIKey.Value, c.encryptionKey); err == nil {
			c.OpenAIAPIKey.Value = encrypted
			c.OpenAIAPIKey.Encrypted = true
		}
	}
	
	if c.GeminiAPIKey.Value != "" && !c.GeminiAPIKey.Encrypted {
		if encrypted, err := encrypt(c.GeminiAPIKey.Value, c.encryptionKey); err == nil {
			c.GeminiAPIKey.Value = encrypted
			c.GeminiAPIKey.Encrypted = true
		}
	}
	
	if c.MistralAPIKey.Value != "" && !c.MistralAPIKey.Encrypted {
		if encrypted, err := encrypt(c.MistralAPIKey.Value, c.encryptionKey); err == nil {
			c.MistralAPIKey.Value = encrypted
			c.MistralAPIKey.Encrypted = true
		}
	}
	
	if c.ClaudeAPIKey.Value != "" && !c.ClaudeAPIKey.Encrypted {
		if encrypted, err := encrypt(c.ClaudeAPIKey.Value, c.encryptionKey); err == nil {
			c.ClaudeAPIKey.Value = encrypted
			c.ClaudeAPIKey.Encrypted = true
		}
	}
}

func (c *Config) validateAPIKeys() {
	if c.OpenAIAPIKey.Value != "" && !c.OpenAIAPIKey.Encrypted {
		if err := c.validateAPIKeyFormat("openai", c.OpenAIAPIKey.Value); err != nil {
			logrus.Warnf("OpenAI API key validation failed: %v", err)
		}
	}
	
	if c.GeminiAPIKey.Value != "" && !c.GeminiAPIKey.Encrypted {
		if err := c.validateAPIKeyFormat("gemini", c.GeminiAPIKey.Value); err != nil {
			logrus.Warnf("Gemini API key validation failed: %v", err)
		}
	}
	
	if c.MistralAPIKey.Value != "" && !c.MistralAPIKey.Encrypted {
		if err := c.validateAPIKeyFormat("mistral", c.MistralAPIKey.Value); err != nil {
			logrus.Warnf("Mistral API key validation failed: %v", err)
		}
	}
	
	if c.ClaudeAPIKey.Value != "" && !c.ClaudeAPIKey.Encrypted {
		if err := c.validateAPIKeyFormat("claude", c.ClaudeAPIKey.Value); err != nil {
			logrus.Warnf("Claude API key validation failed: %v", err)
		}
	}
}

func (c *Config) validateAPIKeyFormat(provider, key string) error {
	if key == "" {
		return nil // Empty keys are allowed (provider will be unavailable)
	}
	
	if len(key) < minAPIKeyLength {
		return fmt.Errorf("%w: key too short", ErrInvalidAPIKey)
	}
	
	if testKeyPattern.MatchString(key) {
		logrus.Infof("Using test key for %s", provider)
		return nil
	}
	
	switch strings.ToLower(provider) {
	case "openai":
		if !openAIKeyPattern.MatchString(key) {
			return fmt.Errorf("%w: invalid OpenAI key format", ErrInvalidAPIKey)
		}
	case "gemini":
		if !geminiKeyPattern.MatchString(key) {
			return fmt.Errorf("%w: invalid Gemini key format", ErrInvalidAPIKey)
		}
	case "mistral":
		if !mistralKeyPattern.MatchString(key) {
			return fmt.Errorf("%w: invalid Mistral key format", ErrInvalidAPIKey)
		}
	case "claude":
		if !claudeKeyPattern.MatchString(key) {
			return fmt.Errorf("%w: invalid Claude key format", ErrInvalidAPIKey)
		}
	default:
		return fmt.Errorf("unknown provider: %s", provider)
	}
	
	return nil
}

func (c *Config) checkForKeyRotation() {
	providers := []string{"openai", "gemini", "mistral", "claude"}
	
	for _, provider := range providers {
		var key APIKey
		
		switch provider {
		case "openai":
			key = c.OpenAIAPIKey
		case "gemini":
			key = c.GeminiAPIKey
		case "mistral":
			key = c.MistralAPIKey
		case "claude":
			key = c.ClaudeAPIKey
		}
		
		if key.Value != "" && time.Since(key.LastRotated).Hours() >= float64(c.KeyRotationHours) {
			logrus.WithFields(logrus.Fields{
				"provider":      provider,
				"current_version": key.Version,
				"last_rotated":  key.LastRotated,
			}).Info("API key due for rotation")
			
			rotatedKeyEnv := fmt.Sprintf("%s_API_KEY_V%d", strings.ToUpper(provider), key.Version+1)
			newKey := os.Getenv(rotatedKeyEnv)
			
			if newKey != "" && newKey != key.Value {
				if err := c.RotateAPIKey(provider, newKey); err != nil {
					logrus.WithError(err).Warnf("Failed to rotate %s API key", provider)
				}
			}
		}
	}
}

func encrypt(plaintext string, key []byte) (string, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}
	
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}
	
	ciphertext := gcm.Seal(nonce, nonce, []byte(plaintext), nil)
	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

func decrypt(encrypted string, key []byte) (string, error) {
	ciphertext, err := base64.StdEncoding.DecodeString(encrypted)
	if err != nil {
		return "", err
	}
	
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}
	
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	
	if len(ciphertext) < gcm.NonceSize() {
		return "", errors.New("ciphertext too short")
	}
	
	nonce, ciphertext := ciphertext[:gcm.NonceSize()], ciphertext[gcm.NonceSize():]
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return "", err
	}
	
	return string(plaintext), nil
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
	valueLower := strings.ToLower(value)
	if valueLower == "true" || value == "1" || valueLower == "yes" {
		return true
	}
	if valueLower == "false" || value == "0" || valueLower == "no" {
		return false
	}
	return defaultValue
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
