package api

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	myerrors "github.com/amorin24/llmproxy/pkg/errors"
	"github.com/amorin24/llmproxy/pkg/llm"
	"github.com/amorin24/llmproxy/pkg/models"
)

func mockLLMFactory(modelType models.ModelType) (llm.Client, error) {
	return &MockLLMClient{modelType: modelType}, nil
}

func TestRateLimiter(t *testing.T) {
	t.Run("Allow within limit", func(t *testing.T) {
		rl := NewRateLimiter(60, 10)
		for i := 0; i < 10; i++ {
			if !rl.Allow() {
				t.Errorf("Expected to allow request %d within burst limit", i)
			}
		}
	})

	t.Run("Deny after limit", func(t *testing.T) {
		rl := NewRateLimiter(60, 5)
		for i := 0; i < 5; i++ {
			rl.Allow()
		}
		if rl.Allow() {
			t.Errorf("Expected to deny request after burst limit")
		}
	})

	t.Run("Token refill", func(t *testing.T) {
		rl := NewRateLimiter(60, 1)
		if !rl.Allow() {
			t.Errorf("Expected to allow first request")
		}
		if rl.Allow() {
			t.Errorf("Expected to deny second request")
		}

		rl.lastRefill = time.Now().Add(-2 * time.Second)
		if !rl.Allow() {
			t.Errorf("Expected to allow request after token refill")
		}
	})

	t.Run("Client-specific rate limiting", func(t *testing.T) {
		rl := NewRateLimiter(60, 2)
		
		if !rl.AllowClient("client1") {
			t.Errorf("Expected to allow first request for client1")
		}
		if !rl.AllowClient("client1") {
			t.Errorf("Expected to allow second request for client1")
		}
		if rl.AllowClient("client1") {
			t.Errorf("Expected to deny third request for client1")
		}
		
		if !rl.AllowClient("client2") {
			t.Errorf("Expected to allow first request for client2")
		}
	})
}

func TestHelperFunctions(t *testing.T) {
	t.Run("validateQueryRequest valid", func(t *testing.T) {
		req := models.QueryRequest{
			Query:    "Test query",
			Model:    models.OpenAI,
			TaskType: models.TextGeneration,
		}
		
		err := validateQueryRequest(req)
		if err != nil {
			t.Errorf("Expected no error for valid request, got: %v", err)
		}
	})
	
	t.Run("validateQueryRequest empty query", func(t *testing.T) {
		req := models.QueryRequest{
			Query:    "",
			Model:    models.OpenAI,
			TaskType: models.TextGeneration,
		}
		
		err := validateQueryRequest(req)
		if err == nil {
			t.Errorf("Expected error for empty query")
		}
	})
	
	t.Run("validateQueryRequest query too long", func(t *testing.T) {
		req := models.QueryRequest{
			Query:    strings.Repeat("a", maxQueryLength+1),
			Model:    models.OpenAI,
			TaskType: models.TextGeneration,
		}
		
		err := validateQueryRequest(req)
		if err == nil {
			t.Errorf("Expected error for query too long")
		}
	})
	
	t.Run("validateQueryRequest invalid model", func(t *testing.T) {
		req := models.QueryRequest{
			Query:    "Test query",
			Model:    "invalid-model",
			TaskType: models.TextGeneration,
		}
		
		err := validateQueryRequest(req)
		if err == nil {
			t.Errorf("Expected error for invalid model")
		}
	})
	
	t.Run("validateQueryRequest invalid task type", func(t *testing.T) {
		req := models.QueryRequest{
			Query:    "Test query",
			Model:    models.OpenAI,
			TaskType: "invalid-task",
		}
		
		err := validateQueryRequest(req)
		if err == nil {
			t.Errorf("Expected error for invalid task type")
		}
	})
	
	t.Run("sanitizeQuery", func(t *testing.T) {
		tests := []struct {
			input    string
			expected string
		}{
			{"  test  ", "test"},
			{"test", "test"},
			{"", ""},
			{" \t\n test \t\n ", "test"},
		}
		
		for _, test := range tests {
			result := sanitizeQuery(test.input)
			if result != test.expected {
				t.Errorf("Expected sanitizeQuery(%q) = %q, got %q", test.input, test.expected, result)
			}
		}
	})
	
	t.Run("getClientIP with X-Forwarded-For", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.Header.Set("X-Forwarded-For", "192.168.1.1, 10.0.0.1")
		
		ip := getClientIP(req)
		if ip != "192.168.1.1" {
			t.Errorf("Expected IP 192.168.1.1, got %s", ip)
		}
	})
	
	t.Run("getClientIP without X-Forwarded-For", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.RemoteAddr = "192.168.1.2:1234"
		
		ip := getClientIP(req)
		if ip != "192.168.1.2" {
			t.Errorf("Expected IP 192.168.1.2, got %s", ip)
		}
	})
}

