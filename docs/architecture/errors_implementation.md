# Error Handling Implementation Documentation

## Overview

The `pkg/errors/errors.go` file implements a comprehensive error handling system for the LLM Proxy. It defines standard error types for common error conditions and provides a structured way to wrap errors with model-specific information. This error system enables consistent error reporting, proper error categorization, and supports retry logic by indicating which errors are retryable.

## Standard Error Types

```go
var (
    ErrTimeout        = errors.New("request timed out")
    ErrRateLimit      = errors.New("rate limit exceeded")
    ErrInvalidResponse = errors.New("invalid response")
    ErrEmptyResponse  = errors.New("empty response from LLM")
    ErrAPIKeyMissing  = errors.New("API key not configured")
    ErrUnavailable    = errors.New("service unavailable")
)
```

These predefined error variables represent common error conditions that can occur when interacting with LLM services:

- **ErrTimeout**: Indicates that a request to an LLM service timed out
- **ErrRateLimit**: Indicates that a rate limit was exceeded for an LLM service
- **ErrInvalidResponse**: Indicates that an LLM service returned an invalid response
- **ErrEmptyResponse**: Indicates that an LLM service returned an empty response
- **ErrAPIKeyMissing**: Indicates that an API key is not configured for an LLM service
- **ErrUnavailable**: Indicates that an LLM service is unavailable

These standard error types provide a consistent way to represent common error conditions throughout the application.

## ModelError Struct

```go
type ModelError struct {
    Model     string
    Code      int
    Err       error
    Retryable bool
}
```

The `ModelError` struct wraps an error with model-specific information:

- **Model**: The name of the LLM model that generated the error
- **Code**: An HTTP-like status code representing the error type
- **Err**: The underlying error
- **Retryable**: A flag indicating whether the error is retryable

This struct allows errors to be associated with specific models and provides additional context for error handling and reporting.

### Error Method

```go
func (e *ModelError) Error() string
```

Implements the `error` interface by returning a formatted error message that includes the model name, error message, and code. This method ensures that `ModelError` can be used as a standard Go error.

### Unwrap Method

```go
func (e *ModelError) Unwrap() error
```

Implements the `Unwrapper` interface by returning the underlying error. This allows `ModelError` to work with Go's error unwrapping functions like `errors.Is` and `errors.As`.

## Constructor Functions

### NewModelError

```go
func NewModelError(model string, code int, err error, retryable bool) *ModelError
```

Creates a new `ModelError` with the specified model name, code, underlying error, and retryable flag. This is the base constructor used by the more specific error constructors.

### NewTimeoutError

```go
func NewTimeoutError(model string) *ModelError
```

Creates a new `ModelError` for a timeout error with the specified model name. Uses HTTP status code 408 (Request Timeout) and marks the error as retryable.

### NewRateLimitError

```go
func NewRateLimitError(model string) *ModelError
```

Creates a new `ModelError` for a rate limit error with the specified model name. Uses HTTP status code 429 (Too Many Requests) and marks the error as retryable.

### NewInvalidResponseError

```go
func NewInvalidResponseError(model string, err error) *ModelError
```

Creates a new `ModelError` for an invalid response error with the specified model name and underlying error. Uses HTTP status code 500 (Internal Server Error) and marks the error as non-retryable.

### NewEmptyResponseError

```go
func NewEmptyResponseError(model string) *ModelError
```

Creates a new `ModelError` for an empty response error with the specified model name. Uses HTTP status code 500 (Internal Server Error) and marks the error as non-retryable.

### NewUnavailableError

```go
func NewUnavailableError(model string) *ModelError
```

Creates a new `ModelError` for a service unavailable error with the specified model name. Uses HTTP status code 503 (Service Unavailable) and marks the error as retryable.

## Usage

The error handling system is used throughout the application to create and handle errors in a consistent way:

```go
// Creating a model-specific error
if response == "" {
    return nil, errors.NewEmptyResponseError("openai")
}

// Checking if an error is a specific type
if errors.Is(err, errors.ErrRateLimit) {
    // Handle rate limit error
}

// Extracting a ModelError from an error
var modelErr *errors.ModelError
if errors.As(err, &modelErr) {
    if modelErr.Retryable {
        // Retry the request
    }
}
```

This consistent error handling approach makes it easier to identify and handle different types of errors throughout the application.

## Integration with Other Components

The error handling system is integrated with other components in the system:

1. **LLM Clients**: Use the error constructors to create model-specific errors
2. **Router**: Uses the `Retryable` flag to determine whether to retry a request
3. **API Handlers**: Use the error types to return appropriate HTTP status codes

This integration ensures that errors are handled consistently throughout the application and that appropriate actions are taken based on the error type.

## Dependencies

- `errors`: Standard Go errors package
- `fmt`: For formatting error messages

## Error Codes

The error handling system uses HTTP-like status codes to represent different types of errors:

- **408**: Request Timeout
- **429**: Too Many Requests
- **500**: Internal Server Error
- **503**: Service Unavailable

These codes are used to categorize errors and determine appropriate actions, such as retrying the request or returning an appropriate HTTP status code to the client.
