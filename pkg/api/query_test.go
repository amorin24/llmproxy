package api

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	myerrors "github.com/amorin24/llmproxy/pkg/errors"
	"github.com/amorin24/llmproxy/pkg/llm"
	"github.com/amorin24/llmproxy/pkg/models"
)

func TestQueryHandlerWithMocks(t *testing.T) {
	testCases := []struct {
		name           string
		requestBody    string
		setupMocks     func(*MockRouter, *MockCache)
		expectedStatus int
		expectedModel  models.ModelType
		expectError    bool
		cancelContext  bool
	}{
		{
			name:        "Successful query",
			requestBody: `{"query":"test query", "model":"openai"}`,
			setupMocks: func(router *MockRouter, cache *MockCache) {
				router.routeRequestFunc = func(ctx context.Context, req models.QueryRequest) (models.ModelType, error) {
					return models.OpenAI, nil
				}
				cache.getFunc = func(req models.QueryRequest) (models.QueryResponse, bool) {
					return models.QueryResponse{}, false // Cache miss
				}
			},
			expectedStatus: http.StatusOK,
			expectedModel:  models.OpenAI,
			expectError:    false,
		},
		{
			name:        "Cache hit",
			requestBody: `{"query":"cached query", "model":"openai"}`,
			setupMocks: func(router *MockRouter, cache *MockCache) {
				cache.getFunc = func(req models.QueryRequest) (models.QueryResponse, bool) {
					if req.Query == "cached query" {
						return models.QueryResponse{
							Response: "Cached response",
							Model:    models.OpenAI,
						}, true // Cache hit
					}
					return models.QueryResponse{}, false
				}
			},
			expectedStatus: http.StatusOK,
			expectedModel:  models.OpenAI,
			expectError:    false,
		},
		{
			name:        "Routing error",
			requestBody: `{"query":"test query"}`,
			setupMocks: func(router *MockRouter, cache *MockCache) {
				router.routeRequestFunc = func(ctx context.Context, req models.QueryRequest) (models.ModelType, error) {
					return "", myerrors.NewUnavailableError("all")
				}
				cache.getFunc = func(req models.QueryRequest) (models.QueryResponse, bool) {
					return models.QueryResponse{}, false // Cache miss
				}
			},
			expectedStatus: http.StatusServiceUnavailable,
			expectError:    true,
		},
		{
			name:        "LLM query error with fallback",
			requestBody: `{"query":"test query", "model":"openai"}`,
			setupMocks: func(router *MockRouter, cache *MockCache) {
				router.routeRequestFunc = func(ctx context.Context, req models.QueryRequest) (models.ModelType, error) {
					return models.OpenAI, nil
				}
				router.fallbackOnErrorFunc = func(ctx context.Context, originalModel models.ModelType, req models.QueryRequest, err error) (models.ModelType, error) {
					return models.Gemini, nil
				}
				cache.getFunc = func(req models.QueryRequest) (models.QueryResponse, bool) {
					return models.QueryResponse{}, false // Cache miss
				}
			},
			expectedStatus: http.StatusOK,
			expectedModel:  models.Gemini,
			expectError:    false,
		},
		{
			name:        "LLM query error with no fallback",
			requestBody: `{"query":"test query", "model":"openai"}`,
			setupMocks: func(router *MockRouter, cache *MockCache) {
				router.routeRequestFunc = func(ctx context.Context, req models.QueryRequest) (models.ModelType, error) {
					return models.OpenAI, nil
				}
				router.fallbackOnErrorFunc = func(ctx context.Context, originalModel models.ModelType, req models.QueryRequest, err error) (models.ModelType, error) {
					return "", myerrors.NewUnavailableError("all")
				}
				cache.getFunc = func(req models.QueryRequest) (models.QueryResponse, bool) {
					return models.QueryResponse{}, false // Cache miss
				}
			},
			expectedStatus: http.StatusInternalServerError,
			expectError:    true,
		},
		{
			name:        "Context canceled",
			requestBody: `{"query":"test query", "model":"openai"}`,
			setupMocks: func(router *MockRouter, cache *MockCache) {
				router.routeRequestFunc = func(ctx context.Context, req models.QueryRequest) (models.ModelType, error) {
					if ctx.Err() != nil {
						return "", ctx.Err()
					}
					return models.OpenAI, nil
				}
				cache.getFunc = func(req models.QueryRequest) (models.QueryResponse, bool) {
					return models.QueryResponse{}, false // Cache miss
				}
			},
			expectedStatus: 499, // Client Closed Request
			expectError:    true,
			cancelContext:  true,
		},
		{
			name:        "Invalid JSON request",
			requestBody: `{"query":"test query", "model":`,
			setupMocks: func(router *MockRouter, cache *MockCache) {},
			expectedStatus: http.StatusBadRequest,
			expectError:    true,
		},
		{
			name:        "Method not allowed",
			requestBody: `{"query":"test query", "model":"openai"}`,
			setupMocks: func(router *MockRouter, cache *MockCache) {},
			expectedStatus: http.StatusMethodNotAllowed,
			expectError:    true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mockRouter := &MockRouter{}
			mockCache := &MockCache{}
			
			tc.setupMocks(mockRouter, mockCache)
			
			originalFactory := llm.Factory
			
			llm.Factory = func(modelType models.ModelType) (llm.Client, error) {
				if modelType == models.OpenAI {
					return &MockLLMClient{
						modelType: models.OpenAI,
						queryFunc: func(ctx context.Context, query string, modelVersion string) (*llm.QueryResult, error) {
							if tc.name == "LLM query error with fallback" || tc.name == "LLM query error with no fallback" {
								return nil, myerrors.NewRateLimitError("openai")
							}
							return &llm.QueryResult{
								Response: "Mock response",
							}, nil
						},
					}, nil
				}
				return &MockLLMClient{
					modelType: modelType,
						queryFunc: func(ctx context.Context, query string, modelVersion string) (*llm.QueryResult, error) {
						return &llm.QueryResult{
							Response: "Fallback response from " + string(modelType),
						}, nil
					},
				}, nil
			}
			
			defer func() {
				llm.Factory = originalFactory
			}()
			
			handler := &Handler{
				router: mockRouter,
				cache:  mockCache,
				rateLimiter: NewRateLimiter(100, 10),
			}
			
			var req *http.Request
			var method string
			
			if tc.name == "Method not allowed" {
				method = http.MethodGet
			} else {
				method = http.MethodPost
			}
			
			req = httptest.NewRequest(method, "/query", bytes.NewBufferString(tc.requestBody))
			w := httptest.NewRecorder()
			
			if tc.cancelContext {
				ctx, cancel := context.WithCancel(req.Context())
				req = req.WithContext(ctx)
				cancel() // Cancel immediately
			}
			
			handler.QueryHandler(w, req)
			
			if w.Code != tc.expectedStatus {
				t.Errorf("Expected status code %d, got %d", tc.expectedStatus, w.Code)
			}
			
			var resp map[string]interface{}
			if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
				t.Fatalf("Error decoding response: %v", err)
			}
			
			if tc.expectError {
				if _, ok := resp["error"]; !ok {
					t.Errorf("Expected error in response, got %+v", resp)
				}
			} else {
				if _, ok := resp["error"]; ok {
					t.Errorf("Did not expect error in response, got %+v", resp)
				}
				
				if model, ok := resp["model"]; ok {
					if model != string(tc.expectedModel) && tc.expectedModel != "" {
						t.Errorf("Expected model %s, got %s", tc.expectedModel, model)
					}
				} else if tc.expectedModel != "" {
					t.Errorf("Expected model in response, got none")
				}
				
				if _, ok := resp["response"]; !ok {
					t.Errorf("Expected response in response, got none")
				}
			}
		})
	}
}