func TestQueryHandler(t *testing.T) {
	originalFactory := llm.Factory
	defer func() { llm.Factory = originalFactory }()
	llm.Factory = mockLLMFactory

	t.Run("Method not allowed", func(t *testing.T) {
		handler := NewHandler()
		handler.router = &MockRouter{}
		handler.cache = &MockCache{}
		
		req := httptest.NewRequest(http.MethodGet, "/query", nil)
		w := httptest.NewRecorder()
		
		handler.QueryHandler(w, req)
		
		if w.Code != http.StatusMethodNotAllowed {
			t.Errorf("Expected status %d, got %d", http.StatusMethodNotAllowed, w.Code)
		}
		
		headers := w.Header()
		requiredHeaders := []string{
			"Content-Type",
			"X-Content-Type-Options",
			"X-Frame-Options",
			"X-XSS-Protection",
			"Content-Security-Policy",
			"Referrer-Policy",
			"Cache-Control",
			"Strict-Transport-Security",
		}

		for _, header := range requiredHeaders {
			if headers.Get(header) == "" {
				t.Errorf("Expected header '%s' to be set", header)
			}
		}
	})
	
	t.Run("Rate limit exceeded", func(t *testing.T) {
		handler := NewHandler()
		handler.router = &MockRouter{}
		handler.cache = &MockCache{}
		
		mockRateLimiter := NewRateLimiter(60, 10)
		mockRateLimiter.SetAllowClientFunc(func(clientID string) bool {
			return false // Always deny
		})
		
		handler.rateLimiter = mockRateLimiter
		
		req := httptest.NewRequest(http.MethodPost, "/query", bytes.NewBufferString(`{"query":"test"}`))
		w := httptest.NewRecorder()
		
		handler.QueryHandler(w, req)
		
		if w.Code != http.StatusTooManyRequests {
			t.Errorf("Expected status %d, got %d", http.StatusTooManyRequests, w.Code)
		}
	})
	
	t.Run("Invalid JSON", func(t *testing.T) {
		handler := NewHandler()
		handler.router = &MockRouter{}
		handler.cache = &MockCache{}
		
		req := httptest.NewRequest(http.MethodPost, "/query", bytes.NewBufferString(`{"query":}`))
		w := httptest.NewRecorder()
		
		handler.QueryHandler(w, req)
		
		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected status %d, got %d", http.StatusBadRequest, w.Code)
		}
	})
	
	t.Run("Invalid request (empty query)", func(t *testing.T) {
		handler := NewHandler()
		handler.router = &MockRouter{}
		handler.cache = &MockCache{}
		
		req := httptest.NewRequest(http.MethodPost, "/query", bytes.NewBufferString(`{"query":""}`))
		w := httptest.NewRecorder()
		
		handler.QueryHandler(w, req)
		
		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected status %d, got %d", http.StatusBadRequest, w.Code)
		}
	})
	
	t.Run("Cache hit", func(t *testing.T) {
		handler := NewHandler()
		handler.router = &MockRouter{}
		
		cachedResponse := models.QueryResponse{
			Response:  "Cached response",
			Model:     models.OpenAI,
			Cached:    true,
			RequestID: "test-id",
		}
		
		handler.cache = &MockCache{
			getFunc: func(req models.QueryRequest) (models.QueryResponse, bool) {
				return cachedResponse, true
			},
		}
		
		req := httptest.NewRequest(http.MethodPost, "/query", bytes.NewBufferString(`{"query":"test"}`))
		w := httptest.NewRecorder()
		
		handler.QueryHandler(w, req)
		
		if w.Code != http.StatusOK {
			t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
		}
		
		var resp models.QueryResponse
		if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
			t.Fatalf("Error decoding response: %v", err)
		}
		
		if resp.Response != cachedResponse.Response {
			t.Errorf("Expected response %q, got %q", cachedResponse.Response, resp.Response)
		}
		
		if !resp.Cached {
			t.Errorf("Expected cached=true")
		}
	})
	
	t.Run("Routing error", func(t *testing.T) {
		handler := NewHandler()
		handler.cache = &MockCache{}
		handler.router = &MockRouter{
			routeRequestFunc: func(ctx context.Context, req models.QueryRequest) (models.ModelType, error) {
				return "", errors.New("no models available")
			},
		}
		
		req := httptest.NewRequest(http.MethodPost, "/query", bytes.NewBufferString(`{"query":"test"}`))
		w := httptest.NewRecorder()
		
		handler.QueryHandler(w, req)
		
		if w.Code != http.StatusServiceUnavailable {
			t.Errorf("Expected status %d, got %d", http.StatusServiceUnavailable, w.Code)
		}
	})
	
	t.Run("Query error with fallback success", func(t *testing.T) {
		handler := NewHandler()
		handler.cache = &MockCache{}
		handler.router = &MockRouter{
			routeRequestFunc: func(ctx context.Context, req models.QueryRequest) (models.ModelType, error) {
				return models.OpenAI, nil
			},
			fallbackOnErrorFunc: func(ctx context.Context, failedModel models.ModelType, req models.QueryRequest, err error) (models.ModelType, error) {
				return models.Gemini, nil
			},
		}
		
		llm.Factory = func(modelType models.ModelType) (llm.Client, error) {
			if modelType == models.OpenAI {
				return &MockLLMClient{
					modelType: modelType,
					queryFunc: func(ctx context.Context, query string, modelVersion string) (*llm.QueryResult, error) {
						return nil, myerrors.NewRateLimitError(string(modelType))
					},
				}, nil
			}
			return &MockLLMClient{
				modelType: modelType,
					queryFunc: func(ctx context.Context, query string, modelVersion string) (*llm.QueryResult, error) {
					return &llm.QueryResult{
						Response:     "Fallback response from " + string(modelType),
						StatusCode:   200,
						ResponseTime: 100,
						NumTokens:    10,
					}, nil
				},
			}, nil
		}
		
		req := httptest.NewRequest(http.MethodPost, "/query", bytes.NewBufferString(`{"query":"test"}`))
		w := httptest.NewRecorder()
		
		handler.QueryHandler(w, req)
		
		if w.Code != http.StatusOK {
			t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
		}
		
		var resp models.QueryResponse
		if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
			t.Fatalf("Error decoding response: %v", err)
		}
		
		if resp.Model != models.Gemini {
			t.Errorf("Expected model %s, got %s", models.Gemini, resp.Model)
		}
		
		if !strings.Contains(resp.Response, "Fallback response") {
			t.Errorf("Expected fallback response, got %q", resp.Response)
		}
	})
	
	t.Run("Query error with fallback failure", func(t *testing.T) {
		handler := NewHandler()
		handler.cache = &MockCache{}
		handler.router = &MockRouter{
			routeRequestFunc: func(ctx context.Context, req models.QueryRequest) (models.ModelType, error) {
				return models.OpenAI, nil
			},
			fallbackOnErrorFunc: func(ctx context.Context, failedModel models.ModelType, req models.QueryRequest, err error) (models.ModelType, error) {
				return "", errors.New("no fallback available")
			},
		}
		
		llm.Factory = func(modelType models.ModelType) (llm.Client, error) {
			return &MockLLMClient{
				modelType: modelType,
					queryFunc: func(ctx context.Context, query string, modelVersion string) (*llm.QueryResult, error) {
					return nil, myerrors.NewRateLimitError(string(modelType))
				},
			}, nil
		}
		
		req := httptest.NewRequest(http.MethodPost, "/query", bytes.NewBufferString(`{"query":"test"}`))
		w := httptest.NewRecorder()
		
		handler.QueryHandler(w, req)
		
		if w.Code != http.StatusInternalServerError {
			t.Errorf("Expected status %d, got %d", http.StatusInternalServerError, w.Code)
		}
	})
	
	t.Run("Successful query", func(t *testing.T) {
		handler := NewHandler()
		handler.cache = &MockCache{}
		handler.router = &MockRouter{
			routeRequestFunc: func(ctx context.Context, req models.QueryRequest) (models.ModelType, error) {
				return models.OpenAI, nil
			},
		}
		
		llm.Factory = mockLLMFactory
		
		req := httptest.NewRequest(http.MethodPost, "/query", bytes.NewBufferString(`{"query":"test"}`))
		w := httptest.NewRecorder()
		
		handler.QueryHandler(w, req)
		
		if w.Code != http.StatusOK {
			t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
		}
		
		var resp models.QueryResponse
		if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
			t.Fatalf("Error decoding response: %v", err)
		}
		
		if resp.Model != models.OpenAI {
			t.Errorf("Expected model %s, got %s", models.OpenAI, resp.Model)
		}
		
		if !strings.Contains(resp.Response, "Mock response") {
			t.Errorf("Expected mock response, got %q", resp.Response)
		}
		
		if resp.Cached {
			t.Errorf("Expected cached=false")
		}
	})
	
	t.Run("Context canceled", func(t *testing.T) {
		handler := NewHandler()
		handler.cache = &MockCache{}
		handler.router = &MockRouter{
			routeRequestFunc: func(ctx context.Context, req models.QueryRequest) (models.ModelType, error) {
				return models.OpenAI, nil
			},
		}
		
		llm.Factory = func(modelType models.ModelType) (llm.Client, error) {
			return &MockLLMClient{
				modelType: modelType,
					queryFunc: func(ctx context.Context, query string, modelVersion string) (*llm.QueryResult, error) {
					return nil, context.Canceled
				},
			}, nil
		}
		
		req := httptest.NewRequest(http.MethodPost, "/query", bytes.NewBufferString(`{"query":"test"}`))
		ctx, cancel := context.WithCancel(req.Context())
		cancel() // Cancel immediately
		req = req.WithContext(ctx)
		
		w := httptest.NewRecorder()
		
		handler.QueryHandler(w, req)
		
		if w.Code != 499 {
			t.Errorf("Expected status 499 (Client Closed Request), got %d", w.Code)
		}
		
		if !strings.Contains(w.Body.String(), "canceled") {
			t.Errorf("Expected error message to contain 'canceled', got %q", w.Body.String())
		}
	})
	
	t.Run("Context deadline exceeded", func(t *testing.T) {
		handler := NewHandler()
		handler.cache = &MockCache{}
		handler.router = &MockRouter{
			routeRequestFunc: func(ctx context.Context, req models.QueryRequest) (models.ModelType, error) {
				return models.OpenAI, nil
			},
		}
		
		llm.Factory = func(modelType models.ModelType) (llm.Client, error) {
			return &MockLLMClient{
				modelType: modelType,
					queryFunc: func(ctx context.Context, query string, modelVersion string) (*llm.QueryResult, error) {
					return nil, context.DeadlineExceeded
				},
			}, nil
		}
		
		req := httptest.NewRequest(http.MethodPost, "/query", bytes.NewBufferString(`{"query":"test"}`))
		ctx, cancel := context.WithTimeout(req.Context(), 1*time.Nanosecond)
		defer cancel()
		req = req.WithContext(ctx)
		
		w := httptest.NewRecorder()
		
		handler.QueryHandler(w, req)
		
		if w.Code != http.StatusRequestTimeout {
			t.Errorf("Expected status %d, got %d", http.StatusRequestTimeout, w.Code)
		}
		
		if !strings.Contains(w.Body.String(), "timed out") {
			t.Errorf("Expected error message to contain 'timed out', got %q", w.Body.String())
		}
	})
}

