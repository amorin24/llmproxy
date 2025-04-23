# TestUtil Utilities Documentation

## Overview

The `pkg/api/testutil/testutil.go` file provides a collection of utility functions for testing HTTP handlers in the LLM Proxy system. These functions simplify common testing tasks such as creating HTTP requests, performing requests, decoding responses, and asserting various aspects of the response. By using these utilities, tests can be more concise, readable, and focused on the specific behavior being tested rather than boilerplate HTTP testing code.

## Functions

### CreateTestRequest

```go
func CreateTestRequest(t *testing.T, method, path string, body interface{}) *http.Request
```

Creates an HTTP request with the specified method, path, and body. If a body is provided, it is marshaled to JSON and set as the request body with the appropriate Content-Type header.

**Parameters:**
- **t**: The testing.T instance for reporting test failures.
- **method**: The HTTP method (e.g., "GET", "POST").
- **path**: The request path.
- **body**: An optional interface{} that will be marshaled to JSON and used as the request body.

**Returns:**
- An *http.Request instance configured with the specified parameters.

**Example:**
```go
req := testutil.CreateTestRequest(t, "POST", "/api/query", models.QueryRequest{
    Query: "test query",
    Model: models.OpenAI,
})
```

### PerformRequest

```go
func PerformRequest(t *testing.T, handler http.Handler, req *http.Request) *httptest.ResponseRecorder
```

Performs an HTTP request using the provided handler and returns a response recorder containing the response.

**Parameters:**
- **t**: The testing.T instance for reporting test failures.
- **handler**: The http.Handler to process the request.
- **req**: The http.Request to perform.

**Returns:**
- A *httptest.ResponseRecorder containing the response.

**Example:**
```go
rr := testutil.PerformRequest(t, handler, req)
```

### DecodeResponse

```go
func DecodeResponse(t *testing.T, rr *httptest.ResponseRecorder, v interface{})
```

Decodes the JSON response body into the provided interface.

**Parameters:**
- **t**: The testing.T instance for reporting test failures.
- **rr**: The response recorder containing the response.
- **v**: A pointer to the interface{} where the decoded response will be stored.

**Example:**
```go
var resp models.QueryResponse
testutil.DecodeResponse(t, rr, &resp)
```

### AssertStatusCode

```go
func AssertStatusCode(t *testing.T, rr *httptest.ResponseRecorder, expected int)
```

Asserts that the response has the expected status code.

**Parameters:**
- **t**: The testing.T instance for reporting test failures.
- **rr**: The response recorder containing the response.
- **expected**: The expected status code.

**Example:**
```go
testutil.AssertStatusCode(t, rr, http.StatusOK)
```

### AssertHeader

```go
func AssertHeader(t *testing.T, rr *httptest.ResponseRecorder, header, expected string)
```

Asserts that the response has the expected header value.

**Parameters:**
- **t**: The testing.T instance for reporting test failures.
- **rr**: The response recorder containing the response.
- **header**: The header name.
- **expected**: The expected header value.

**Example:**
```go
testutil.AssertHeader(t, rr, "Content-Type", "application/json")
```

### AssertHeaderContains

```go
func AssertHeaderContains(t *testing.T, rr *httptest.ResponseRecorder, header, expected string)
```

Asserts that the response header contains the expected substring.

**Parameters:**
- **t**: The testing.T instance for reporting test failures.
- **rr**: The response recorder containing the response.
- **header**: The header name.
- **expected**: The expected substring in the header value.

**Example:**
```go
testutil.AssertHeaderContains(t, rr, "Cache-Control", "no-cache")
```

## Usage in Tests

These utility functions are used extensively in the API handler tests to simplify the testing code and make it more readable. Here's an example of how they might be used together:

```go
func TestQueryHandler(t *testing.T) {
    // Create the handler
    handler := &api.Handler{
        router:      mockRouter,
        cache:       mockCache,
        rateLimiter: api.NewRateLimiter(60, 10),
    }
    
    // Create a request
    req := testutil.CreateTestRequest(t, "POST", "/api/query", models.QueryRequest{
        Query: "test query",
        Model: models.OpenAI,
    })
    
    // Perform the request
    rr := testutil.PerformRequest(t, http.HandlerFunc(handler.QueryHandler), req)
    
    // Assert the status code
    testutil.AssertStatusCode(t, rr, http.StatusOK)
    
    // Assert the content type
    testutil.AssertHeader(t, rr, "Content-Type", "application/json")
    
    // Decode the response
    var resp models.QueryResponse
    testutil.DecodeResponse(t, rr, &resp)
    
    // Assert the response properties
    if resp.Response == "" {
        t.Errorf("Expected non-empty response")
    }
}
```

## Dependencies

- `bytes`: For working with byte slices
- `encoding/json`: For JSON encoding/decoding
- `io`: For I/O operations
- `net/http`: Standard Go HTTP package
- `net/http/httptest`: Package for HTTP testing
- `testing`: Standard Go testing package

## Integration with Test Suite

These utility functions are used throughout the API handler tests to provide a consistent and simplified approach to HTTP testing. They help ensure that tests are focused on the specific behavior being tested rather than the mechanics of HTTP testing.
