# API Interfaces Documentation

## Overview

The `pkg/api/interfaces.go` file defines the core interfaces used by the API package in the LLM Proxy system. These interfaces establish the contracts between the API handlers and other components of the system, specifically the router and cache components. By defining these interfaces, the API package can interact with these components without being tightly coupled to their implementations, enabling better testability and modularity.

## Interfaces

### RouterInterface

```go
type RouterInterface interface {
    RouteRequest(ctx context.Context, req models.QueryRequest) (models.ModelType, error)
    FallbackOnError(ctx context.Context, originalModel models.ModelType, req models.QueryRequest, err error) (models.ModelType, error)
    GetAvailability() models.StatusResponse
}
```

The `RouterInterface` defines the contract for components that route LLM queries to appropriate providers. It includes three methods:

1. **RouteRequest**: Determines which LLM provider to use for a given query based on various factors such as task type, user preferences, and model availability.
   - Parameters:
     - `ctx context.Context`: The request context, which can be used for cancellation and timeouts
     - `req models.QueryRequest`: The query request containing the query text, preferred model, and task type
   - Returns:
     - `models.ModelType`: The selected model type (e.g., OpenAI, Gemini, Mistral, Claude)
     - `error`: An error if no suitable model is available

2. **FallbackOnError**: Determines an alternative LLM provider to use when the primary provider fails.
   - Parameters:
     - `ctx context.Context`: The request context
     - `originalModel models.ModelType`: The original model that failed
     - `req models.QueryRequest`: The query request
     - `err error`: The error that occurred with the original model
   - Returns:
     - `models.ModelType`: The fallback model type
     - `error`: An error if no suitable fallback is available

3. **GetAvailability**: Provides information about the availability of different LLM providers.
   - Returns:
     - `models.StatusResponse`: A struct containing availability information for each LLM provider

### CacheInterface

```go
type CacheInterface interface {
    Get(req models.QueryRequest) (models.QueryResponse, bool)
    Set(req models.QueryRequest, resp models.QueryResponse)
}
```

The `CacheInterface` defines the contract for components that cache query responses to improve performance and reduce costs. It includes two methods:

1. **Get**: Retrieves a cached response for a given query request.
   - Parameters:
     - `req models.QueryRequest`: The query request to look up in the cache
   - Returns:
     - `models.QueryResponse`: The cached response, if found
     - `bool`: A boolean indicating whether a cache hit occurred (true) or not (false)

2. **Set**: Stores a response in the cache for future use.
   - Parameters:
     - `req models.QueryRequest`: The query request to use as the cache key
     - `resp models.QueryResponse`: The response to cache

## Usage

These interfaces are used by the API handlers to interact with the router and cache components:

```go
type Handler struct {
    router      RouterInterface
    cache       CacheInterface
    rateLimiter *RateLimiter
}
```

The `Handler` struct in the API package depends on these interfaces rather than concrete implementations, which allows for:

1. **Dependency Injection**: The router and cache components can be injected into the handler, making it more configurable.
2. **Testability**: Mock implementations of these interfaces can be used in tests to isolate the handler logic.
3. **Modularity**: The router and cache components can be replaced with different implementations without changing the API handlers.

## Dependencies

- `context`: Standard Go package for context handling
- `github.com/amorin24/llmproxy/pkg/models`: For data models used in the interfaces

## Integration with Other Components

These interfaces are implemented by other components in the system:

1. **RouterInterface**: Implemented by the `router.Router` struct in the `pkg/router` package.
2. **CacheInterface**: Implemented by the `cache.Cache` struct in the `pkg/cache` package.

Mock implementations of these interfaces are also used in tests to isolate the API handlers for testing.