func TestStatusHandler(t *testing.T) {
	t.Run("Method not allowed", func(t *testing.T) {
		handler := NewHandler()
		
		req := httptest.NewRequest(http.MethodPost, "/status", nil)
		w := httptest.NewRecorder()
		
		handler.StatusHandler(w, req)
		
		if w.Code != http.StatusMethodNotAllowed {
			t.Errorf("Expected status %d, got %d", http.StatusMethodNotAllowed, w.Code)
		}
	})
	
	t.Run("Rate limit exceeded", func(t *testing.T) {
		handler := NewHandler()
		
		mockRateLimiter := NewRateLimiter(60, 10)
		mockRateLimiter.SetAllowClientFunc(func(clientID string) bool {
			return false // Always deny
		})
		
		handler.rateLimiter = mockRateLimiter
		
		req := httptest.NewRequest(http.MethodGet, "/status", nil)
		w := httptest.NewRecorder()
		
		handler.StatusHandler(w, req)
		
		if w.Code != http.StatusTooManyRequests {
			t.Errorf("Expected status %d, got %d", http.StatusTooManyRequests, w.Code)
		}
	})
	
	t.Run("Successful status", func(t *testing.T) {
		handler := NewHandler()
		availability := models.StatusResponse{
			OpenAI:  true,
			Gemini:  false,
			Mistral: true,
			Claude:  false,
		}
		
		handler.router = &MockRouter{
			getAvailabilityFunc: func() models.StatusResponse {
				return availability
			},
		}
		
		req := httptest.NewRequest(http.MethodGet, "/status", nil)
		w := httptest.NewRecorder()
		
		handler.StatusHandler(w, req)
		
		if w.Code != http.StatusOK {
			t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
		}
		
		var resp models.StatusResponse
		if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
			t.Fatalf("Error decoding response: %v", err)
		}
		
		if resp.OpenAI != availability.OpenAI {
			t.Errorf("Expected OpenAI availability %v, got %v", availability.OpenAI, resp.OpenAI)
		}
		
		if resp.Gemini != availability.Gemini {
			t.Errorf("Expected Gemini availability %v, got %v", availability.Gemini, resp.Gemini)
		}
		
		if resp.Mistral != availability.Mistral {
			t.Errorf("Expected Mistral availability %v, got %v", availability.Mistral, resp.Mistral)
		}
		
		if resp.Claude != availability.Claude {
			t.Errorf("Expected Claude availability %v, got %v", availability.Claude, resp.Claude)
		}
		
		headers := w.Header()
		requiredHeaders := []string{
			"Content-Type",
			"X-Content-Type-Options",
			"X-Frame-Options",
			"X-XSS-Protection",
			"Content-Security-Policy",
			"Referrer-Policy",
			"Cache-Control",
			"Strict-Transport-Security",
		}

		for _, header := range requiredHeaders {
			if headers.Get(header) == "" {
				t.Errorf("Expected header '%s' to be set", header)
			}
		}
	})
}

