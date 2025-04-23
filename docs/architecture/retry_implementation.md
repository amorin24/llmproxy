# Retry Package Documentation

## Overview

The `pkg/retry/retry.go` file implements a robust retry mechanism with exponential backoff for handling transient errors in API calls. This package is essential for improving the reliability of the LLM Proxy system when interacting with external LLM APIs, which may occasionally experience temporary issues such as rate limiting or network problems. The implementation includes configurable retry parameters, exponential backoff with jitter, and context-aware cancellation.

## Components

### Config Struct

```go
type Config struct {
    MaxRetries     int
    InitialBackoff time.Duration
    MaxBackoff     time.Duration
    BackoffFactor  float64
    Jitter         float64
}
```

The `Config` struct defines the parameters for the retry mechanism:

- **MaxRetries**: Maximum number of retry attempts (excluding the initial attempt)
- **InitialBackoff**: Starting backoff duration before the first retry
- **MaxBackoff**: Maximum backoff duration to cap exponential growth
- **BackoffFactor**: Multiplier for exponential backoff calculation
- **Jitter**: Random factor to add variability to backoff times (0.0-1.0)

This configuration allows fine-tuning of the retry behavior based on the specific requirements of different API integrations.

### DefaultConfig

```go
var DefaultConfig = Config{
    MaxRetries:     3,
    InitialBackoff: 1 * time.Second,
    MaxBackoff:     30 * time.Second,
    BackoffFactor:  2.0,
    Jitter:         0.1,
}
```

The `DefaultConfig` provides sensible default values for the retry configuration:

- 3 retry attempts (4 total attempts including the initial one)
- 1 second initial backoff
- 30 second maximum backoff
- Exponential factor of 2.0 (doubling each time)
- 10% jitter to prevent thundering herd problems

These defaults are suitable for most API interactions and provide a good balance between persistence and avoiding overwhelming the target service.

### Do Function

```go
func Do(ctx context.Context, f func() (interface{}, error), cfg Config) (interface{}, error)
```

The `Do` function is the core of the retry package, executing a function with retries according to the provided configuration:

1. **Function Execution**: Calls the provided function and returns immediately if successful
2. **Error Classification**: Uses the custom errors package to determine if an error is retryable
3. **Retry Logic**: Implements a loop for retry attempts with backoff
4. **Logging**: Logs each retry attempt with relevant context
5. **Context Handling**: Respects context cancellation for early termination
6. **Backoff Calculation**: Uses the calculateBackoff function to determine wait time

This function provides a generic mechanism for retrying any operation that returns a value and an error, making it versatile for various use cases in the system.

### calculateBackoff Function

```go
func calculateBackoff(attempt int, cfg Config) time.Duration
```

The `calculateBackoff` function calculates the backoff duration for a specific retry attempt:

1. **Exponential Growth**: Increases backoff exponentially based on attempt number
2. **Maximum Cap**: Ensures backoff doesn't exceed the configured maximum
3. **Jitter Addition**: Adds randomness to prevent synchronized retries
4. **Duration Conversion**: Returns the final backoff as a time.Duration

This implementation follows industry best practices for retry mechanisms, using exponential backoff with jitter to provide efficient and effective retries while avoiding overwhelming the target service.

## Error Handling

The retry package integrates with the custom errors package to determine if errors are retryable:

```go
var modelErr *myerrors.ModelError
if !errors.As(err, &modelErr) || !modelErr.Retryable {
    return nil, err
}
```

This integration ensures that:

1. Only errors that are explicitly marked as retryable will trigger retry attempts
2. Non-retryable errors (e.g., authentication failures, invalid requests) are immediately returned
3. The system doesn't waste resources retrying operations that are unlikely to succeed

## Context Integration

The retry package respects context cancellation for early termination:

```go
select {
case <-ctx.Done():
    timer.Stop()
    return nil, ctx.Err()
case <-timer.C:
}
```

This integration ensures that:

1. Long-running retry sequences can be cancelled by the caller
2. The system respects deadlines and timeouts set in the context
3. Resources are properly cleaned up when cancellation occurs

## Logging

The retry package includes comprehensive logging to aid in monitoring and debugging:

```go
logrus.WithFields(logrus.Fields{
    "attempt":      attempt + 1,
    "max_attempts": cfg.MaxRetries + 1,
    "backoff_ms":   backoff.Milliseconds(),
    "error":        err.Error(),
}).Warn("Retrying request after error")
```

This logging provides visibility into:

1. Current attempt number and maximum attempts
2. Backoff duration in milliseconds
3. Error message that triggered the retry
4. Clear indication that a retry is occurring

## Usage Examples

### Basic Usage with Default Configuration
```go
result, err := retry.Do(ctx, func() (interface{}, error) {
    return client.Query(ctx, query)
}, retry.DefaultConfig)
```

### Custom Configuration
```go
config := retry.Config{
    MaxRetries:     5,
    InitialBackoff: 500 * time.Millisecond,
    MaxBackoff:     10 * time.Second,
    BackoffFactor:  1.5,
    Jitter:         0.2,
}

result, err := retry.Do(ctx, func() (interface{}, error) {
    return client.Query(ctx, query)
}, config)
```

### With Context Timeout
```go
ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
defer cancel()

result, err := retry.Do(ctx, func() (interface{}, error) {
    return client.Query(ctx, query)
}, retry.DefaultConfig)
```

## Dependencies

- `context`: For request cancellation and timeouts
- `errors`: For error type assertions
- `math`: For exponential calculations
- `math/rand`: For jitter randomization
- `time`: For duration handling
- `github.com/amorin24/llmproxy/pkg/errors`: For retryable error classification
- `github.com/sirupsen/logrus`: For structured logging

## Integration with Other Components

The retry package is integrated throughout the LLM Proxy system:

1. **LLM Clients**: Use retry for API calls to external LLM providers
2. **API Handlers**: May use retry for database operations or other external services
3. **Router**: Could use retry for availability checks
4. **Error Handling**: Integrates with the custom errors package for retryable error classification

## Best Practices

1. **Function Design**:
   - Design functions to be idempotent when using with retry
   - Ensure functions clean up resources properly if they fail
2. **Configuration**:
   - Use DefaultConfig for most cases
   - Customize configuration based on specific API characteristics
   - Consider shorter backoffs for user-facing operations
3. **Context Usage**:
   - Always provide a context with appropriate timeout
   - Cancel contexts when operations are no longer needed
4. **Error Classification**:
   - Mark only truly transient errors as retryable
   - Consider rate limiting, network issues, and temporary service unavailability as retryable
   - Consider authentication failures, validation errors, and permanent service issues as non-retryable

This retry package provides a robust foundation for reliable API interactions in the LLM Proxy system, helping to handle the inherent instability of external services while maintaining a good user experience.
