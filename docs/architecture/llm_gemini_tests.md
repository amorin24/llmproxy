# Gemini LLM Client Tests Documentation

## Overview

The `pkg/llm/gemini_test.go` file contains comprehensive unit tests for the Gemini LLM client implementation. These tests verify that the client correctly interacts with Gemini's API, handles various error conditions, and processes responses appropriately. The tests use a table-driven approach and mock HTTP responses to test the client's behavior in different scenarios without making actual API calls.

## Test Functions

### TestGeminiClient_GetModelType

```go
func TestGeminiClient_GetModelType(t *testing.T)
```

Tests that the Gemini client returns the correct model type:

1. **Setup**: Creates a new Gemini client
2. **Execution**: Calls the `GetModelType()` method
3. **Verification**: Checks that the returned model type is `models.Gemini`

This test ensures that the Gemini client correctly identifies itself as a Gemini model, which is important for the router to route requests to the appropriate client.

### TestGeminiClient_Query

```go
func TestGeminiClient_Query(t *testing.T)
```

Tests the `Query` method of the Gemini client with various scenarios:

1. **Table-Driven**: Uses a table of test cases with different API keys, status codes, response bodies, and expected errors
2. **Mock HTTP Client**: Uses a mock HTTP client to simulate responses from Gemini's API
3. **Test Cases**:
   - **Successful query**: Tests that a successful query returns the expected response
   - **Missing API key**: Tests that a missing API key returns an appropriate error
   - **Rate limit error**: Tests that a rate limit error is correctly identified
   - **Server error**: Tests that a server error is correctly handled
   - **Empty response**: Tests that an empty response returns an appropriate error
   - **Invalid JSON response**: Tests that an invalid JSON response returns an appropriate error
4. **Response Verification**: For successful queries, verifies that:
   - Response text matches the expected text
   - Token counts match the expected values
   - Error conditions are correctly identified and handled

This test ensures that the Gemini client correctly handles various scenarios when querying Gemini's API, including error conditions and successful responses.

### TestGeminiClient_CheckAvailability

```go
func TestGeminiClient_CheckAvailability(t *testing.T)
```

Tests the `CheckAvailability` method of the Gemini client:

1. **Table-Driven**: Uses a table of test cases with different API keys, status codes, and expected results
2. **Mock HTTP Client**: Uses a mock HTTP client to simulate responses from Gemini's API
3. **Test Cases**:
   - **Available**: Tests that the client correctly identifies when Gemini's API is available
   - **Unavailable**: Tests that the client correctly identifies when Gemini's API is unavailable
   - **No API key**: Tests that the client correctly identifies when no API key is configured
4. **Result Verification**: Verifies that the result matches the expected availability

This test ensures that the Gemini client correctly checks the availability of Gemini's API, which is important for the router to determine whether to route requests to Gemini.

## Testing Techniques

The file demonstrates several testing techniques:

1. **Table-Driven Tests**: Uses tables of test cases to test multiple scenarios efficiently
2. **Mock HTTP Client**: Uses a mock HTTP client to simulate responses from Gemini's API
3. **Error Checking**: Verifies that errors are correctly identified and handled
4. **Response Parsing**: Verifies that responses are correctly parsed and processed
5. **URL Verification**: Checks that the API key is correctly included in the request URL
6. **Header Verification**: Checks that the correct headers are set in requests
7. **Status Code Handling**: Tests handling of different HTTP status codes

## Test Coverage

The tests cover the following aspects of the Gemini client:

1. **Model Type**: Tests that the client returns the correct model type
2. **API Key Handling**: Tests that the client correctly handles missing API keys
3. **Request URL**: Verifies that the API key is correctly included in the request URL
4. **Request Headers**: Verifies that the client sets the correct headers in requests
5. **Response Parsing**: Tests that the client correctly parses responses
6. **Error Handling**: Verifies that the client correctly handles various error conditions
7. **Token Counting**: Tests that the client correctly extracts token counts from responses
8. **Availability Checking**: Verifies that the client correctly checks the availability of Gemini's API

## Dependencies

- `context`: For creating contexts for requests
- `encoding/json`: For parsing JSON responses
- `errors`: For error handling
- `io/ioutil`: For reading response bodies
- `net/http`: For HTTP client and response handling
- `strings`: For creating string readers for response bodies and URL checking
- `testing`: Standard Go testing package
- `time`: For setting timeouts
- `github.com/amorin24/llmproxy/pkg/errors`: For error handling
- `github.com/amorin24/llmproxy/pkg/models`: For model types

## Integration with the Gemini Client

These tests verify the functionality of the Gemini client, ensuring that:

1. The client correctly implements the LLM interface
2. The client correctly interacts with Gemini's API
3. The client correctly handles various error conditions
4. The client correctly processes responses from Gemini's API

## Usage

Run these tests using the Go test command:

```bash
go test -v github.com/amorin24/llmproxy/pkg/llm -run TestGeminiClient
```

These tests are also run as part of the continuous integration process to ensure that changes to the Gemini client implementation do not break existing functionality.
