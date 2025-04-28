package llm

import (
	"context"
	"encoding/json"
	"errors"
	"io/ioutil"
	"net/http"
	"strings"
	"testing"
	"time"

	myerrors "github.com/amorin24/llmproxy/pkg/errors"
	"github.com/amorin24/llmproxy/pkg/models"
)

func TestGeminiClient_GetModelType(t *testing.T) {
	client := NewGeminiClient()
	if client.GetModelType() != models.Gemini {
		t.Errorf("Expected model type %s, got %s", models.Gemini, client.GetModelType())
	}
}

func TestGeminiClient_Query(t *testing.T) {
	testCases := []struct {
		name        string
		apiKey      string
		statusCode  int
		responseBody string
		expectError bool
		errorType   error
	}{
		{
			name:        "Successful query",
			apiKey:      "test-key",
			statusCode:  http.StatusOK,
			responseBody: `{
				"candidates": [
					{
						"content": {
							"parts": [
								{
									"text": "This is a test response"
								}
							]
						},
						"finishReason": "STOP",
						"tokenCount": {
							"totalTokens": 30
						}
					}
				]
			}`,
			expectError: false,
		},
		{
			name:        "Missing API key",
			apiKey:      "",
			expectError: true,
			errorType:   myerrors.ErrAPIKeyMissing,
		},
		{
			name:        "Rate limit error",
			apiKey:      "test-key",
			statusCode:  http.StatusTooManyRequests,
			responseBody: `{"error": {"message": "Rate limit exceeded", "status": "RESOURCE_EXHAUSTED"}}`,
			expectError: true,
			errorType:   myerrors.ErrRateLimit,
		},
		{
			name:        "Server error",
			apiKey:      "test-key",
			statusCode:  http.StatusInternalServerError,
			responseBody: `{"error": {"message": "Server error", "status": "INTERNAL"}}`,
			expectError: true,
		},
		{
			name:        "Empty response",
			apiKey:      "test-key",
			statusCode:  http.StatusOK,
			responseBody: `{"candidates": []}`,
			expectError: true,
			errorType:   myerrors.ErrEmptyResponse,
		},
		{
			name:        "Invalid JSON response",
			apiKey:      "test-key",
			statusCode:  http.StatusOK,
			responseBody: `{invalid json}`,
			expectError: true,
			errorType:   myerrors.ErrInvalidResponse,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			httpClient := &http.Client{
				Transport: &mockTransport{
					roundTripFunc: func(req *http.Request) (*http.Response, error) {
						if tc.apiKey == "" {
							return nil, errors.New("no API key")
						}
						
						if !strings.Contains(req.URL.String(), "key="+tc.apiKey) {
							t.Errorf("Expected URL to contain API key '%s'", tc.apiKey)
						}
						
						if req.Header.Get("Content-Type") != "application/json" {
							t.Errorf("Expected Content-Type header 'application/json', got '%s'", req.Header.Get("Content-Type"))
						}
						
						if tc.responseBody == `{invalid json}` {
							return &http.Response{
								StatusCode: tc.statusCode,
								Body:       ioutil.NopCloser(strings.NewReader(tc.responseBody)),
							}, nil
						}
						
						return &http.Response{
							StatusCode: tc.statusCode,
							Body:       ioutil.NopCloser(strings.NewReader(tc.responseBody)),
						}, nil
					},
				},
				Timeout: 30 * time.Second,
			}
			
			client := &GeminiClient{
				apiKey: tc.apiKey,
				client: httpClient,
			}
			
			result, err := client.Query(context.Background(), "Test query", "gemini-pro")
			
			if tc.expectError {
				if err == nil {
					t.Errorf("Expected error, got nil")
				}
				
				if tc.errorType != nil {
					var modelErr *myerrors.ModelError
					if errors.As(err, &modelErr) {
						if !errors.Is(modelErr.Unwrap(), tc.errorType) {
							t.Errorf("Expected error type %v, got %v", tc.errorType, modelErr.Unwrap())
						}
					} else {
						t.Errorf("Expected ModelError, got %T", err)
					}
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error, got %v", err)
				}
				
				if result == nil {
					t.Errorf("Expected result, got nil")
				} else {
					var geminiResp GeminiResponse
					json.Unmarshal([]byte(tc.responseBody), &geminiResp)
					
					expectedResponse := geminiResp.Candidates[0].Content.Parts[0].Text
					if result.Response != expectedResponse {
						t.Errorf("Expected response '%s', got '%s'", expectedResponse, result.Response)
					}
					
					expectedTotalTokens := geminiResp.Candidates[0].TokenCount.TotalTokens
					if result.TotalTokens != expectedTotalTokens && expectedTotalTokens > 0 {
						t.Errorf("Expected total tokens %d, got %d", expectedTotalTokens, result.TotalTokens)
					}
				}
			}
		})
	}
}

func TestGeminiClient_CheckAvailability(t *testing.T) {
	testCases := []struct {
		name        string
		apiKey      string
		statusCode  int
		expected    bool
	}{
		{
			name:       "Available",
			apiKey:     "test-key",
			statusCode: http.StatusOK,
			expected:   true,
		},
		{
			name:       "Unavailable",
			apiKey:     "test-key",
			statusCode: http.StatusInternalServerError,
			expected:   false,
		},
		{
			name:       "No API key",
			apiKey:     "",
			expected:   false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			httpClient := &http.Client{
				Transport: &mockTransport{
					roundTripFunc: func(req *http.Request) (*http.Response, error) {
						if tc.apiKey == "" {
							return nil, errors.New("no API key")
						}
						
						return &http.Response{
							StatusCode: tc.statusCode,
							Body:       ioutil.NopCloser(strings.NewReader(`{}`)),
						}, nil
					},
				},
				Timeout: 30 * time.Second,
			}
			
			client := &GeminiClient{
				apiKey: tc.apiKey,
				client: httpClient,
			}
			
			result := client.CheckAvailability()
			
			if result != tc.expected {
				t.Errorf("Expected %v, got %v", tc.expected, result)
			}
		})
	}
}
