# API Handlers Documentation

## Overview

The `pkg/api/handlers.go` file is a core component of the LLM Proxy system that implements the HTTP handlers for processing API requests. It handles routing requests to appropriate LLM providers, manages rate limiting, implements caching, and provides comprehensive error handling and security features.

## Key Components

### RateLimiter

A token bucket-based rate limiter implementation that controls the rate of requests to the API.

**Features:**
- Token bucket algorithm with configurable refill rate and burst capacity
- Per-client (IP-based) rate limiting
- Automatic token refill based on elapsed time
- Support for custom client allowance functions (for testing)

**Methods:**
- `NewRateLimiter(requestsPerMinute, burst int)`: Creates a new rate limiter with specified limits
- `Allow()`: Checks if a request is allowed based on available tokens
- `AllowClient(clientID string)`: Checks if a request from a specific client is allowed
- `SetAllowClientFunc(fn func(clientID string) bool)`: Sets a custom function for client allowance (for testing)

### Handler

The main struct that processes API requests and manages dependencies.

**Fields:**
- `router`: Interface for routing requests to appropriate LLM providers
- `cache`: Interface for caching query responses
- `rateLimiter`: Rate limiter for controlling request rates

**Methods:**
- `NewHandler()`: Creates a new handler with default configuration
- `QueryHandler(w http.ResponseWriter, r *http.Request)`: Handles LLM query requests
- `StatusHandler(w http.ResponseWriter, r *http.Request)`: Provides status information about available LLM models
- `HealthHandler(w http.ResponseWriter, r *http.Request)`: Provides system health information

### QueryHandler

The main handler for processing LLM queries. It implements a comprehensive workflow:

1. **Request Validation**: Validates the request method, body size, and query parameters
2. **Rate Limiting**: Enforces rate limits based on client IP
3. **Cache Checking**: Checks if the response is already cached
4. **Request Routing**: Routes the request to the appropriate LLM provider
5. **Query Processing**: Sends the query to the selected LLM provider
6. **Fallback Handling**: Attempts to use alternative providers if the primary one fails
7. **Response Formatting**: Formats the response with appropriate headers and content
8. **Caching**: Caches the response for future requests
9. **Logging**: Logs request and response details for monitoring and debugging

**Error Handling:**
- Context cancellation (client disconnects)
- Request timeouts
- Rate limiting errors
- Model-specific errors (API key missing, service unavailable, etc.)
- Request validation errors

### StatusHandler

Provides information about the availability of different LLM models.

**Features:**
- Returns a status object with availability information for each LLM provider
- Enforces rate limits to prevent abuse
- Sets appropriate security headers

### HealthHandler

Provides system health information for monitoring purposes.

**Features:**
- Returns a simple status object indicating system health
- Enforces rate limits to prevent abuse
- Sets appropriate security headers

### Utility Functions

- `getClientIP(r *http.Request)`: Extracts the client IP from the request
- `validateQueryRequest(req models.QueryRequest)`: Validates query request parameters
- `sanitizeQuery(query string)`: Sanitizes the query string
- `sendJSONResponse(w http.ResponseWriter, data interface{}, statusCode int)`: Sends a JSON response with appropriate headers
- `handleError(w http.ResponseWriter, message string, statusCode int)`: Handles and formats error responses
- `getEnvAsInt(key string, defaultValue int)`: Gets an environment variable as an integer with a default value
- `min(a, b float64)`: Returns the minimum of two float values

## Security Features

The handlers implement several security best practices:

1. **Rate Limiting**: Prevents abuse through configurable rate limits
2. **Request Size Limits**: Prevents denial-of-service attacks through large requests
3. **Input Validation**: Validates all input parameters to prevent injection attacks
4. **Security Headers**: Sets appropriate security headers in all responses:
   - Content-Type: Specifies the content type to prevent content sniffing
   - X-Content-Type-Options: Prevents content type sniffing
   - X-Frame-Options: Prevents clickjacking attacks
   - X-XSS-Protection: Enables browser XSS protection
   - Content-Security-Policy: Restricts resource loading
   - Referrer-Policy: Controls referrer information
   - Cache-Control: Prevents caching of sensitive information
   - Strict-Transport-Security: Enforces HTTPS

## Error Handling

The handlers implement comprehensive error handling:

1. **Client Errors**: Invalid requests, rate limiting, etc.
2. **Server Errors**: Internal errors, service unavailability, etc.
3. **Context Errors**: Request cancellation, timeouts, etc.
4. **Model-Specific Errors**: API key missing, rate limiting by provider, etc.

All errors are properly logged and formatted as JSON responses with appropriate status codes.

## Integration with Other Components

The handlers integrate with several other components of the LLM Proxy system:

1. **Router**: Uses the router to determine which LLM provider to use for a query
2. **Cache**: Caches responses to improve performance and reduce costs
3. **LLM Clients**: Communicates with different LLM providers through their respective clients
4. **Logging**: Logs request and response details for monitoring and debugging
5. **Error Handling**: Uses custom error types for specific error scenarios

## Configuration

The handlers use several configuration parameters:

1. **Rate Limits**: Configurable through environment variables
   - RATE_LIMIT: Requests per minute (default: 60)
   - RATE_LIMIT_BURST: Burst capacity (default: 10)
2. **Request Limits**:
   - Maximum request body size: 1MB
   - Maximum query length: 32,000 characters
3. **Timeout**: Default timeout for LLM queries (30 seconds)

## Usage

The handlers are registered with the HTTP router in the main application:

```go
r := mux.NewRouter()
handler := api.NewHandler()
r.HandleFunc("/api/query", handler.QueryHandler).Methods("POST")
r.HandleFunc("/api/status", handler.StatusHandler).Methods("GET")
r.HandleFunc("/api/health", handler.HealthHandler).Methods("GET")
```

## Dependencies

- `net/http`: Standard Go HTTP package
- `encoding/json`: For JSON encoding/decoding
- `context`: For request context handling
- `time`: For timeouts and timestamps
- `github.com/amorin24/llmproxy/pkg/cache`: For response caching
- `github.com/amorin24/llmproxy/pkg/errors`: For custom error types
- `github.com/amorin24/llmproxy/pkg/llm`: For LLM client interfaces
- `github.com/amorin24/llmproxy/pkg/logging`: For logging
- `github.com/amorin24/llmproxy/pkg/models`: For data models
- `github.com/amorin24/llmproxy/pkg/router`: For request routing
- `github.com/google/uuid`: For generating request IDs
- `github.com/sirupsen/logrus`: For structured logging
