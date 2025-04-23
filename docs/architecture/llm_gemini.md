# Gemini LLM Client Implementation Documentation

## Overview

The `pkg/llm/gemini.go` file implements a client for interacting with Google's Gemini language model API. This client is part of the LLM Proxy system and provides a standardized interface for sending queries to Gemini, handling responses, and managing errors. The implementation includes features for authentication, request formatting, response parsing, error handling, and availability checking.

## Components

### GeminiClient Struct

```go
type GeminiClient struct {
    apiKey string
    client *http.Client
}
```

The `GeminiClient` struct represents a client for interacting with Gemini's API:

- **apiKey**: The API key for authenticating with Gemini's API
- **client**: An HTTP client for making requests to Gemini's API with a configured timeout

### Request and Response Structs

```go
type GeminiRequest struct {
    Contents []GeminiContent `json:"contents"`
    GenerationConfig GeminiGenerationConfig `json:"generationConfig"`
}

type GeminiContent struct {
    Parts []GeminiPart `json:"parts"`
}

type GeminiPart struct {
    Text string `json:"text"`
}

type GeminiGenerationConfig struct {
    Temperature float64 `json:"temperature"`
    MaxOutputTokens int `json:"maxOutputTokens"`
}

type GeminiResponse struct {
    Candidates []struct {
        Content struct {
            Parts []struct {
                Text string `json:"text"`
            } `json:"parts"`
        } `json:"content"`
        FinishReason string `json:"finishReason"`
        TokenCount struct {
            TotalTokens int `json:"totalTokens"`
        } `json:"tokenCount,omitempty"`
    } `json:"candidates"`
    Error struct {
        Code    int    `json:"code"`
        Message string `json:"message"`
        Status  string `json:"status"`
    } `json:"error"`
}
```

These structs define the structure of requests to and responses from Gemini's API:

- **GeminiRequest**: Represents a request to Gemini's API, including the content to send and generation configuration
- **GeminiContent**: Represents the content of a request, consisting of parts
- **GeminiPart**: Represents a part of the content, containing text
- **GeminiGenerationConfig**: Represents the configuration for text generation, including temperature and maximum output tokens
- **GeminiResponse**: Represents a response from Gemini's API, including the generated text, token usage, and any errors

### Constructor Function

```go
func NewGeminiClient() *GeminiClient
```

Creates a new `GeminiClient` with the API key from the configuration and a configured HTTP client. The client has a timeout of 30 seconds to prevent hanging requests.

### Interface Implementation Methods

#### GetModelType

```go
func (c *GeminiClient) GetModelType() models.ModelType
```

Returns the model type for this client, which is `models.Gemini`. This method is part of the LLM interface and allows the router to identify the client's model type.

#### Query

```go
func (c *GeminiClient) Query(ctx context.Context, query string) (*QueryResult, error)
```

Sends a query to Gemini's API and returns the result. This method:

1. Checks if the API key is configured
2. Creates a retry function that calls `executeQuery`
3. Uses the retry package to execute the query with retries
4. Returns the result or an error

This method is part of the LLM interface and provides a standardized way to query Gemini's API with retry support.

#### executeQuery

```go
func (c *GeminiClient) executeQuery(ctx context.Context, query string) (*QueryResult, error)
```

Executes a query to Gemini's API and returns the result. This method:

1. Creates a request with the query and model parameters
2. Sends the request to Gemini's API
3. Parses the response
4. Handles errors, including rate limiting and timeouts
5. Extracts the response text and token usage
6. Estimates token usage if not provided by the API

This is an internal method used by `Query` to execute the actual API request.

#### CheckAvailability

```go
func (c *GeminiClient) CheckAvailability() bool
```

Checks if Gemini's API is available by making a request to the models endpoint. This method:

1. Checks if the API key is configured
2. Creates a request to the models endpoint
3. Sends the request with a 5-second timeout
4. Returns true if the response status is OK, false otherwise

This method is part of the LLM interface and allows the router to check if Gemini's API is available before routing requests to it.

## Error Handling

The Gemini client includes comprehensive error handling:

1. **API Key Missing**: Returns an error if the API key is not configured
2. **Request Creation Errors**: Returns an error if the request cannot be created
3. **Request Sending Errors**: Returns an error if the request cannot be sent
4. **Response Reading Errors**: Returns an error if the response cannot be read
5. **Response Parsing Errors**: Returns an error if the response cannot be parsed
6. **Rate Limiting**: Returns a rate limit error if Gemini's API returns a 429 status code or error code
7. **Timeouts**: Returns a timeout error if the context is canceled or times out
8. **Empty Responses**: Returns an error if Gemini's API returns an empty response

All errors are wrapped with model-specific information using the errors package, which allows the router to handle them appropriately.

## Retry Logic

The Gemini client uses the retry package to retry failed requests:

1. **Retry Function**: The `Query` method creates a retry function that calls `executeQuery`
2. **Retry Configuration**: The retry function is executed with the default retry configuration
3. **Retryable Errors**: Only errors marked as retryable are retried, such as timeouts and rate limiting

This retry logic improves the reliability of the Gemini client by automatically retrying failed requests that are likely to succeed on retry.

## Token Counting

The Gemini client includes token counting:

1. **API-Provided Counts**: Uses the token counts provided by Gemini's API when available
2. **Estimated Counts**: Estimates token counts using the `EstimateTokens` function when API-provided counts are not available
3. **Token Usage Tracking**: Tracks total tokens for billing and monitoring

This token counting allows the LLM Proxy to track token usage for billing and monitoring purposes.

## API Integration

The Gemini client integrates with Gemini's API:

1. **API Endpoint**: Uses the `https://generativelanguage.googleapis.com/v1/models/gemini-pro:generateContent` endpoint for queries
2. **API Key**: Includes the API key as a query parameter in the URL
3. **Model Selection**: Uses the `gemini-pro` model
4. **Parameters**: Configures temperature and maximum output tokens for the request

This API integration allows the Gemini client to communicate with Gemini's API using the appropriate endpoints, authentication, and parameters.

## Dependencies

- `bytes`: For creating request bodies
- `context`: For request cancellation and timeouts
- `encoding/json`: For JSON encoding and decoding
- `fmt`: For error formatting and URL construction
- `io/ioutil`: For reading response bodies
- `net/http`: For making HTTP requests
- `time`: For timeouts and response time tracking
- `github.com/amorin24/llmproxy/pkg/config`: For getting the API key
- `github.com/amorin24/llmproxy/pkg/errors`: For error handling
- `github.com/amorin24/llmproxy/pkg/models`: For model types
- `github.com/amorin24/llmproxy/pkg/retry`: For retry logic
- `github.com/sirupsen/logrus`: For logging

## Integration with Other Components

The Gemini client is integrated with other components in the system:

1. **Router**: The router uses the Gemini client to route requests to Gemini's API
2. **Configuration**: The Gemini client uses the configuration to get the API key
3. **Error Handling**: The Gemini client uses the error handling system to create model-specific errors
4. **Retry Logic**: The Gemini client uses the retry package to retry failed requests
5. **Token Counting**: The Gemini client uses the token counting system to track token usage

This integration ensures that the Gemini client works seamlessly with the rest of the LLM Proxy system to provide a reliable and efficient interface to Gemini's API.
