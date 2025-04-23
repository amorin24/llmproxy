# API Middleware Documentation

## Overview

The `pkg/api/middleware.go` file implements several HTTP middleware functions for the LLM Proxy system. These middleware functions provide cross-cutting concerns such as security, logging, CORS support, and rate limiting. They are designed to be applied to HTTP handlers to enhance their functionality without modifying the handlers themselves.

## Middleware Functions

### SecurityHeadersMiddleware

```go
func SecurityHeadersMiddleware(next http.Handler) http.Handler
```

This middleware adds security headers to HTTP responses to protect against common web vulnerabilities.

**Headers Set:**
- **Content-Security-Policy**: Restricts the sources from which content can be loaded, helping to prevent cross-site scripting (XSS) attacks.
- **X-Content-Type-Options**: Prevents browsers from MIME-sniffing a response away from the declared content type, reducing the risk of drive-by downloads.
- **X-Frame-Options**: Prevents the page from being displayed in an iframe, protecting against clickjacking attacks.
- **X-XSS-Protection**: Enables the browser's built-in XSS protection.
- **Referrer-Policy**: Controls how much referrer information is included with requests.
- **Strict-Transport-Security**: Enforces the use of HTTPS, protecting against protocol downgrade attacks and cookie hijacking.
- **Cache-Control**: Prevents caching of sensitive information.

**Usage:**
```go
router.Use(SecurityHeadersMiddleware)
```

### LoggingMiddleware

```go
func LoggingMiddleware(next http.Handler) http.Handler
```

This middleware logs information about HTTP requests and responses for monitoring and debugging purposes.

**Logged Information:**
- HTTP method
- Request path
- Response status code
- Request duration (in milliseconds)
- User agent
- Client IP address

The middleware uses a custom `responseWriter` to capture the status code of the response.

**Usage:**
```go
router.Use(LoggingMiddleware)
```

### CORSMiddleware

```go
func CORSMiddleware(next http.Handler) http.Handler
```

This middleware handles Cross-Origin Resource Sharing (CORS) to allow requests from different origins. It adds the necessary headers to responses and handles preflight requests.

**Headers Set:**
- **Access-Control-Allow-Origin**: Specifies which origins are allowed to access the resource.
- **Access-Control-Allow-Methods**: Specifies which HTTP methods are allowed.
- **Access-Control-Allow-Headers**: Specifies which headers are allowed in the actual request.

The middleware also handles OPTIONS preflight requests by returning a 200 OK response without passing the request to the next handler.

**Usage:**
```go
router.Use(CORSMiddleware)
```

### RateLimitMiddleware

```go
func RateLimitMiddleware(next http.Handler, rateLimiter *RateLimiter) http.Handler
```

This middleware enforces rate limits on API requests based on client IP address. It uses the `RateLimiter` struct defined in the API package to track and limit request rates.

**Parameters:**
- **next**: The next handler in the chain.
- **rateLimiter**: A pointer to a `RateLimiter` instance that tracks request rates.

If a request exceeds the rate limit, the middleware returns a 429 Too Many Requests response with an error message.

**Usage:**
```go
rateLimiter := NewRateLimiter(60, 10) // 60 requests per minute, burst of 10
router.Use(func(next http.Handler) http.Handler {
    return RateLimitMiddleware(next, rateLimiter)
})
```

## Helper Types

### responseWriter

```go
type responseWriter struct {
    http.ResponseWriter
    statusCode int
}
```

This struct wraps the standard `http.ResponseWriter` to capture the status code of the response. It is used by the `LoggingMiddleware` to log the status code.

**Methods:**
- **WriteHeader(code int)**: Overrides the standard `WriteHeader` method to capture the status code before passing it to the wrapped `ResponseWriter`.

## Integration with Other Components

These middleware functions are typically applied to the HTTP router in the main application:

```go
r := mux.NewRouter()
rateLimiter := api.NewRateLimiter(60, 10)

// Apply middleware
r.Use(api.SecurityHeadersMiddleware)
r.Use(api.LoggingMiddleware)
r.Use(api.CORSMiddleware)
r.Use(func(next http.Handler) http.Handler {
    return api.RateLimitMiddleware(next, rateLimiter)
})

// Register handlers
r.HandleFunc("/api/query", handler.QueryHandler).Methods("POST")
r.HandleFunc("/api/status", handler.StatusHandler).Methods("GET")
```

## Dependencies

- `net/http`: Standard Go HTTP package
- `time`: Standard Go package for time-related operations
- `github.com/sirupsen/logrus`: External package for structured logging

## Security Considerations

The middleware functions implement several security best practices:

1. **Security Headers**: Protect against common web vulnerabilities such as XSS, clickjacking, and content type sniffing.
2. **Rate Limiting**: Prevent abuse and denial-of-service attacks by limiting request rates.
3. **CORS**: Control which origins can access the API, reducing the risk of cross-site request forgery (CSRF) attacks.

These security measures are essential for protecting the API and its users from various threats.
