# Router Package Tests Documentation

## Overview

The `pkg/router/router_test.go` file contains comprehensive unit tests for the router package, verifying its functionality across various scenarios. These tests ensure that the router correctly handles model selection based on user preferences, task types, and availability, as well as fallback mechanisms, concurrent access, and availability updates. The tests use a table-driven approach and test mode to simulate different routing scenarios without requiring actual API calls.

## Test Functions

### TestRouteRequest

```go
func TestRouteRequest(t *testing.T)
```

Tests the main routing function with various scenarios:

1. **User specified model available**: Verifies that the router respects user preferences when the requested model is available
2. **User specified model unavailable**: Confirms that the router falls back to an available model when the requested model is unavailable
3. **Route by task type**: Tests routing based on task type when no model is specified
4. **Task type model unavailable**: Verifies fallback to a random available model when the preferred model for a task type is unavailable
5. **Random model selection**: Tests selection of a random available model when no model or task type is specified
6. **No models available**: Confirms that an error is returned when no models are available
7. **Context cancellation**: Tests that context cancellation is respected
8. **Model preference with task type**: Verifies that model preference overrides task type routing
9. **Limited availability**: Tests routing when only one model is available

This test ensures that the core routing logic works correctly in all expected scenarios.

### TestFallbackOnError

```go
func TestFallbackOnError(t *testing.T)
```

Tests the fallback mechanism when errors occur:

1. **Fallback on retryable error**: Verifies that the router provides an alternative model when a retryable error occurs
2. **No fallback available**: Confirms that an error is returned when no alternative models are available
3. **Non-retryable error**: Tests that non-retryable errors are not retried
4. **Retryable error with custom error**: Verifies handling of custom error types
5. **Fallback with specific model preference**: Tests that user preferences are respected during fallback
6. **Multiple fallback attempts**: Confirms that multiple fallback options are considered
7. **Context cancellation during fallback**: Tests that context cancellation is respected during fallback

This test ensures that the fallback mechanism correctly handles errors and provides alternative models when appropriate.

### TestGetAvailability

```go
func TestGetAvailability(t *testing.T)
```

Tests retrieving model availability status:

1. **Availability reporting**: Verifies that the router correctly reports the availability status of all models
2. **Test mode**: Confirms that test mode allows manual setting of availability

This test ensures that the availability reporting mechanism works correctly.

### TestGetRandomAvailableModel

```go
func TestGetRandomAvailableModel(t *testing.T)
```

Tests random model selection:

1. **No models available**: Verifies that an error is returned when no models are available
2. **Single model available**: Confirms that the only available model is selected
3. **Multiple models available**: Tests that one of the available models is randomly selected

This test ensures that the random model selection mechanism works correctly.

### TestGetAvailableModelsExcept

```go
func TestGetAvailableModelsExcept(t *testing.T)
```

Tests excluding specific models:

1. **No models available**: Verifies that an empty list is returned when no models are available
2. **Exclude only available model**: Confirms that an empty list is returned when the only available model is excluded
3. **Exclude one of multiple models**: Tests that the excluded model is not in the result
4. **Multiple available models**: Verifies that all available models except the excluded one are returned

This test ensures that the model exclusion mechanism works correctly.

### TestRouteByTaskType

```go
func TestRouteByTaskType(t *testing.T)
```

Tests task-based routing:

1. **Text generation**: Verifies that OpenAI is selected for text generation tasks
2. **Summarization**: Confirms that Claude is selected for summarization tasks
3. **Sentiment analysis**: Tests that Gemini is selected for sentiment analysis tasks
4. **Question answering**: Verifies that Mistral is selected for question answering tasks
5. **Preferred model unavailable**: Tests fallback to an alternative model when the preferred model for a task type is unavailable
6. **Unknown task type**: Confirms that a random available model is selected for unknown task types
7. **No models available**: Verifies that an error is returned when no models are available

This test ensures that the task-based routing mechanism works correctly.

### TestConcurrentAccess

```go
func TestConcurrentAccess(t *testing.T)
```

Tests thread safety with concurrent operations:

1. **Concurrent readers**: Runs multiple goroutines that read availability information and get random models
2. **Concurrent writers**: Runs goroutines that update model availability
3. **Consistency verification**: Checks that the final state is consistent after all concurrent operations

This test ensures that the router is thread-safe and can be used in concurrent environments.

