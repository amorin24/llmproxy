# Error Handling Tests Documentation

## Overview

The `pkg/errors/errors_test.go` file contains comprehensive unit tests for the error handling system in the LLM Proxy. These tests verify the functionality, reliability, and thread safety of the error components, ensuring that the error handling system works correctly under various conditions and scenarios.

## Test Functions

### TestModelError

```go
func TestModelError(t *testing.T)
```

Tests the basic functionality of the ModelError struct:

1. **Table-Driven**: Uses a table of test cases with different model names, error codes, and underlying errors
2. **Constructor**: Tests the NewModelError constructor
3. **Field Values**: Verifies that the ModelError fields are correctly set
4. **Error Method**: Tests the Error() method for correct message formatting
5. **Unwrap Method**: Tests the Unwrap() method for correct error unwrapping

This test ensures that the ModelError struct correctly wraps errors with model-specific information and provides appropriate error messages.

### TestHelperFunctions

```go
func TestHelperFunctions(t *testing.T)
```

Tests the helper functions for creating different types of model errors:

1. **NewTimeoutError**: Tests creating timeout errors
2. **NewRateLimitError**: Tests creating rate limit errors
3. **NewInvalidResponseError**: Tests creating invalid response errors
4. **NewEmptyResponseError**: Tests creating empty response errors
5. **NewUnavailableError**: Tests creating service unavailable errors

This test ensures that the helper functions correctly create ModelError instances with the appropriate error codes, underlying errors, and retryable flags.

### TestErrorsIs

```go
func TestErrorsIs(t *testing.T)
```

Tests the compatibility of ModelError with Go's errors.Is function:

1. **Standard Errors**: Tests that errors.Is correctly identifies standard errors wrapped in ModelError
2. **Positive Cases**: Tests cases where errors.Is should return true
3. **Negative Cases**: Tests cases where errors.Is should return false

This test ensures that the error handling system works correctly with Go's error unwrapping mechanism, which is important for error handling throughout the application.

### TestErrorsAs

```go
func TestErrorsAs(t *testing.T)
```

Tests the compatibility of ModelError with Go's errors.As function:

1. **Type Assertion**: Tests that errors.As correctly extracts ModelError instances
2. **Field Verification**: Verifies that the extracted ModelError fields match the expected values

This test ensures that the error handling system works correctly with Go's type assertion mechanism, which is important for extracting error details throughout the application.

### TestErrorChaining

```go
func TestErrorChaining(t *testing.T)
```

Tests error chaining and unwrapping:

1. **Multiple Levels**: Tests wrapping errors in multiple levels of ModelError
2. **Unwrapping Chain**: Tests unwrapping errors through the chain
3. **Error Message**: Tests that error messages correctly include information from the entire chain

This test ensures that the error handling system correctly supports error chaining, which is important for providing detailed error information.

### TestErrorMessageFormatting

```go
func TestErrorMessageFormatting(t *testing.T)
```

Tests error message formatting for different scenarios:

1. **Simple Errors**: Tests formatting simple error messages
2. **Multiline Errors**: Tests formatting error messages with multiple lines
3. **Special Characters**: Tests formatting error messages with special characters
4. **Edge Cases**: Tests formatting empty error messages and nil errors

This test ensures that the error handling system correctly formats error messages for various types of errors.

### TestContextCancellationErrors

```go
func TestContextCancellationErrors(t *testing.T)
```

Tests integration with context cancellation errors:

1. **Context.Canceled**: Tests wrapping context.Canceled in ModelError
2. **Context.DeadlineExceeded**: Tests wrapping context.DeadlineExceeded in ModelError
3. **Error Identification**: Tests that errors.Is correctly identifies context errors

This test ensures that the error handling system correctly integrates with Go's context package, which is important for handling timeouts and cancellations.

### TestConcurrentErrorHandling

```go
func TestConcurrentErrorHandling(t *testing.T)
```

Tests concurrent error handling:

1. **Multiple Goroutines**: Launches multiple goroutines to create errors concurrently
2. **Different Error Types**: Each goroutine creates a different type of error based on its ID
3. **Verification**: Ensures that all errors are correctly created and stored

This test ensures that the error handling system is thread-safe and can be used in a concurrent environment.

## Testing Techniques

The file demonstrates several testing techniques:

1. **Table-Driven Tests**: Uses tables of test cases for most test functions
2. **Concurrency Testing**: Tests the thread safety of the error handling system
3. **Error Unwrapping**: Tests Go's error unwrapping mechanism
4. **Type Assertion**: Tests Go's type assertion mechanism
5. **Edge Cases**: Tests boundary conditions like nil errors and empty error messages

## Dependencies

- `context`: For testing integration with context cancellation
- `errors`: Standard Go errors package
- `fmt`: For formatting error messages in tests
- `strings`: For string manipulation in tests
- `sync`: For synchronization in concurrent tests
- `testing`: Standard Go testing package
- `time`: For time-related operations in context tests

## Integration with the Errors Package

These tests verify the functionality of the errors package, ensuring that:

1. The ModelError struct correctly implements the error interface
2. The helper functions correctly create different types of model errors
3. The error handling system works correctly with Go's error unwrapping mechanism
4. The error handling system is thread-safe for concurrent use

## Usage

Run these tests using the Go test command:

```bash
go test -v github.com/amorin24/llmproxy/pkg/errors
```

These tests are also run as part of the continuous integration process to ensure that changes to the error handling system do not break existing functionality.
