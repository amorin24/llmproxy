# Router Package Documentation

## Overview

The `pkg/router/router.go` file implements the core request routing logic for the LLM Proxy system. It provides intelligent model selection based on user preferences, task types, model availability, and fallback mechanisms. The router is responsible for determining which LLM provider to use for each request, ensuring high availability and optimal model selection for different tasks. It also provides status information about which models are currently available.

## Components

### Router Struct

```go
type Router struct {
    availableModels     map[models.ModelType]bool
    testMode            bool
    lastUpdated         time.Time
    availabilityTTL     time.Duration
    availabilityMutex   sync.RWMutex
    randomSource        *rand.Rand
    randomSourceMutex   sync.Mutex
}
```

The `Router` struct maintains the state needed for routing decisions:

- **availableModels**: Map tracking which models are currently available
- **testMode**: Flag to disable real availability checks during testing
- **lastUpdated**: Timestamp of the last availability update
- **availabilityTTL**: Time-to-live for availability information (defaults to 5 minutes)
- **availabilityMutex**: Read-write mutex for thread-safe access to availability data
- **randomSource**: Random number generator for selecting random models
- **randomSourceMutex**: Mutex for thread-safe access to the random source

This structure ensures thread-safe access to availability information and provides configuration options for testing and TTL settings.

### Constructor Function

```go
func NewRouter() *Router
```

Creates a new `Router` instance with:

1. **TTL Configuration**: Reads the `AVAILABILITY_TTL` environment variable or uses the default (5 minutes)
2. **Random Source**: Initializes a random number generator with the current time as seed
3. **Default Settings**: Sets up empty availability map and default configuration

This constructor provides a ready-to-use router with sensible defaults while allowing customization through environment variables.

### Test Mode Functions

```go
func (r *Router) SetTestMode(enabled bool)
func (r *Router) SetModelAvailability(model models.ModelType, available bool)
```

These functions support testing:

- **SetTestMode**: Enables or disables test mode, which bypasses real availability checks
- **SetModelAvailability**: Manually sets the availability of a model for testing purposes

These functions are essential for unit testing the router without making actual API calls to check availability.

### Availability Management

```go
func (r *Router) UpdateAvailability()
func (r *Router) ensureAvailabilityUpdated()
func (r *Router) GetAvailability() models.StatusResponse
```

These methods manage model availability information:

- **UpdateAvailability**: Checks the availability of all models by creating clients and calling their `CheckAvailability` method
- **ensureAvailabilityUpdated**: Ensures availability information is up-to-date, updating it if the TTL has expired
- **GetAvailability**: Returns the current availability status of all models as a `StatusResponse`

These methods ensure that the router has accurate and up-to-date information about which models are available, while respecting the TTL to avoid excessive availability checks.

### Request Routing

```go
func (r *Router) RouteRequest(ctx context.Context, req models.QueryRequest) (models.ModelType, error)
```

The core routing function that determines which model to use for a request:

1. **Context Checking**: Respects context cancellation for early termination
2. **User Preference**: Uses the user's preferred model if specified and available
3. **Task-Based Routing**: Routes based on task type if specified
4. **Random Selection**: Falls back to a random available model if no other criteria apply
5. **Logging**: Logs routing decisions with reasoning

This function implements the primary routing logic, balancing user preferences with model availability and task suitability.

### Fallback Handling

```go
func (r *Router) FallbackOnError(ctx context.Context, originalModel models.ModelType, req models.QueryRequest, err error) (models.ModelType, error)
```

Provides fallback options when a model fails:

1. **Error Classification**: Determines if the error is retryable
2. **Alternative Selection**: Finds available models excluding the failed one
3. **User Preference**: Respects user preference for fallback if specified
4. **Random Selection**: Selects a random available model if no preference
5. **Logging**: Logs fallback decisions with reasoning

This function ensures resilience by providing alternative models when the primary model fails, improving the overall reliability of the system.

### Helper Functions

```go
func (r *Router) isModelAvailable(model models.ModelType) bool
func (r *Router) routeByTaskType(taskType models.TaskType) (models.ModelType, error)
func (r *Router) getRandomAvailableModel() (models.ModelType, error)
func (r *Router) getAvailableModelsExcept(excludeModel models.ModelType) []models.ModelType
```

These helper functions support the main routing logic:

- **isModelAvailable**: Checks if a specific model is available
- **routeByTaskType**: Selects the best model for a specific task type
- **getRandomAvailableModel**: Selects a random available model
- **getAvailableModelsExcept**: Gets all available models except a specified one

These functions encapsulate specific routing logic, making the main routing functions more readable and maintainable.

## Task-Based Routing Logic

The router implements intelligent task-based routing:

```go
func (r *Router) routeByTaskType(taskType models.TaskType) (models.ModelType, error) {
    switch taskType {
    case models.TextGeneration:
        if r.isModelAvailable(models.OpenAI) {
            return models.OpenAI, nil
        }
    case models.Summarization:
        if r.isModelAvailable(models.Claude) {
            return models.Claude, nil
        }
    case models.SentimentAnalysis:
        if r.isModelAvailable(models.Gemini) {
            return models.Gemini, nil
        }
    case models.QuestionAnswering:
        if r.isModelAvailable(models.Mistral) {
            return models.Mistral, nil
        }
    }

    return r.getRandomAvailableModel()
}
```

