# OpenAI LLM Client Tests Documentation

## Overview

The `pkg/llm/openai_test.go` file contains comprehensive unit tests for the OpenAI LLM client implementation. These tests verify that the client correctly interacts with OpenAI's API, handles various error conditions, and processes responses appropriately. The tests use a table-driven approach and mock HTTP responses to test the client's behavior in different scenarios without making actual API calls.

## Test Functions

### TestOpenAIClient_GetModelType

```go
func TestOpenAIClient_GetModelType(t *testing.T)
```

Tests that the OpenAI client returns the correct model type:

1. **Setup**: Creates a new OpenAI client
2. **Execution**: Calls the `GetModelType()` method
3. **Verification**: Checks that the returned model type is `models.OpenAI`

This test ensures that the OpenAI client correctly identifies itself as an OpenAI model, which is important for the router to route requests to the appropriate client.

### TestOpenAIClient_Query

```go
func TestOpenAIClient_Query(t *testing.T)
```

Tests the `Query` method of the OpenAI client with various scenarios:

1. **Table-Driven**: Uses a table of test cases with different API keys, status codes, response bodies, and expected errors
2. **Mock HTTP Client**: Uses a mock HTTP client to simulate responses from OpenAI's API
3. **Test Cases**:
   - **Successful query**: Tests that a successful query returns the expected response
   - **Missing API key**: Tests that a missing API key returns an appropriate error
   - **Rate limit error**: Tests that a rate limit error is correctly identified
   - **Server error**: Tests that a server error is correctly handled
   - **Empty response**: Tests that an empty response returns an appropriate error
   - **Invalid JSON response**: Tests that an invalid JSON response returns an appropriate error
4. **Response Verification**: For successful queries, verifies that:
   - Response text matches the expected text
   - Input token count matches the expected value
   - Output token count matches the expected value
   - Total token count matches the expected value
   - Error conditions are correctly identified and handled

This test ensures that the OpenAI client correctly handles various scenarios when querying OpenAI's API, including error conditions and successful responses.

### TestOpenAIClient_CheckAvailability

```go
func TestOpenAIClient_CheckAvailability(t *testing.T)
```

Tests the `CheckAvailability` method of the OpenAI client:

1. **Table-Driven**: Uses a table of test cases with different API keys, status codes, and expected results
2. **Mock HTTP Client**: Uses a mock HTTP client to simulate responses from OpenAI's API
3. **Test Cases**:
   - **Available**: Tests that the client correctly identifies when OpenAI's API is available
   - **Unavailable**: Tests that the client correctly identifies when OpenAI's API is unavailable
   - **No API key**: Tests that the client correctly identifies when no API key is configured
4. **Result Verification**: Verifies that the result matches the expected availability

This test ensures that the OpenAI client correctly checks the availability of OpenAI's API, which is important for the router to determine whether to route requests to OpenAI.

## Testing Techniques

The file demonstrates several testing techniques:

1. **Table-Driven Tests**: Uses tables of test cases to test multiple scenarios efficiently
2. **Mock HTTP Client**: Uses a mock HTTP client to simulate responses from OpenAI's API
3. **Error Checking**: Verifies that errors are correctly identified and handled
4. **Response Parsing**: Verifies that responses are correctly parsed and processed
5. **Header Verification**: Checks that the correct headers are set in requests
   - Verifies Bearer token authentication
   - Verifies Content-Type header
6. **Status Code Handling**: Tests handling of different HTTP status codes

## Test Coverage

The tests cover the following aspects of the OpenAI client:

1. **Model Type**: Tests that the client returns the correct model type
2. **API Key Handling**: Tests that the client correctly handles missing API keys
3. **Request Headers**: Verifies that the client sets the correct headers in requests
4. **Response Parsing**: Tests that the client correctly parses responses
5. **Error Handling**: Verifies that the client correctly handles various error conditions
6. **Token Counting**: Tests that the client correctly extracts token counts from responses
7. **Availability Checking**: Verifies that the client correctly checks the availability of OpenAI's API

## Dependencies

- `context`: For creating contexts for requests
- `encoding/json`: For parsing JSON responses
- `errors`: For error handling
- `io/ioutil`: For reading response bodies
- `net/http`: For HTTP client and response handling
- `strings`: For creating string readers for response bodies
- `testing`: Standard Go testing package
- `time`: For setting timeouts
- `github.com/amorin24/llmproxy/pkg/errors`: For error handling
- `github.com/amorin24/llmproxy/pkg/models`: For model types

## Integration with the OpenAI Client

These tests verify the functionality of the OpenAI client, ensuring that:

1. The client correctly implements the LLM interface
2. The client correctly interacts with OpenAI's API
3. The client correctly handles various error conditions
4. The client correctly processes responses from OpenAI's API

## Usage

Run these tests using the Go test command:

```bash
go test -v github.com/amorin24/llmproxy/pkg/llm -run TestOpenAIClient
```

These tests are also run as part of the continuous integration process to ensure that changes to the OpenAI client implementation do not break existing functionality.