func TestQueryHandlerWithTimeout(t *testing.T) {
	mockRouter := &MockRouter{}
	mockCache := &MockCache{}
	
	mockRouter.routeRequestFunc = func(ctx context.Context, req models.QueryRequest) (models.ModelType, error) {
		return models.OpenAI, nil
	}
	
	mockCache.getFunc = func(req models.QueryRequest) (models.QueryResponse, bool) {
		return models.QueryResponse{}, false // Cache miss
	}
	
	originalFactory := llm.Factory
	
	llm.Factory = func(modelType models.ModelType) (llm.Client, error) {
		return &MockLLMClient{
			modelType: modelType,
			queryFunc: func(ctx context.Context, query string, modelVersion string) (*llm.QueryResult, error) {
				select {
				case <-time.After(200 * time.Millisecond):
					return &llm.QueryResult{
						Response: "Mock response",
					}, nil
				case <-ctx.Done():
					return nil, ctx.Err()
				}
			},
		}, nil
	}
	
	defer func() {
		llm.Factory = originalFactory
	}()
	
	handler := &Handler{
		router: mockRouter,
		cache:  mockCache,
		rateLimiter: NewRateLimiter(100, 10),
	}
	
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	
	req := httptest.NewRequest(http.MethodPost, "/query", bytes.NewBufferString(`{"query":"test query"}`))
	req = req.WithContext(ctx)
	
	w := httptest.NewRecorder()
	
	handler.QueryHandler(w, req)
	
	if w.Code != http.StatusRequestTimeout {
		t.Errorf("Expected status code %d, got %d", http.StatusRequestTimeout, w.Code)
	}
	
	var resp map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("Error decoding response: %v", err)
	}
	
	if _, ok := resp["error"]; !ok {
		t.Errorf("Expected error in response, got %+v", resp)
	}
}
