# TestUtil Mock Rate Limiter Documentation

## Overview

The `pkg/api/testutil/mock_rate_limiter.go` file provides a mock implementation of a rate limiter for testing purposes. This mock allows tests to simulate rate limiting behavior without implementing the full rate limiting logic, enabling controlled and reproducible testing of components that depend on rate limiting.

## Components

### MockRateLimiter

```go
type MockRateLimiter struct {
    allowClientFunc func(clientID string) bool
}
```

The `MockRateLimiter` struct implements a simple rate limiter interface for testing. It contains a single field:

- **allowClientFunc**: A function that determines whether a client with a given ID is allowed to make a request.

## Functions and Methods

### NewMockRateLimiter

```go
func NewMockRateLimiter() *MockRateLimiter
```

Creates a new instance of `MockRateLimiter` with a default behavior that allows all requests. This provides a convenient way to create a mock rate limiter that doesn't restrict any requests by default.

**Returns:**
- A pointer to a new `MockRateLimiter` instance with the default behavior.

### AllowClient

```go
func (m *MockRateLimiter) AllowClient(clientID string) bool
```

Determines whether a client with the given ID is allowed to make a request. This method simply delegates to the `allowClientFunc` field, which can be customized to implement different rate limiting behaviors.

**Parameters:**
- **clientID**: A string identifying the client making the request.

**Returns:**
- A boolean indicating whether the client is allowed to make a request.

### SetAllowClientFunc

```go
func (m *MockRateLimiter) SetAllowClientFunc(fn func(clientID string) bool)
```

Sets a custom function for determining whether a client is allowed to make a request. This allows tests to configure the rate limiter's behavior for specific test scenarios.

**Parameters:**
- **fn**: A function that takes a client ID and returns a boolean indicating whether the client is allowed to make a request.

## Usage in Tests

The `MockRateLimiter` is used in tests to simulate various rate limiting scenarios:

1. **Allow all requests**:
   ```go
   mockRateLimiter := testutil.NewMockRateLimiter()
   // All requests will be allowed
   ```

2. **Deny all requests**:
   ```go
   mockRateLimiter := testutil.NewMockRateLimiter()
   mockRateLimiter.SetAllowClientFunc(func(clientID string) bool {
       return false
   })
   // All requests will be denied
   ```

3. **Allow specific clients**:
   ```go
   mockRateLimiter := testutil.NewMockRateLimiter()
   allowedClients := map[string]bool{"client1": true, "client2": true}
   mockRateLimiter.SetAllowClientFunc(func(clientID string) bool {
       return allowedClients[clientID]
   })
   // Only requests from client1 and client2 will be allowed
   ```

4. **Implement custom rate limiting logic**:
   ```go
   mockRateLimiter := testutil.NewMockRateLimiter()
   requestCounts := make(map[string]int)
   mockRateLimiter.SetAllowClientFunc(func(clientID string) bool {
       requestCounts[clientID]++
       return requestCounts[clientID] <= 5 // Allow only 5 requests per client
   })
   // Each client will be allowed to make up to 5 requests
   ```

## Integration with Test Suite

This mock implementation is used in the API middleware tests to simulate rate limiting without implementing the full token bucket algorithm. It allows tests to verify that the rate limiting middleware correctly rejects requests when the rate limiter denies them.

The `MockRateLimiter` is typically used in conjunction with other mocks from the testutil package to provide a complete testing environment for the API handlers and middleware.
