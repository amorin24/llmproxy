# Configuration Implementation Documentation

## Overview

The `pkg/config/config.go` file implements a secure configuration management system for the LLM Proxy. It handles loading configuration from environment variables, validating and encrypting API keys, and providing a centralized access point for configuration settings throughout the application. The implementation includes features for API key rotation, format validation, and secure storage.

## Components

### APIKey Struct

```go
type APIKey struct {
    Value       string    // Encrypted value
    Provider    string    // Provider name (openai, gemini, etc.)
    Version     int       // Key version for rotation
    LastRotated time.Time // When the key was last rotated
    Encrypted   bool      // Whether the key is encrypted
}
```

This struct represents an API key with metadata for security and management:

- **Value**: The API key value, which may be encrypted
- **Provider**: The LLM provider this key belongs to
- **Version**: A version number for key rotation
- **LastRotated**: Timestamp of the last rotation
- **Encrypted**: Flag indicating whether the key is encrypted

The `String()` method provides a safe string representation of the API key, masking the actual value to prevent accidental exposure in logs or error messages.

### Config Struct

```go
type Config struct {
    OpenAIAPIKey      APIKey
    GeminiAPIKey      APIKey
    MistralAPIKey     APIKey
    ClaudeAPIKey      APIKey
    Port              string
    CacheEnabled      bool
    CacheTTL          int  // Time to live in seconds
    KeyRotationHours  int  // Hours between key rotations
    lastKeyCheck      time.Time
    encryptionKey     []byte
    mutex             sync.RWMutex
}
```

This is the main configuration struct that holds all application settings:

- **API Keys**: Separate APIKey structs for each LLM provider
- **Port**: The HTTP server port
- **Cache Settings**: Flags and values for the caching system
- **Key Rotation**: Settings for automatic API key rotation
- **Encryption**: The key used for encrypting API keys
- **Thread Safety**: A mutex for concurrent access

### Singleton Pattern

The `GetConfig()` function implements a singleton pattern to ensure that only one configuration instance exists:

```go
func GetConfig() *Config {
    configOnce.Do(func() {
        // Initialize config instance
    })
    
    // Check for key rotation if needed
    
    return config
}
```

This ensures that all components in the system use the same configuration instance, preventing inconsistencies and redundant loading of environment variables.

## Key Security Features

### API Key Encryption

The configuration system can encrypt API keys using AES-GCM encryption:

1. **Encryption Key**: Loaded from the `LLM_PROXY_ENCRYPTION_KEY` environment variable
2. **Encryption Algorithm**: AES-GCM with a random nonce
3. **Storage Format**: Base64-encoded ciphertext

When encryption is enabled, API keys are encrypted in memory to protect against memory dumps or debugging exposures.

### API Key Validation

The system validates API keys against provider-specific patterns:

```go
var (
    openAIKeyPattern  = regexp.MustCompile(`^sk-[a-zA-Z0-9]{32,}$`)
    geminiKeyPattern  = regexp.MustCompile(`^[a-zA-Z0-9_-]{39}$`)
    mistralKeyPattern = regexp.MustCompile(`^[a-zA-Z0-9]{32,}$`)
    claudeKeyPattern  = regexp.MustCompile(`^sk-[a-zA-Z0-9]{40,}$`)
)
```

These patterns ensure that API keys have the correct format for each provider, helping to catch configuration errors early.

### API Key Rotation

The system supports automatic API key rotation:

1. **Rotation Interval**: Configurable through the `KEY_ROTATION_HOURS` environment variable
2. **Version Tracking**: Each key has a version number that increments on rotation
3. **New Key Source**: New keys are loaded from environment variables with a version suffix (e.g., `OPENAI_API_KEY_V2`)

This feature allows for secure key rotation without application restarts.

## Configuration Methods

### GetAPIKey

```go
func (c *Config) GetAPIKey(provider string) (string, error)
```

Retrieves an API key for the specified provider:

1. Determines which API key to return based on the provider name
2. Decrypts the key if it's encrypted
3. Returns the plaintext key or an error if decryption fails

### SetAPIKey

```go
func (c *Config) SetAPIKey(provider, value string) error
```

Sets a new API key for the specified provider:

1. Validates the key format
2. Encrypts the key if encryption is enabled
3. Updates the appropriate field in the Config struct

### RotateAPIKey

```go
func (c *Config) RotateAPIKey(provider, newValue string) error
```

Rotates an API key for the specified provider:

1. Validates the new key format
2. Increments the key version
3. Updates the rotation timestamp
4. Encrypts the new key if encryption is enabled
5. Logs the rotation event

## Helper Functions

### Environment Variable Helpers

The file includes several helper functions for loading environment variables:

1. **getEnvWithDefault**: Gets a string environment variable with a default value
2. **getEnvAsBool**: Parses a boolean environment variable with a default value
3. **getEnvAsInt**: Parses an integer environment variable with a default value

These functions make it easy to load configuration values with appropriate defaults.

### Encryption Functions

The file includes functions for encrypting and decrypting API keys:

1. **encrypt**: Encrypts a plaintext string using AES-GCM
2. **decrypt**: Decrypts a ciphertext string using AES-GCM

These functions handle the cryptographic operations needed for API key security.

## Thread Safety

The configuration implementation is thread-safe, using a read-write mutex to protect all operations:

1. **Read Operations**: Use a read lock, allowing multiple concurrent reads
2. **Write Operations**: Use a write lock, ensuring exclusive access during writes

This allows the configuration to be safely accessed from multiple goroutines.

## Usage

The configuration is used throughout the application to access settings and API keys:

```go
// Get the configuration
config := config.GetConfig()

// Access a setting
port := config.Port

// Get an API key
openaiKey, err := config.GetAPIKey("openai")
if err != nil {
    // Handle error
}
```

The singleton pattern ensures that all components use the same configuration instance.

## Dependencies

- `crypto/aes`, `crypto/cipher`, `crypto/rand`: For encryption and decryption
- `encoding/base64`: For encoding encrypted data
- `regexp`: For API key validation
- `sync`: For thread-safe operations
- `time`: For key rotation timing
- `github.com/joho/godotenv`: For loading environment variables from .env files
- `github.com/sirupsen/logrus`: For logging

## Integration with Other Components

The configuration is integrated with other components in the system:

1. **LLM Clients**: Use the configuration to get API keys for their respective providers
2. **API Handlers**: Use the configuration to get the server port and other settings
3. **Cache**: Uses the configuration to determine whether caching is enabled and the TTL

This integration ensures that all components have access to the same configuration settings.
