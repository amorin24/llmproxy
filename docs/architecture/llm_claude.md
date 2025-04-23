# Claude LLM Client Implementation Documentation

## Overview

The `pkg/llm/claude.go` file implements a client for interacting with Anthropic's Claude language model API. This client is part of the LLM Proxy system and provides a standardized interface for sending queries to Claude, handling responses, and managing errors. The implementation includes features for authentication, request formatting, response parsing, error handling, and availability checking.

## Components

### ClaudeClient Struct

```go
type ClaudeClient struct {
    apiKey string
    client *http.Client
}
```

The `ClaudeClient` struct represents a client for interacting with Claude's API:

- **apiKey**: The API key for authenticating with Claude's API
- **client**: An HTTP client for making requests to Claude's API with a configured timeout

### Request and Response Structs

```go
type ClaudeRequest struct {
    Model       string         `json:"model"`
    Messages    []ClaudeMessage `json:"messages"`
    Temperature float64        `json:"temperature"`
    MaxTokens   int            `json:"max_tokens"`
}

type ClaudeMessage struct {
    Role    string `json:"role"`
    Content string `json:"content"`
}

type ClaudeResponse struct {
    Id      string `json:"id"`
    Content []struct {
        Text string `json:"text"`
        Type string `json:"type"`
    } `json:"content"`
    Model     string `json:"model"`
    Usage     struct {
        InputTokens  int `json:"input_tokens"`
        OutputTokens int `json:"output_tokens"`
    } `json:"usage"`
    Error struct {
        Type    string `json:"type"`
        Message string `json:"message"`
    } `json:"error"`
}
```

These structs define the structure of requests to and responses from Claude's API:

- **ClaudeRequest**: Represents a request to Claude's API, including the model to use, messages to send, temperature, and maximum tokens
- **ClaudeMessage**: Represents a message in a conversation with Claude, including the role (user or assistant) and content
- **ClaudeResponse**: Represents a response from Claude's API, including the generated text, token usage, and any errors

### Constructor Function

```go
func NewClaudeClient() *ClaudeClient
```

Creates a new `ClaudeClient` with the API key from the configuration and a configured HTTP client. The client has a timeout of 30 seconds to prevent hanging requests.

### Interface Implementation Methods

#### GetModelType

```go
func (c *ClaudeClient) GetModelType() models.ModelType
```

Returns the model type for this client, which is `models.Claude`. This method is part of the LLM interface and allows the router to identify the client's model type.

#### Query

```go
func (c *ClaudeClient) Query(ctx context.Context, query string) (*QueryResult, error)
```

Sends a query to Claude's API and returns the result. This method:

1. Checks if the API key is configured
2. Creates a retry function that calls `executeQuery`
3. Uses the retry package to execute the query with retries
4. Returns the result or an error

This method is part of the LLM interface and provides a standardized way to query Claude's API with retry support.

#### executeQuery

```go
func (c *ClaudeClient) executeQuery(ctx context.Context, query string) (*QueryResult, error)
```

Executes a query to Claude's API and returns the result. This method:

1. Creates a request with the query and model parameters
2. Sends the request to Claude's API
3. Parses the response
4. Handles errors, including rate limiting and timeouts
5. Extracts the response text and token usage
6. Estimates token usage if not provided by the API

This is an internal method used by `Query` to execute the actual API request.

#### CheckAvailability

```go
func (c *ClaudeClient) CheckAvailability() bool
```

Checks if Claude's API is available by making a request to the models endpoint. This method:

1. Checks if the API key is configured
2. Creates a request to the models endpoint
3. Sends the request with a 5-second timeout
4. Returns true if the response status is OK, false otherwise

This method is part of the LLM interface and allows the router to check if Claude's API is available before routing requests to it.

## Error Handling

The Claude client includes comprehensive error handling:

1. **API Key Missing**: Returns an error if the API key is not configured
2. **Request Creation Errors**: Returns an error if the request cannot be created
3. **Request Sending Errors**: Returns an error if the request cannot be sent
4. **Response Reading Errors**: Returns an error if the response cannot be read
5. **Response Parsing Errors**: Returns an error if the response cannot be parsed
6. **Rate Limiting**: Returns a rate limit error if Claude's API returns a 429 status code
7. **Timeouts**: Returns a timeout error if the context is canceled or times out
8. **Empty Responses**: Returns an error if Claude's API returns an empty response

All errors are wrapped with model-specific information using the errors package, which allows the router to handle them appropriately.

## Retry Logic

The Claude client uses the retry package to retry failed requests:

1. **Retry Function**: The `Query` method creates a retry function that calls `executeQuery`
2. **Retry Configuration**: The retry function is executed with the default retry configuration
3. **Retryable Errors**: Only errors marked as retryable are retried, such as timeouts and rate limiting

This retry logic improves the reliability of the Claude client by automatically retrying failed requests that are likely to succeed on retry.

## Token Counting

The Claude client includes token counting:

1. **API-Provided Counts**: Uses the token counts provided by Claude's API when available
2. **Estimated Counts**: Estimates token counts using the `EstimateTokens` function when API-provided counts are not available
3. **Token Usage Tracking**: Tracks input tokens, output tokens, and total tokens for billing and monitoring

This token counting allows the LLM Proxy to track token usage for billing and monitoring purposes.

## API Integration

The Claude client integrates with Claude's API:

1. **API Endpoint**: Uses the `https://api.anthropic.com/v1/messages` endpoint for queries
2. **API Version**: Uses the `2023-06-01` API version
3. **Authentication**: Uses the API key for authentication
4. **Model Selection**: Uses the `claude-3-sonnet-20240229` model by default
5. **Parameters**: Configures temperature and maximum tokens for the request

This API integration allows the Claude client to communicate with Claude's API using the appropriate endpoints, authentication, and parameters.

## Dependencies

- `bytes`: For creating request bodies
- `context`: For request cancellation and timeouts
- `encoding/json`: For JSON encoding and decoding
- `fmt`: For error formatting
- `io/ioutil`: For reading response bodies
- `net/http`: For making HTTP requests
- `time`: For timeouts and response time tracking
- `github.com/amorin24/llmproxy/pkg/config`: For getting the API key
- `github.com/amorin24/llmproxy/pkg/errors`: For error handling
- `github.com/amorin24/llmproxy/pkg/models`: For model types
- `github.com/amorin24/llmproxy/pkg/retry`: For retry logic
- `github.com/sirupsen/logrus`: For logging

## Integration with Other Components

The Claude client is integrated with other components in the system:

1. **Router**: The router uses the Claude client to route requests to Claude's API
2. **Configuration**: The Claude client uses the configuration to get the API key
3. **Error Handling**: The Claude client uses the error handling system to create model-specific errors
4. **Retry Logic**: The Claude client uses the retry package to retry failed requests
5. **Token Counting**: The Claude client uses the token counting system to track token usage

This integration ensures that the Claude client works seamlessly with the rest of the LLM Proxy system to provide a reliable and efficient interface to Claude's API.
