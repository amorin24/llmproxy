# LLM Core Interface Documentation

## Overview

The `pkg/llm/llm.go` file defines the core interfaces and types for the LLM Proxy system. It provides the foundation for all LLM client implementations, including the common query result structure, client interface, factory function for creating clients, and token estimation utilities.

## Components

### QueryResult Struct

```go
type QueryResult struct {
    Response        string
    ResponseTime    int64
    StatusCode      int
    InputTokens     int
    OutputTokens    int
    TotalTokens     int
    NumTokens       int // Deprecated: Use TotalTokens instead
    NumRetries      int
    Error           error
}
```

The `QueryResult` struct represents the result of a query to any LLM provider:

- **Response**: The text response from the LLM
- **ResponseTime**: Time taken to get the response in milliseconds
- **StatusCode**: HTTP status code from the API response
- **InputTokens**: Number of tokens in the input query
- **OutputTokens**: Number of tokens in the response
- **TotalTokens**: Total number of tokens used (input + output)
- **NumTokens**: Deprecated field for backward compatibility
- **NumRetries**: Number of retry attempts made
- **Error**: Any error that occurred during the query

### Client Interface

```go
type Client interface {
    Query(ctx context.Context, query string) (*QueryResult, error)
    CheckAvailability() bool
    GetModelType() models.ModelType
}
```

The `Client` interface defines the required methods for all LLM clients:

- **Query**: Sends a query to the LLM and returns the result
  - Takes a context for cancellation and timeout
  - Takes a query string to send to the LLM
  - Returns a QueryResult and any error that occurred
- **CheckAvailability**: Checks if the LLM service is available
  - Returns true if the service is available, false otherwise
- **GetModelType**: Returns the type of LLM model
  - Used by the router to identify the client type

### Factory Function

```go
var Factory = func(modelType models.ModelType) (Client, error)
```

The `Factory` function creates LLM clients based on the model type:

1. Takes a model type parameter
2. Returns the appropriate client implementation:
   - OpenAI client for `models.OpenAI`
   - Gemini client for `models.Gemini`
   - Mistral client for `models.Mistral`
   - Claude client for `models.Claude`
3. Returns an error for unknown model types

This factory pattern allows the router to create appropriate clients dynamically based on configuration or request parameters.

### Token Estimation Functions

```go
func EstimateTokenCount(text string) int
func EstimateTokens(result *QueryResult, query, response string)
```

These functions provide token counting functionality:

#### EstimateTokenCount
- Takes a text string
- Returns an estimated token count based on text length
- Uses a simple heuristic of dividing text length by 4
- Includes special case handling for specific test strings

#### EstimateTokens
- Takes a QueryResult and the query and response strings
- Updates the token counts in the QueryResult if they're not already set
- Uses EstimateTokenCount for estimation
- Maintains backward compatibility with the deprecated NumTokens field

These functions are used when an LLM provider doesn't return token counts in their response, ensuring that token usage information is always available for monitoring and billing.

## Usage

The core interfaces and types are used throughout the LLM Proxy system:

1. **Client Implementation**:
   ```go
   type MyLLMClient struct {
       // Client-specific fields
   }
   
   func (c *MyLLMClient) Query(ctx context.Context, query string) (*QueryResult, error) {
       // Implementation
   }
   
   func (c *MyLLMClient) CheckAvailability() bool {
       // Implementation
   }
   
   func (c *MyLLMClient) GetModelType() models.ModelType {
       // Implementation
   }
   ```

2. **Client Creation**:
   ```go
   client, err := Factory(models.OpenAI)
   if err != nil {
       // Handle error
   }
   ```

3. **Query Execution**:
   ```go
   result, err := client.Query(ctx, "What is the weather?")
   if err != nil {
       // Handle error
   }
   fmt.Printf("Response: %s\n", result.Response)
   ```

## Dependencies

- `context`: For request cancellation and timeouts
- `github.com/amorin24/llmproxy/pkg/errors`: For error handling
- `github.com/amorin24/llmproxy/pkg/models`: For model type definitions

## Integration with Other Components

The core interfaces and types are integrated with other components in the system:

1. **Router**: Uses the Factory to create clients and the Client interface to interact with them
2. **API Handlers**: Use the QueryResult struct to format responses to API requests
3. **LLM Clients**: Implement the Client interface and use the QueryResult struct
4. **Error Handling**: Uses the error field in QueryResult for consistent error reporting
5. **Token Tracking**: Uses the token count fields for monitoring and billing

This central interface ensures consistency across different LLM implementations and provides a stable API for the rest of the system to interact with LLM providers.
