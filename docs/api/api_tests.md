# API Tests Documentation

## Overview

The `pkg/api/api_test.go` file contains unit tests for the API handlers in the LLM Proxy system. These tests verify the correct behavior of the API endpoints, input validation, security headers, and rate limiting functionality.

## Test Functions

### TestAPIStatusHandler

Tests the `/status` endpoint that provides information about the availability of different LLM models.

**Purpose:**
- Verifies that the status endpoint returns a 200 OK response
- Ensures the response can be properly decoded into a StatusResponse struct

**Implementation Details:**
- Creates a new handler using `NewHandler()`
- Makes a GET request to the `/status` endpoint
- Verifies the status code is 200 OK
- Attempts to decode the response into a StatusResponse struct

### TestQueryHandlerValidation

Tests the input validation for the `/query` endpoint with various invalid inputs.

**Purpose:**
- Ensures the API properly validates request inputs
- Verifies appropriate error responses for invalid inputs

**Test Cases:**
1. **Empty request**: Tests that an empty JSON object is rejected
2. **Missing query**: Tests that a request with a model but no query is rejected
3. **Query too long**: Tests that a request with an excessively long query is rejected
4. **Invalid model**: Tests that a request with an invalid model name is rejected
5. **Invalid task type**: Tests that a request with an invalid task type is rejected

**Implementation Details:**
- Uses table-driven tests to run multiple test cases
- For each case, creates a POST request with the specified JSON body
- Verifies the status code matches the expected value (400 Bad Request)
- Confirms that an error message is included in the response

### TestAPIHealthHandler

Tests the `/health` endpoint that provides system health information.

**Purpose:**
- Verifies that the health endpoint returns a 200 OK response
- Ensures the response can be properly decoded

**Implementation Details:**
- Creates a new handler using `NewHandler()`
- Makes a GET request to the `/health` endpoint
- Verifies the status code is 200 OK
- Attempts to decode the response into a map

### TestAPISecurityHeaders

Tests that security headers are properly set in API responses.

**Purpose:**
- Ensures that all required security headers are present in API responses
- Verifies that the headers have the correct values

**Headers Tested:**
- Content-Type: Should be "application/json"
- X-Content-Type-Options: Should be "nosniff"
- X-Frame-Options: Should be "DENY"
- X-XSS-Protection: Should be "1; mode=block"
- Cache-Control: Should contain "no-store", "no-cache", and "must-revalidate"

**Implementation Details:**
- Makes a request to the `/status` endpoint
- Checks each expected header against the actual value in the response
- For Cache-Control, verifies that it contains all required directives

### TestRateLimiting

Tests that rate limiting is properly applied to API endpoints.

**Purpose:**
- Verifies that the API enforces rate limits
- Ensures that requests beyond the rate limit receive a 429 Too Many Requests response

**Implementation Details:**
- Makes multiple requests to the `/health` endpoint in a loop
- Expects the first several requests to succeed with 200 OK
- Expects subsequent requests to be rate-limited with 429 Too Many Requests
- Verifies that rate-limited responses include an error message

## Integration with the API Package

These tests verify the functionality of the handlers defined in the API package. They ensure that:

1. The API endpoints respond correctly to valid requests
2. Input validation properly rejects invalid requests
3. Security headers are correctly set in responses
4. Rate limiting is properly enforced

The tests use the standard Go testing package and the httptest package to create mock HTTP requests and responses without requiring a running server.

## Dependencies

- `net/http`: Standard Go HTTP package
- `net/http/httptest`: Package for HTTP testing
- `testing`: Standard Go testing package
- `encoding/json`: For JSON encoding/decoding
- `bytes`: For working with byte slices
- `github.com/amorin24/llmproxy/pkg/models`: For the StatusResponse struct

## Usage

Run these tests using the Go test command:

```bash
go test -v github.com/amorin24/llmproxy/pkg/api
```

These tests are also run as part of the continuous integration process to ensure that changes to the API handlers do not break existing functionality.