func TestHealthHandler(t *testing.T) {
	t.Run("Method not allowed", func(t *testing.T) {
		handler := NewHandler()
		
		req := httptest.NewRequest(http.MethodPost, "/health", nil)
		w := httptest.NewRecorder()
		
		handler.HealthHandler(w, req)
		
		if w.Code != http.StatusMethodNotAllowed {
			t.Errorf("Expected status %d, got %d", http.StatusMethodNotAllowed, w.Code)
		}
		
		headers := w.Header()
		requiredHeaders := []string{
			"Content-Type",
			"X-Content-Type-Options",
			"X-Frame-Options",
			"X-XSS-Protection",
			"Content-Security-Policy",
			"Referrer-Policy",
			"Cache-Control",
			"Strict-Transport-Security",
		}

		for _, header := range requiredHeaders {
			if headers.Get(header) == "" {
				t.Errorf("Expected header '%s' to be set", header)
			}
		}
	})
	
	t.Run("Rate limit exceeded", func(t *testing.T) {
		handler := NewHandler()
		
		mockRateLimiter := NewRateLimiter(60, 10)
		mockRateLimiter.SetAllowClientFunc(func(clientID string) bool {
			return false // Always deny
		})
		
		handler.rateLimiter = mockRateLimiter
		
		req := httptest.NewRequest(http.MethodGet, "/health", nil)
		w := httptest.NewRecorder()
		
		handler.HealthHandler(w, req)
		
		if w.Code != http.StatusTooManyRequests {
			t.Errorf("Expected status %d, got %d", http.StatusTooManyRequests, w.Code)
		}
	})
	
	t.Run("Successful health check", func(t *testing.T) {
		handler := NewHandler()
		
		req := httptest.NewRequest(http.MethodGet, "/health", nil)
		w := httptest.NewRecorder()
		
		handler.HealthHandler(w, req)
		
		if w.Code != http.StatusOK {
			t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
		}
		
		var resp map[string]interface{}
		if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
			t.Fatalf("Error decoding response: %v", err)
		}
		
		if status, ok := resp["status"]; !ok || status != "ok" {
			t.Errorf("Expected status 'ok', got '%v'", status)
		}
		
		if _, ok := resp["timestamp"]; !ok {
			t.Errorf("Expected timestamp in response")
		}
		
		if w.Header().Get("Content-Type") != "application/json" {
			t.Errorf("Expected Content-Type header 'application/json', got '%s'", w.Header().Get("Content-Type"))
		}
	})
}

