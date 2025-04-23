# Logging Implementation Documentation

## Overview

The `pkg/logging/logging.go` file implements structured logging functionality for the LLM Proxy system using the logrus library. It provides standardized logging for LLM requests, responses, router activities, and error conditions, with support for different log levels, JSON formatting, and comprehensive field tracking.

## Components

### LogFields Struct

```go
type LogFields struct {
    Model           string
    Query           string
    Response        string
    ResponseTime    int64
    Cached          bool
    Error           string
    ErrorType       string
    StatusCode      int
    Timestamp       time.Time
    RequestID       string
    InputTokens     int
    OutputTokens    int
    TotalTokens     int
    NumTokens       int // Deprecated: Use TotalTokens instead
    NumRetries      int
    OriginalModel   string
    FallbackModel   string
}
```

The `LogFields` struct defines the structure for logging LLM-related events:

- **Model**: The LLM model being used (e.g., OpenAI, Gemini, etc.)
- **Query**: The input query sent to the model
- **Response**: The model's response text
- **ResponseTime**: Time taken to get the response in milliseconds
- **Cached**: Whether the response was served from cache
- **Error/ErrorType**: Error information if any occurred
- **StatusCode**: HTTP status code from the API response
- **Timestamp**: When the event occurred
- **RequestID**: Unique identifier for request tracking
- **Token Counts**: Input, output, and total token usage
- **NumRetries**: Number of retry attempts made
- **Model Selection**: Original and fallback model information

### Setup Function

```go
func SetupLogging()
```

Configures the logging system with:

1. **JSON Formatter**: Uses JSON format with nanosecond timestamp precision
2. **Output**: Directs logs to stdout
3. **Log Level**: Configurable via LOG_LEVEL environment variable
   - Defaults to "info" if not specified
   - Falls back to InfoLevel if parsing fails

### Request Logging

```go
func LogRequest(fields LogFields)
```

Logs LLM query requests with:

1. **Required Fields**:
   - Model name
   - Query text
   - Timestamp (defaults to current time if not provided)
   - Request ID
   - Event type ("llm_request")
2. **Log Level**: Uses Info level for requests
3. **Timestamp Handling**: Automatically sets current time if not provided

### Response Logging

```go
func LogResponse(fields LogFields)
```

Logs LLM query responses with comprehensive field tracking:

1. **Basic Fields**:
   - Model name
   - Response time
   - Cache status
   - Timestamp
   - Request ID
   - Event type ("llm_response")

2. **Token Usage**:
   - Total tokens (if available)
   - Input tokens (if available)
   - Output tokens (if available)
   - Legacy token count support

3. **Response Content**:
   - Truncates responses longer than 500 characters
   - Includes full response in debug mode
   - Omits empty responses

4. **Error Handling**:
   - Logs errors with Error level
   - Includes error message and type
   - Includes status code if available

5. **Performance Metrics**:
   - Response time
   - Number of retries
   - Cache status

6. **Model Selection**:
   - Original model
   - Fallback model (if used)

### Router Activity Logging

```go
func LogRouterActivity(originalModel, selectedModel string, taskType, reason string)
```

Logs model selection decisions by the router:

1. **Fields**:
   - Original requested model
   - Selected model
   - Task type
   - Selection reason
   - Timestamp
   - Event type ("router_activity")
2. **Log Level**: Uses Info level for router activities

## Usage Examples

### Basic Request Logging
```go
LogRequest(LogFields{
    Model:     "openai",
    Query:     "What is the weather?",
    RequestID: "req-123",
})
```

### Success Response Logging
```go
LogResponse(LogFields{
    Model:        "openai",
    Response:     "The weather is sunny",
    ResponseTime: 500,
    TotalTokens:  50,
    RequestID:    "req-123",
})
```

### Error Response Logging
```go
LogResponse(LogFields{
    Model:     "openai",
    Error:     "API rate limit exceeded",
    ErrorType: "rate_limit",
    RequestID: "req-123",
})
```

### Router Activity Logging
```go
LogRouterActivity(
    "openai",
    "claude",
    "text_generation",
    "openai_unavailable",
)
```

## Dependencies

- `os`: For environment variable access and stdout
- `time`: For timestamp handling
- `github.com/sirupsen/logrus`: For structured logging

## Integration with Other Components

The logging system is integrated throughout the LLM Proxy:

1. **API Handlers**: Log incoming requests and responses
2. **LLM Clients**: Log model-specific interactions
3. **Router**: Log model selection decisions
4. **Error Handling**: Log errors with appropriate context
5. **Performance Monitoring**: Track response times and token usage

## Configuration

The logging system can be configured through:

1. **Environment Variables**:
   - LOG_LEVEL: Sets the logging level (debug, info, warn, error)
2. **JSON Formatting**:
   - Uses RFC3339Nano for precise timestamps
   - Structured output for easy parsing
3. **Output Destination**:
   - Defaults to stdout
   - Can be modified in SetupLogging if needed

## Best Practices

1. **Request Tracking**:
   - Always include RequestID for correlation
   - Set appropriate timestamps
2. **Error Logging**:
   - Include both error message and type
   - Log at Error level
3. **Response Content**:
   - Truncate long responses
   - Include full content in debug mode
4. **Performance Metrics**:
   - Track response times
   - Monitor token usage
5. **Model Selection**:
   - Log both original and selected models
   - Include selection reasoning

This logging implementation provides comprehensive visibility into the LLM Proxy system's operation, helping with monitoring, debugging, and performance optimization.
