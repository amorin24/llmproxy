# Configuration Tests Documentation

## Overview

The `pkg/config/config_test.go` file contains unit tests for the configuration management system in the LLM Proxy. These tests verify that the configuration is correctly loaded from environment variables, default values are used when environment variables are not set, and helper functions for parsing environment variables work as expected.

## Test Functions

### TestGetConfig

```go
func TestGetConfig(t *testing.T)
```

Tests that the configuration is correctly loaded from environment variables:

1. **Setup**: Sets environment variables for API keys, port, cache settings, etc.
2. **Execution**: Calls `GetConfig()` to load the configuration
3. **Verification**: Checks that the configuration values match the environment variables

This test ensures that the configuration system correctly reads and parses environment variables, which is essential for the application to be configurable through environment variables.

### TestGetConfigDefaults

```go
func TestGetConfigDefaults(t *testing.T)
```

Tests that default values are used when environment variables are not set:

1. **Setup**: Unsets all environment variables and resets the configuration singleton
2. **Execution**: Calls `GetConfig()` to load the configuration
3. **Verification**: Checks that the configuration values match the expected default values

This test ensures that the application can run with reasonable defaults even when environment variables are not explicitly set, which is important for ease of use and deployment.

### TestGetEnvWithDefault

```go
func TestGetEnvWithDefault(t *testing.T)
```

Tests the helper function for getting environment variables with default values:

1. **Case 1**: Tests that the function returns the environment variable value when it is set
2. **Case 2**: Tests that the function returns the default value when the environment variable is not set

This test ensures that the helper function correctly handles both cases, which is important for the configuration system to work correctly.

### TestGetEnvAsBool

```go
func TestGetEnvAsBool(t *testing.T)
```

Tests the helper function for parsing boolean environment variables:

1. **Table-Driven**: Uses a table of test cases with different environment variable values and expected results
2. **Cases**: Tests various boolean representations ("true", "TRUE", "1", "false", "FALSE", "0", etc.)
3. **Edge Cases**: Tests invalid values and empty values

This test ensures that the helper function correctly parses boolean values from environment variables, which is important for configuration options that are boolean flags.

### TestGetEnvAsInt

```go
func TestGetEnvAsInt(t *testing.T)
```

Tests the helper function for parsing integer environment variables:

1. **Table-Driven**: Uses a table of test cases with different environment variable values and expected results
2. **Cases**: Tests positive integers, negative integers, and zero
3. **Edge Cases**: Tests invalid values and empty values

This test ensures that the helper function correctly parses integer values from environment variables, which is important for configuration options that are numeric values.

## Testing Techniques

The file demonstrates several testing techniques:

1. **Environment Variable Manipulation**: Sets and unsets environment variables to test different scenarios
2. **Singleton Reset**: Resets the configuration singleton to test initialization with different environment variables
3. **Table-Driven Tests**: Uses tables of test cases for the boolean and integer parsing functions
4. **Cleanup**: Uses `defer` to restore the original environment variables after the tests

## Dependencies

- `os`: For manipulating environment variables
- `sync`: For resetting the configuration singleton
- `testing`: Standard Go testing package

## Integration with the Config Package

These tests verify the functionality of the config package, ensuring that:

1. The `GetConfig()` function correctly loads configuration from environment variables
2. Default values are used when environment variables are not set
3. Helper functions for parsing environment variables work correctly

## Usage

Run these tests using the Go test command:

```bash
go test -v github.com/amorin24/llmproxy/pkg/config
```

These tests are also run as part of the continuous integration process to ensure that changes to the configuration implementation do not break existing functionality.
