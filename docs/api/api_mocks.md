# API Mocks Documentation

## Overview

The `pkg/api/mocks_test.go` file contains mock implementations of the interfaces used in the API package for testing purposes. These mocks allow the API handlers to be tested in isolation from the actual router, cache, and LLM client implementations. By providing configurable mock behaviors, tests can simulate various scenarios such as routing errors, cache hits, and LLM client failures.

## Mock Implementations

### MockRouter

```go
type MockRouter struct {
    routeRequestFunc    func(ctx context.Context, req models.QueryRequest) (models.ModelType, error)
    fallbackOnErrorFunc func(ctx context.Context, originalModel models.ModelType, req models.QueryRequest, err error) (models.ModelType, error)
    getAvailabilityFunc func() models.StatusResponse
}
```

The `MockRouter` implements the `RouterInterface` defined in the API package. It allows tests to control the behavior of the router by providing custom functions for each method.

**Methods:**

1. **RouteRequest**: Determines which LLM provider to use for a given query.
   - If `routeRequestFunc` is provided, it uses that function.
   - Otherwise, it returns `models.OpenAI` as the default model.

2. **FallbackOnError**: Determines an alternative LLM provider to use when the primary provider fails.
   - If `fallbackOnErrorFunc` is provided, it uses that function.
   - Otherwise, it returns `models.Gemini` as the default fallback model.

3. **GetAvailability**: Provides information about the availability of different LLM providers.
   - If `getAvailabilityFunc` is provided, it uses that function.
   - Otherwise, it returns a status response with all models available.

The `MockRouter` also includes several stub methods that are part of the router implementation but not used in tests:
- `SetTestMode`
- `SetModelAvailability`
- `UpdateAvailability`
- `ensureAvailabilityUpdated`
- `isModelAvailable`
- `routeByTaskType`
- `getRandomAvailableModel`
- `getAvailableModelsExcept`

### MockCache

```go
type MockCache struct {
    mutex   sync.RWMutex
    getFunc func(req models.QueryRequest) (models.QueryResponse, bool)
    setFunc func(req models.QueryRequest, resp models.QueryResponse)
}
```

The `MockCache` implements the `CacheInterface` defined in the API package. It allows tests to control the behavior of the cache by providing custom functions for each method.

**Methods:**

1. **Get**: Retrieves a cached response for a given query request.
   - Uses a read lock to ensure thread safety.
   - If `getFunc` is provided, it uses that function.
   - Otherwise, it returns an empty response and `false` to indicate a cache miss.

2. **Set**: Stores a response in the cache for future use.
   - Uses a write lock to ensure thread safety.
   - If `setFunc` is provided, it uses that function.
   - Otherwise, it does nothing.

The use of mutexes in the `MockCache` ensures that it can be safely used in concurrent tests, mimicking the behavior of a real cache implementation.

### MockLLMClient

```go
type MockLLMClient struct {
    modelType models.ModelType
    queryFunc func(ctx context.Context, query string) (*llm.QueryResult, error)
}
```

The `MockLLMClient` implements the `llm.Client` interface. It allows tests to control the behavior of the LLM client by providing a custom function for the `Query` method.

**Methods:**

1. **Query**: Sends a query to the LLM provider and returns the result.
   - If `queryFunc` is provided, it uses that function.
   - Otherwise, it returns a default mock response.

2. **GetModelType**: Returns the model type of the client.
   - Returns the `modelType` field.

3. **CheckAvailability**: Checks if the LLM provider is available.
   - Always returns `true` in the mock implementation.

## Usage in Tests

These mock implementations are used extensively in the API handler tests to simulate various scenarios:

1. **MockRouter**:
   - Simulate routing to different models based on request parameters.
   - Simulate routing errors.
   - Simulate fallback behavior when a model fails.
   - Provide custom availability information.

2. **MockCache**:
   - Simulate cache hits and misses.
   - Verify that responses are properly cached.

3. **MockLLMClient**:
   - Simulate successful queries with custom responses.
   - Simulate query failures with custom errors.
   - Simulate context cancellation and timeouts.

Example usage in a test:

```go
func TestQueryHandler(t *testing.T) {
    // Create mock router with custom behavior
    mockRouter := &MockRouter{
        routeRequestFunc: func(ctx context.Context, req models.QueryRequest) (models.ModelType, error) {
            return models.OpenAI, nil
        },
    }
    
    // Create mock cache with custom behavior
    mockCache := &MockCache{
        getFunc: func(req models.QueryRequest) (models.QueryResponse, bool) {
            return models.QueryResponse{}, false // Cache miss
        },
    }
    
    // Create handler with mocks
    handler := &Handler{
        router:      mockRouter,
        cache:       mockCache,
        rateLimiter: NewRateLimiter(60, 10),
    }
    
    // Test the handler
    // ...
}
```

## Dependencies

- `context`: Standard Go package for context handling
- `sync`: Standard Go package for synchronization primitives
- `github.com/amorin24/llmproxy/pkg/llm`: For the LLM client interface
- `github.com/amorin24/llmproxy/pkg/models`: For data models used in the interfaces

## Integration with the API Package

These mock implementations are used exclusively in tests and are not part of the production code. They allow the API handlers to be tested in isolation from the actual router, cache, and LLM client implementations, making the tests more reliable and focused on the handler logic.