This function maps task types to the most suitable models:

- **Text Generation**: OpenAI (GPT models excel at general text generation)
- **Summarization**: Claude (Anthropic's Claude is optimized for summarization)
- **Sentiment Analysis**: Gemini (Google's Gemini performs well on sentiment tasks)
- **Question Answering**: Mistral (Mistral AI's models are strong at Q&A)

If the preferred model for a task is unavailable, the function falls back to a random available model.

## Thread Safety

The router implements comprehensive thread safety:

1. **Read-Write Mutex**: Uses a read-write mutex for availability data
   - Read locks for operations that only read availability
   - Write locks for operations that update availability
2. **Random Source Mutex**: Uses a separate mutex for the random source
3. **Atomic Operations**: Ensures atomic updates to availability information

This thread safety ensures that the router can be safely used in concurrent environments, such as web servers handling multiple requests simultaneously.

## Logging Integration

The router integrates with the logging system:

```go
logging.LogRouterActivity(string(originalModel), string(fallbackModel), string(req.TaskType), "error_fallback")
```

This integration provides visibility into routing decisions:

1. **Original Model**: The initially selected model
2. **Selected Model**: The model ultimately chosen
3. **Task Type**: The type of task being routed
4. **Reason**: The reason for the routing decision (e.g., "user_preference", "task_type", "fallback")

This logging is essential for monitoring and debugging routing decisions.

## Error Handling

The router implements comprehensive error handling:

1. **Context Errors**: Checks for context cancellation at multiple points
2. **Unavailability Errors**: Returns specific errors when no models are available
3. **Error Classification**: Uses the custom errors package to determine if errors are retryable
4. **Error Propagation**: Propagates non-retryable errors to the caller

This error handling ensures that the router behaves predictably in error scenarios and provides useful error information to callers.

## Usage Examples

### Basic Routing
```go
router := router.NewRouter()
model, err := router.RouteRequest(ctx, models.QueryRequest{
    Query: "What is the weather like today?",
})
```

### Routing with User Preference
```go
model, err := router.RouteRequest(ctx, models.QueryRequest{
    Query: "What is the weather like today?",
    Model: models.OpenAI,
})
```

### Routing with Task Type
```go
model, err := router.RouteRequest(ctx, models.QueryRequest{
    Query: "Summarize this article: ...",
    TaskType: models.Summarization,
})
```

### Fallback Handling
```go
model, err := router.RouteRequest(ctx, req)
if err != nil {
    return err
}

result, err := client.Query(ctx, req.Query)
if err != nil {
    fallbackModel, fallbackErr := router.FallbackOnError(ctx, model, req, err)
    if fallbackErr != nil {
        return fallbackErr
    }
    
    fallbackClient, _ := llm.Factory(fallbackModel)
    result, err = fallbackClient.Query(ctx, req.Query)
}
```

### Getting Availability Status
```go
status := router.GetAvailability()
fmt.Printf("OpenAI available: %v\n", status.OpenAI)
fmt.Printf("Claude available: %v\n", status.Claude)
```

## Dependencies

- `context`: For request cancellation and timeouts
- `errors`: For error type assertions
- `math/rand`: For random model selection
- `os`: For environment variable access
- `strconv`: For string to integer conversion
- `sync`: For thread-safe access to shared data
- `time`: For TTL and timestamp handling
- `github.com/amorin24/llmproxy/pkg/errors`: For error handling
- `github.com/amorin24/llmproxy/pkg/llm`: For LLM client creation
- `github.com/amorin24/llmproxy/pkg/logging`: For logging routing decisions
- `github.com/amorin24/llmproxy/pkg/models`: For model types and request/response structures
- `github.com/sirupsen/logrus`: For structured logging

## Integration with Other Components

The router is integrated throughout the LLM Proxy system:

1. **API Handlers**: Use the router to determine which model to use for requests
2. **LLM Clients**: Created by the router to check availability and process requests
3. **Logging**: Used by the router to log routing decisions
4. **Error Handling**: Integrated with the custom errors package for error classification

## Best Practices

1. **Availability Management**:
   - Use the TTL to balance freshness with performance
   - Consider increasing the TTL for production environments
   - Consider decreasing the TTL for rapidly changing environments
2. **Routing Strategy**:
   - Customize the task-based routing for specific use cases
   - Consider adding more sophisticated routing logic for specific domains
3. **Fallback Handling**:
   - Implement circuit breakers for persistently unavailable models
   - Consider adding retry logic before falling back
4. **Thread Safety**:
   - Always use the provided mutex when accessing shared data
   - Consider using a more sophisticated locking strategy for high-concurrency environments

This router package provides a robust foundation for intelligent request routing in the LLM Proxy system, ensuring high availability and optimal model selection for different tasks.