### TestEnsureAvailabilityUpdated

```go
func TestEnsureAvailabilityUpdated(t *testing.T)
```

Tests the TTL-based availability update mechanism:

1. **Initial update**: Verifies that the first call to ensureAvailabilityUpdated updates the availability information
2. **TTL not expired**: Confirms that a second call within the TTL does not update the availability information
3. **TTL expired**: Tests that a call after the TTL expires updates the availability information

This test ensures that the TTL-based availability update mechanism works correctly.

## Testing Techniques

The file demonstrates several testing techniques:

1. **Table-Driven Tests**: Uses tables of test cases to test multiple scenarios efficiently
2. **Test Mode**: Uses a test mode to bypass real availability checks
3. **Manual Availability Setting**: Sets model availability manually for testing
4. **Context Cancellation**: Tests behavior with cancelled contexts
5. **Error Checking**: Verifies that errors are correctly identified and handled
6. **Concurrency Testing**: Tests behavior in concurrent environments
7. **TTL Testing**: Tests time-based behavior using short TTLs and sleep

## Test Coverage

The tests cover the following aspects of the router package:

1. **Routing Logic**: Tests that requests are routed to the appropriate model
2. **Fallback Mechanism**: Verifies that fallback works correctly when errors occur
3. **Availability Reporting**: Tests that availability status is correctly reported
4. **Random Selection**: Verifies that random model selection works correctly
5. **Model Exclusion**: Tests that models can be excluded from selection
6. **Task-Based Routing**: Verifies that task-based routing works correctly
7. **Thread Safety**: Tests that the router is thread-safe
8. **TTL Mechanism**: Verifies that the TTL-based availability update mechanism works correctly

## Dependencies

- `context`: For creating contexts for tests
- `errors`: For creating and checking errors
- `sync`: For synchronizing concurrent tests
- `testing`: Standard Go testing package
- `time`: For timing operations and setting timeouts
- `github.com/amorin24/llmproxy/pkg/errors`: For creating model-specific errors
- `github.com/amorin24/llmproxy/pkg/models`: For model types and request/response structures

## Integration with the Router Package

These tests verify the functionality of the router package, ensuring that:

1. The router correctly implements the routing logic
2. The fallback mechanism correctly handles errors
3. The availability reporting mechanism works correctly
4. The random model selection mechanism works correctly
5. The model exclusion mechanism works correctly
6. The task-based routing mechanism works correctly
7. The router is thread-safe
8. The TTL-based availability update mechanism works correctly

## Usage

Run these tests using the Go test command:

```bash
go test -v github.com/amorin24/llmproxy/pkg/router
```

These tests are also run as part of the continuous integration process to ensure that changes to the router package implementation do not break existing functionality.

## Best Practices Demonstrated

The tests demonstrate several best practices for testing Go code:

1. **Table-Driven Tests**: Using tables of test cases to test multiple scenarios efficiently
2. **Isolated Tests**: Each test function focuses on a specific aspect of the router package
3. **Clear Test Names**: Test names clearly indicate what is being tested
4. **Expected vs. Actual**: Tests clearly compare expected and actual results
5. **Error Messages**: Error messages include both expected and actual values
6. **Test Mode**: Using a test mode to bypass real availability checks
7. **Concurrency Testing**: Testing behavior in concurrent environments
8. **TTL Testing**: Testing time-based behavior using short TTLs and sleep

These best practices ensure that the tests are comprehensive, maintainable, and provide clear feedback when failures occur.

## Key Test Scenarios

### Routing Priority

The tests verify that the router follows the correct priority order for routing:

1. User preference (if available)
2. Task type (if specified and the preferred model is available)
3. Random available model (as a fallback)

### Fallback Behavior

The tests confirm that the fallback mechanism:

1. Only retries retryable errors
2. Respects user preferences for fallback
3. Excludes the failed model from fallback options
4. Returns an error when no fallback options are available

### Concurrency Handling

The concurrency tests verify that:

1. Multiple readers can access availability information simultaneously
2. Writers can update availability information while readers are accessing it
3. The final state is consistent after all concurrent operations

### TTL Behavior

The TTL tests confirm that:

1. Availability information is updated on the first call
2. Subsequent calls within the TTL do not update the information
3. Calls after the TTL expires update the information

These key test scenarios ensure that the router behaves correctly in all expected usage patterns.
