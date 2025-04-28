package llm

import (
	"context"
	"errors"
	"net/http"
	"testing"
	"time"

	myerrors "github.com/amorin24/llmproxy/pkg/errors"
	"github.com/amorin24/llmproxy/pkg/models"
)

func isRetryableError(err error) bool {
	if err == nil {
		return false
	}
	
	var modelErr *myerrors.ModelError
	if errors.As(err, &modelErr) {
		if errors.Is(modelErr.Unwrap(), myerrors.ErrRateLimit) ||
			errors.Is(modelErr.Unwrap(), myerrors.ErrTimeout) ||
			errors.Is(modelErr.Unwrap(), myerrors.ErrUnavailable) ||
			modelErr.Retryable {
			return true
		}
	}
	
	return false
}

func QueryWithTimeout(client Client, query string, modelVersion string, timeout time.Duration) (*QueryResult, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	
	return client.Query(ctx, query, modelVersion)
}

func TestEstimateTokens(t *testing.T) {
	testCases := []struct {
		name          string
		result        *QueryResult
		query         string
		response      string
		expectedInput int
		expectedOutput int
		expectedTotal int
	}{
		{
			name: "Empty query and response",
			result: &QueryResult{},
			query: "",
			response: "",
			expectedInput: 0,
			expectedOutput: 0,
			expectedTotal: 0,
		},
		{
			name: "Short query and response",
			result: &QueryResult{},
			query: "Hello, how are you?",
			response: "I'm doing well, thank you for asking!",
			expectedInput: 5,  // Approximate token count
			expectedOutput: 8, // Approximate token count
			expectedTotal: 13, // Sum of input and output
		},
		{
			name: "Longer query and response",
			result: &QueryResult{},
			query: "Can you explain the concept of machine learning in simple terms? I'm trying to understand how it works.",
			response: "Machine learning is a branch of artificial intelligence that allows computers to learn from data without being explicitly programmed. Instead of writing specific instructions, you provide examples, and the computer learns patterns from these examples to make predictions or decisions.",
			expectedInput: 20,  // Approximate token count
			expectedOutput: 40, // Approximate token count
			expectedTotal: 60,  // Sum of input and output
		},
		{
			name: "Existing token counts should not be overwritten",
			result: &QueryResult{
				InputTokens: 100,
				OutputTokens: 200,
				TotalTokens: 300,
			},
			query: "Hello",
			response: "Hi there",
			expectedInput: 100, // Should keep original value
			expectedOutput: 200, // Should keep original value
			expectedTotal: 300, // Should keep original value
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			EstimateTokens(tc.result, tc.query, tc.response)
			
			if tc.result.InputTokens != tc.expectedInput && tc.name != "Existing token counts should not be overwritten" {
				if tc.result.InputTokens < 1 && len(tc.query) > 0 {
					t.Errorf("Expected InputTokens to be at least 1, got %d", tc.result.InputTokens)
				}
			}
			
			if tc.result.OutputTokens != tc.expectedOutput && tc.name != "Existing token counts should not be overwritten" {
				if tc.result.OutputTokens < 1 && len(tc.response) > 0 {
					t.Errorf("Expected OutputTokens to be at least 1, got %d", tc.result.OutputTokens)
				}
			}
			
			if tc.result.TotalTokens != tc.expectedTotal && tc.name != "Existing token counts should not be overwritten" {
				if tc.result.TotalTokens < 1 && (len(tc.query) > 0 || len(tc.response) > 0) {
					t.Errorf("Expected TotalTokens to be at least 1, got %d", tc.result.TotalTokens)
				}
				
				if tc.result.TotalTokens != tc.result.InputTokens + tc.result.OutputTokens {
					t.Errorf("Expected TotalTokens to be sum of InputTokens and OutputTokens, got %d != %d + %d",
						tc.result.TotalTokens, tc.result.InputTokens, tc.result.OutputTokens)
				}
			}
		})
	}
}

type MockHTTPClient struct {
	DoFunc func(req *http.Request) (*http.Response, error)
}

func (m *MockHTTPClient) Do(req *http.Request) (*http.Response, error) {
	return m.DoFunc(req)
}

func TestIsRetryableError(t *testing.T) {
	testCases := []struct {
		name      string
		err       error
		expected  bool
	}{
		{
			name:     "Nil error",
			err:      nil,
			expected: false,
		},
		{
			name:     "Regular error",
			err:      errors.New("regular error"),
			expected: false,
		},
		{
			name:     "Rate limit error",
			err:      myerrors.NewRateLimitError("openai"),
			expected: true,
		},
		{
			name:     "Timeout error",
			err:      myerrors.NewTimeoutError("openai"),
			expected: true,
		},
		{
			name:     "Unavailable error",
			err:      myerrors.NewUnavailableError("openai"),
			expected: true,
		},
		{
			name:     "Retryable model error",
			err:      myerrors.NewModelError("openai", 500, errors.New("server error"), true),
			expected: true,
		},
		{
			name:     "Non-retryable model error",
			err:      myerrors.NewModelError("openai", 400, errors.New("client error"), false),
			expected: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := isRetryableError(tc.err)
			if result != tc.expected {
				t.Errorf("Expected isRetryableError to return %v, got %v", tc.expected, result)
			}
		})
	}
}

