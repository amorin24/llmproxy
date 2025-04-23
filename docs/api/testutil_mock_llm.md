# TestUtil Mock LLM Documentation

## Overview

The `pkg/api/testutil/mock_llm.go` file provides a mock implementation of the `llm.Client` interface for testing purposes. This mock allows tests to simulate the behavior of LLM clients without making actual API calls to external services, enabling controlled and reproducible testing of components that depend on LLM clients.

## Components

### MockLLMClient

```go
type MockLLMClient struct {
    ModelType  models.ModelType
    Available  bool
    QueryFunc  func(ctx context.Context, query string) (*llm.QueryResult, error)
    QueryError error
}
```

The `MockLLMClient` struct implements the `llm.Client` interface and provides configurable behavior for testing. It includes the following fields:

- **ModelType**: The type of LLM model this client represents (e.g., OpenAI, Gemini, Mistral, Claude).
- **Available**: A boolean indicating whether the model is available.
- **QueryFunc**: An optional function that provides custom behavior for the `Query` method.
- **QueryError**: An optional error to be returned by the `Query` method.

## Methods

### Query

```go
func (m *MockLLMClient) Query(ctx context.Context, query string) (*llm.QueryResult, error)
```

Simulates sending a query to an LLM provider and returns a result or error. The behavior of this method can be configured in several ways:

1. If `QueryFunc` is provided, it uses that function to determine the response.
2. If `QueryError` is provided, it returns that error.
3. Otherwise, it returns a default mock response with token counts based on the query length.

The default response includes:
- A response string that includes the original query
- A status code of 200
- Token counts calculated based on the query length
- Both the deprecated `NumTokens` field and the newer `TotalTokens` field for backward compatibility

### CheckAvailability

```go
func (m *MockLLMClient) CheckAvailability() bool
```

Returns the value of the `Available` field, indicating whether the mock LLM provider is available. This allows tests to simulate scenarios where certain models are unavailable.

### GetModelType

```go
func (m *MockLLMClient) GetModelType() models.ModelType
```

Returns the value of the `ModelType` field, indicating the type of LLM model this client represents. This allows tests to verify that the correct model type is being used.

## Usage in Tests

The `MockLLMClient` is used in tests to simulate various scenarios:

1. **Successful queries**:
   ```go
   mockClient := &MockLLMClient{
       ModelType: models.OpenAI,
       Available: true,
   }
   result, err := mockClient.Query(ctx, "test query")
   // result contains a mock response
   ```

2. **Error scenarios**:
   ```go
   mockClient := &MockLLMClient{
       ModelType:  models.OpenAI,
       Available:  true,
       QueryError: myerrors.NewRateLimitError("openai"),
   }
   result, err := mockClient.Query(ctx, "test query")
   // err contains the specified error
   ```

3. **Custom behavior**:
   ```go
   mockClient := &MockLLMClient{
       ModelType: models.OpenAI,
       Available: true,
       QueryFunc: func(ctx context.Context, query string) (*llm.QueryResult, error) {
           select {
           case <-time.After(200 * time.Millisecond):
               return &llm.QueryResult{
                   Response: "Custom response",
               }, nil
           case <-ctx.Done():
               return nil, ctx.Err()
           }
       },
   }
   // This client will wait for 200ms or until the context is canceled
   ```

## Dependencies

- `context`: Standard Go package for context handling
- `github.com/amorin24/llmproxy/pkg/llm`: For the LLM client interface and QueryResult struct
- `github.com/amorin24/llmproxy/pkg/models`: For the ModelType enum

## Integration with Test Suite

This mock implementation is used extensively in the API handler tests to simulate various LLM client behaviors without making actual API calls. It allows tests to be fast, reliable, and independent of external services.

The `MockLLMClient` is typically used in conjunction with other mocks from the testutil package, such as `MockRouter` and `MockCache`, to provide a complete testing environment for the API handlers.
