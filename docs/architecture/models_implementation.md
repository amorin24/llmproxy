# Models Package Documentation

## Overview

The `pkg/models/models.go` file defines the core data structures and types used throughout the LLM Proxy system. It provides standardized definitions for model types, task types, request and response structures, and status information. These definitions ensure consistency across the system and provide a clear contract for API interactions.

## Components

### ModelType Enum

```go
type ModelType string

const (
    OpenAI  ModelType = "openai"
    Gemini  ModelType = "gemini"
    Mistral ModelType = "mistral"
    Claude  ModelType = "claude"
)
```

The `ModelType` enum defines the supported LLM providers:

- **OpenAI**: For OpenAI's GPT models
- **Gemini**: For Google's Gemini models
- **Mistral**: For Mistral AI's models
- **Claude**: For Anthropic's Claude models

This enum is used throughout the system to identify which LLM provider to use for a query, track which provider was used in responses, and check availability status.

### TaskType Enum

```go
type TaskType string

const (
    TextGeneration   TaskType = "text_generation"
    Summarization    TaskType = "summarization"
    SentimentAnalysis TaskType = "sentiment_analysis"
    QuestionAnswering TaskType = "question_answering"
    Other            TaskType = "other"
)
```

The `TaskType` enum categorizes the type of task being requested:

- **TextGeneration**: General text generation tasks
- **Summarization**: Text summarization tasks
- **SentimentAnalysis**: Sentiment analysis of text
- **QuestionAnswering**: Question answering tasks
- **Other**: Fallback for uncategorized tasks

This categorization helps the router make intelligent decisions about which model to use based on task suitability.

### QueryRequest Struct

```go
type QueryRequest struct {
    Query     string    `json:"query"`
    Model     ModelType `json:"model,omitempty"`     // Optional - if not provided, will be determined by the proxy
    TaskType  TaskType  `json:"task_type,omitempty"` // Optional - helps with model selection
    RequestID string    `json:"request_id,omitempty"` // Optional - for tracking requests
}
```

The `QueryRequest` struct represents an incoming request to the LLM Proxy:

- **Query**: The text query to send to the LLM (required)
- **Model**: The preferred LLM provider (optional)
- **TaskType**: The type of task being requested (optional)
- **RequestID**: A unique identifier for tracking the request (optional)

This structure provides flexibility in how requests are made, allowing users to specify their preferences while also supporting automatic model selection when preferences aren't specified.

### QueryResponse Struct

```go
type QueryResponse struct {
    Response      string    `json:"response"`
    Model         ModelType `json:"model"`
    ResponseTime  int64     `json:"response_time_ms"`
    Timestamp     time.Time `json:"timestamp"`
    Cached        bool      `json:"cached"`
    Error         string    `json:"error,omitempty"`
    ErrorType     string    `json:"error_type,omitempty"`
    InputTokens   int       `json:"input_tokens,omitempty"`
    OutputTokens  int       `json:"output_tokens,omitempty"`
    TotalTokens   int       `json:"total_tokens,omitempty"`
    NumTokens     int       `json:"num_tokens,omitempty"` // Deprecated: Use TotalTokens instead
    NumRetries    int       `json:"num_retries,omitempty"`
    RequestID     string    `json:"request_id,omitempty"`
    OriginalModel ModelType `json:"original_model,omitempty"` // If fallback occurred
}
```

The `QueryResponse` struct represents a response from the LLM Proxy:

- **Response**: The text response from the LLM
- **Model**: The LLM provider that generated the response
- **ResponseTime**: Time taken to get the response in milliseconds
- **Timestamp**: When the response was generated
- **Cached**: Whether the response was served from cache
- **Error/ErrorType**: Error information if any occurred
- **Token Counts**: Input, output, and total token usage
- **NumRetries**: Number of retry attempts made
- **RequestID**: The unique identifier from the request
- **OriginalModel**: The initially requested model if fallback occurred

This comprehensive response structure provides not only the LLM's answer but also valuable metadata about the request processing, which is useful for monitoring, debugging, and billing.

### StatusResponse Struct

```go
type StatusResponse struct {
    OpenAI  bool `json:"openai"`
    Gemini  bool `json:"gemini"`
    Mistral bool `json:"mistral"`
    Claude  bool `json:"claude"`
}
```

The `StatusResponse` struct represents the availability status of each LLM provider:

- **OpenAI**: Whether OpenAI's API is available
- **Gemini**: Whether Gemini's API is available
- **Mistral**: Whether Mistral's API is available
- **Claude**: Whether Claude's API is available

This structure is used by the status endpoint to report the current availability of each LLM provider, which is useful for monitoring and debugging.

## Usage Examples

### Creating a Query Request
```go
request := models.QueryRequest{
    Query:    "What is the weather like today?",
    Model:    models.OpenAI,
    TaskType: models.QuestionAnswering,
}
```

### Creating a Query Response
```go
response := models.QueryResponse{
    Response:     "The weather is sunny with a high of 75°F.",
    Model:        models.OpenAI,
    ResponseTime: 500,
    Timestamp:    time.Now(),
    InputTokens:  10,
    OutputTokens: 15,
    TotalTokens:  25,
}
```

### Creating a Status Response
```go
status := models.StatusResponse{
    OpenAI:  true,
    Gemini:  true,
    Mistral: false,
    Claude:  true,
}
```

## Dependencies

- `time`: For timestamp handling in the QueryResponse struct

## Integration with Other Components

The models package is integrated throughout the LLM Proxy system:

1. **API Handlers**: Use QueryRequest and QueryResponse for request/response handling
2. **LLM Clients**: Implement support for specific ModelTypes
3. **Router**: Uses TaskType for intelligent routing decisions
4. **Status Endpoint**: Uses StatusResponse to report availability
5. **Logging**: Uses model structures for consistent log formatting

## Best Practices

1. **Request Creation**:
   - Always provide a Query
   - Include a RequestID for tracking
   - Specify Model and TaskType when you have preferences
2. **Response Handling**:
   - Check for Error before using Response
   - Use TotalTokens instead of the deprecated NumTokens
   - Check Cached to distinguish cached from fresh responses
3. **Model Selection**:
   - Use the ModelType enum for type safety
   - Compare with equality operators (e.g., `model == models.OpenAI`)
4. **Task Categorization**:
   - Use the TaskType enum for type safety
   - Choose the most specific task type available

This models package provides the foundation for the LLM Proxy system, ensuring consistent data structures and clear interfaces throughout the codebase.
