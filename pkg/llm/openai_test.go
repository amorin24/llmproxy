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

func TestOpenAIClient_GetModelType(t *testing.T) {
	client := NewOpenAIClient()
	if client.GetModelType() != models.OpenAI {
		t.Errorf("Expected model type %s, got %s", models.OpenAI, client.GetModelType())
	}
}

func TestOpenAIClient_Query(t *testing.T) {
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
				"choices": [
					{
						"message": {
							"content": "This is a test response"
						},
						"finish_reason": "stop"
					}
				],
				"usage": {
					"prompt_tokens": 10,
					"completion_tokens": 20,
					"total_tokens": 30
				}
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
			responseBody: `{"error": {"message": "Rate limit exceeded", "type": "rate_limit_error"}}`,
			expectError: true,
			errorType:   myerrors.ErrRateLimit,
		},
		{
			name:        "Server error",
			apiKey:      "test-key",
			statusCode:  http.StatusInternalServerError,
			responseBody: `{"error": {"message": "Server error", "type": "server_error"}}`,
			expectError: true,
		},
		{
			name:        "Empty response",
			apiKey:      "test-key",
			statusCode:  http.StatusOK,
			responseBody: `{"choices": []}`,
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
						
						if req.Header.Get("Authorization") != "Bearer "+tc.apiKey {
							t.Errorf("Expected Authorization header 'Bearer %s', got '%s'", tc.apiKey, req.Header.Get("Authorization"))
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
			
			client := &OpenAIClient{
				apiKey: tc.apiKey,
				client: httpClient,
			}
			
			result, err := client.Query(context.Background(), "Test query")
			
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
					var openaiResp OpenAIResponse
					json.Unmarshal([]byte(tc.responseBody), &openaiResp)
					
					expectedResponse := openaiResp.Choices[0].Message.Content
					if result.Response != expectedResponse {
						t.Errorf("Expected response '%s', got '%s'", expectedResponse, result.Response)
					}
					
					expectedInputTokens := openaiResp.Usage.PromptTokens
					if result.InputTokens != expectedInputTokens {
						t.Errorf("Expected input tokens %d, got %d", expectedInputTokens, result.InputTokens)
					}
					
					expectedOutputTokens := openaiResp.Usage.CompletionTokens
					if result.OutputTokens != expectedOutputTokens {
						t.Errorf("Expected output tokens %d, got %d", expectedOutputTokens, result.OutputTokens)
					}
					
					expectedTotalTokens := openaiResp.Usage.TotalTokens
					if result.TotalTokens != expectedTotalTokens {
						t.Errorf("Expected total tokens %d, got %d", expectedTotalTokens, result.TotalTokens)
					}
				}
			}
		})
	}
}

func TestOpenAIClient_CheckAvailability(t *testing.T) {
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
			
			client := &OpenAIClient{
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