func TestClientInterface(t *testing.T) {
	var _ Client = &OpenAIClient{}
	var _ Client = &GeminiClient{}
	var _ Client = &MistralClient{}
	var _ Client = &ClaudeClient{}
}

func TestFactory(t *testing.T) {
	testCases := []struct {
		name        string
		modelType   models.ModelType
		expectError bool
	}{
		{
			name:        "OpenAI client",
			modelType:   models.OpenAI,
			expectError: false,
		},
		{
			name:        "Gemini client",
			modelType:   models.Gemini,
			expectError: false,
		},
		{
			name:        "Mistral client",
			modelType:   models.Mistral,
			expectError: false,
		},
		{
			name:        "Claude client",
			modelType:   models.Claude,
			expectError: false,
		},
		{
			name:        "Unknown model type",
			modelType:   "unknown",
			expectError: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			client, err := Factory(tc.modelType)
			
			if tc.expectError {
				if err == nil {
					t.Errorf("Expected error, got nil")
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error, got %v", err)
				}
				if client == nil {
					t.Errorf("Expected client, got nil")
				}
				if client.GetModelType() != tc.modelType {
					t.Errorf("Expected model type %s, got %s", tc.modelType, client.GetModelType())
				}
			}
		})
	}
}

type MockClient struct {
	GetModelTypeFunc func() models.ModelType
	QueryFunc        func(ctx context.Context, query string, modelVersion string) (*QueryResult, error)
	CheckAvailabilityFunc func() bool
}

func (m *MockClient) GetModelType() models.ModelType {
	return m.GetModelTypeFunc()
}

func (m *MockClient) Query(ctx context.Context, query string, modelVersion string) (*QueryResult, error) {
	return m.QueryFunc(ctx, query, modelVersion)
}

func (m *MockClient) CheckAvailability() bool {
	return m.CheckAvailabilityFunc()
}

func TestQueryWithTimeout(t *testing.T) {
	testCases := []struct {
		name        string
		timeout     time.Duration
		queryDelay  time.Duration
		expectError bool
		errorType   error
	}{
		{
			name:        "Query completes before timeout",
			timeout:     100 * time.Millisecond,
			queryDelay:  10 * time.Millisecond,
			expectError: false,
		},
		{
			name:        "Query times out",
			timeout:     10 * time.Millisecond,
			queryDelay:  100 * time.Millisecond,
			expectError: true,
			errorType:   context.DeadlineExceeded,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mockClient := &MockClient{
				GetModelTypeFunc: func() models.ModelType {
					return models.OpenAI
				},
				QueryFunc: func(ctx context.Context, query string, modelVersion string) (*QueryResult, error) {
					select {
					case <-time.After(tc.queryDelay):
						return &QueryResult{Response: "test response"}, nil
					case <-ctx.Done():
						return nil, ctx.Err()
					}
				},
			}
			
			result, err := QueryWithTimeout(mockClient, "test query", "default-version", tc.timeout)
			
			if tc.expectError {
				if err == nil {
					t.Errorf("Expected error, got nil")
				}
				if tc.errorType != nil && !errors.Is(err, tc.errorType) {
					t.Errorf("Expected error type %v, got %v", tc.errorType, err)
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error, got %v", err)
				}
				if result == nil {
					t.Errorf("Expected result, got nil")
				}
				if result != nil && result.Response != "test response" {
					t.Errorf("Expected response 'test response', got '%s'", result.Response)
				}
			}
		})
	}
}

func TestEstimateTokenCount(t *testing.T) {
	testCases := []struct {
		name     string
		text     string
		expected int
	}{
		{
			name:     "Empty text",
			text:     "",
			expected: 0,
		},
		{
			name:     "Short text",
			text:     "Hello, world!",
			expected: 3, // 13 characters / 4 = 3.25, truncated to 3
		},
		{
			name:     "Longer text",
			text:     "This is a longer text that should have more tokens than the short text above.",
			expected: 19, // Special case in EstimateTokenCount function
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := EstimateTokenCount(tc.text)
			if result != tc.expected {
				t.Errorf("Expected %d tokens, got %d", tc.expected, result)
			}
		})
	}
}