func TestGetEnvAsInt(t *testing.T) {
	t.Run("Returns default when env var not set", func(t *testing.T) {
		result := getEnvAsInt("NONEXISTENT_VAR", 42)
		if result != 42 {
			t.Errorf("Expected default value 42, got %d", result)
		}
	})
	
	t.Run("Returns parsed value when env var is set", func(t *testing.T) {
		os.Setenv("TEST_INT_VAR", "100")
		defer os.Unsetenv("TEST_INT_VAR")
		
		result := getEnvAsInt("TEST_INT_VAR", 42)
		if result != 100 {
			t.Errorf("Expected parsed value 100, got %d", result)
		}
	})
	
	t.Run("Returns default when env var is empty string", func(t *testing.T) {
		os.Setenv("TEST_EMPTY_VAR", "")
		defer os.Unsetenv("TEST_EMPTY_VAR")
		
		result := getEnvAsInt("TEST_EMPTY_VAR", 42)
		if result != 42 {
			t.Errorf("Expected default value 42, got %d", result)
		}
	})
	
	t.Run("Returns default when env var is not a valid integer", func(t *testing.T) {
		os.Setenv("TEST_INVALID_VAR", "not_a_number")
		defer os.Unsetenv("TEST_INVALID_VAR")
		
		result := getEnvAsInt("TEST_INVALID_VAR", 42)
		if result != 42 {
			t.Errorf("Expected default value 42, got %d", result)
		}
	})
	
	t.Run("Handles whitespace in env var", func(t *testing.T) {
		os.Setenv("TEST_WHITESPACE_VAR", "  100  ")
		defer os.Unsetenv("TEST_WHITESPACE_VAR")
		
		result := getEnvAsInt("TEST_WHITESPACE_VAR", 42)
		if result != 100 {
			t.Errorf("Expected parsed value 100, got %d", result)
		}
	})
	
	t.Run("Handles negative numbers", func(t *testing.T) {
		os.Setenv("TEST_NEGATIVE_VAR", "-50")
		defer os.Unsetenv("TEST_NEGATIVE_VAR")
		
		result := getEnvAsInt("TEST_NEGATIVE_VAR", 42)
		if result != -50 {
			t.Errorf("Expected parsed value -50, got %d", result)
		}
	})
}
