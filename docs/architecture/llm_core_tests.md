# LLM Core Tests Documentation

## Overview

The `pkg/llm/llm_test.go` file contains comprehensive unit tests for the core LLM functionality defined in `llm.go`. These tests verify the token estimation, error handling, client interface compliance, factory creation, and timeout handling functionality that is central to the LLM Proxy system.

## Test Functions

### TestEstimateTokens

```go
func TestEstimateTokens(t *testing.T)
```

Tests the token estimation functionality:

1. **Table-Driven**: Uses test cases with different query and response lengths
2. **Test Cases**:
   - Empty query and response
   - Short query and response
   - Longer query and response
   - Existing token counts (should not be overwritten)
3. **Verification**:
   - Checks input token estimation
   - Checks output token estimation
   - Verifies total tokens are sum of input and output
   - Ensures non-zero counts for non-empty text
   - Verifies existing counts are preserved

### TestIsRetryableError

```go
func TestIsRetryableError(t *testing.T)
```

Tests error classification for retry decisions:

1. **Test Cases**:
   - Nil error (not retryable)
   - Regular error (not retryable)
   - Rate limit error (retryable)
   - Timeout error (retryable)
   - Unavailable error (retryable)
   - Retryable model error
   - Non-retryable model error
2. **Verification**: Ensures errors are correctly classified for retry decisions

### TestClientInterface

```go
func TestClientInterface(t *testing.T)
```

Verifies that all LLM clients implement the Client interface:

1. **Clients Tested**:
   - OpenAIClient
   - GeminiClient
   - MistralClient
   - ClaudeClient
2. **Verification**: Uses Go's type system to ensure interface compliance

### TestFactory

```go
func TestFactory(t *testing.T)
```

Tests the LLM client factory function:

1. **Test Cases**:
   - OpenAI client creation
   - Gemini client creation
   - Mistral client creation
   - Claude client creation
   - Unknown model type
2. **Verification**:
   - Checks successful client creation
   - Verifies correct model type assignment
   - Ensures appropriate error handling

### TestQueryWithTimeout

```go
func TestQueryWithTimeout(t *testing.T)
```

Tests the query timeout functionality:

1. **Test Cases**:
   - Query completes before timeout
   - Query times out
2. **Mock Client**: Uses a mock client to simulate different query durations
3. **Verification**:
   - Checks successful query completion
   - Verifies timeout behavior
   - Ensures correct error types

### TestEstimateTokenCount

```go
func TestEstimateTokenCount(t *testing.T)
```

Tests the individual token counting function:

1. **Test Cases**:
   - Empty text
   - Short text
   - Longer text with special case handling
2. **Verification**: Ensures token counts match expected values

## Mock Types

### MockHTTPClient

```go
type MockHTTPClient struct {
    DoFunc func(req *http.Request) (*http.Response, error)
}
```

Mocks the HTTP client for testing:
- Allows customizing request handling
- Simulates different response scenarios
- Used for testing HTTP-based operations

### MockClient

```go
type MockClient struct {
    GetModelTypeFunc func() models.ModelType
    QueryFunc        func(ctx context.Context, query string) (*QueryResult, error)
    CheckAvailabilityFunc func() bool
}
```

Mocks the LLM client interface:
- Allows customizing all interface method behaviors
- Used for testing timeout and context handling
- Simulates different client scenarios

## Helper Functions

### isRetryableError

```go
func isRetryableError(err error) bool
```

Determines if an error should trigger a retry:
- Checks for rate limit errors
- Checks for timeout errors
- Checks for unavailable errors
- Checks for explicitly marked retryable errors

### QueryWithTimeout

```go
func QueryWithTimeout(client Client, query string, timeout time.Duration) (*QueryResult, error)
```

Executes a query with a timeout:
- Creates a context with timeout
- Executes the query
- Handles timeout cancellation
- Returns result or error

## Dependencies

- `context`: For timeout and cancellation
- `errors`: For error handling and type assertions
- `net/http`: For HTTP client mocking
- `testing`: Standard Go testing package
- `time`: For timeout durations
- `github.com/amorin24/llmproxy/pkg/errors`: For custom error types
- `github.com/amorin24/llmproxy/pkg/models`: For model types

## Test Coverage

The tests cover:

1. **Token Estimation**:
   - Input token counting
   - Output token counting
   - Total token calculation
   - Special case handling

2. **Error Handling**:
   - Error classification
   - Retry decisions
   - Timeout handling
   - Error wrapping

3. **Client Interface**:
   - Interface compliance
   - Factory creation
   - Model type handling
   - Availability checking

4. **Timeout Handling**:
   - Query timeouts
   - Context cancellation
   - Response timing

## Usage

Run these tests using:

```bash
go test -v github.com/amorin24/llmproxy/pkg/llm
```

These tests ensure the core LLM functionality works correctly and maintains backward compatibility. They are part of the continuous integration process to prevent regressions in the LLM Proxy system's core functionality.
