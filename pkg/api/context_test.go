package api

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/amorin24/llmproxy/pkg/api/testutil"
	"github.com/amorin24/llmproxy/pkg/llm"
	"github.com/amorin24/llmproxy/pkg/models"
)

func TestContextCancellation(t *testing.T) {
	originalFactory := llm.Factory
	
	mockFactory := func(modelType models.ModelType) (llm.Client, error) {
		return &testutil.MockLLMClient{
			ModelType: modelType,
			QueryFunc: func(ctx context.Context, query string) (*llm.QueryResult, error) {
				select {
				case <-ctx.Done():
					return nil, ctx.Err()
				case <-time.After(100 * time.Millisecond):
					return &llm.QueryResult{
						Response: "This should not be returned",
					}, nil
				}
			},
		}, nil
	}
	
	defer func() { 
		llm.Factory = originalFactory 
	}()
	
	t.Run("Context canceled during query", func(t *testing.T) {
		handler := NewHandler()
		
		llm.Factory = mockFactory
		
		ctx, cancel := context.WithCancel(context.Background())
		req := httptest.NewRequest(http.MethodPost, "/query", bytes.NewBufferString(`{"query":"test"}`)).WithContext(ctx)
		w := httptest.NewRecorder()
		
		go func() {
			time.Sleep(10 * time.Millisecond)
			cancel()
		}()
		
		handler.QueryHandler(w, req)
		
		if w.Code != 499 && w.Code != http.StatusInternalServerError {
			t.Errorf("Expected status 499 or 500, got %d", w.Code)
		}
		
		var resp map[string]interface{}
		if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
			t.Fatalf("Error decoding response: %v", err)
		}
		
		if _, ok := resp["error"]; !ok {
			t.Errorf("Expected error in response, got %+v", resp)
		}
	})
}

func TestRequestSizeLimits(t *testing.T) {
	t.Run("Request too large", func(t *testing.T) {
		handler := NewHandler()
		
	largeQuery := strings.Repeat("a", 32000+1) // maxQueryLength is 32000
		req := httptest.NewRequest(http.MethodPost, "/query", bytes.NewBufferString(`{"query":"`+largeQuery+`"}`))
		w := httptest.NewRecorder()
		
		handler.QueryHandler(w, req)
		
		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected status %d, got %d", http.StatusBadRequest, w.Code)
		}
		
		var resp map[string]interface{}
		if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
			t.Fatalf("Error decoding response: %v", err)
		}
		
		if _, ok := resp["error"]; !ok {
			t.Errorf("Expected error in response, got %+v", resp)
		}
	})
}

func TestContextSecurityHeaders(t *testing.T) {
	t.Run("Security headers in response", func(t *testing.T) {
		handler := NewHandler()
		
		req := httptest.NewRequest(http.MethodPost, "/query", bytes.NewBufferString(`{"query":"test"}`))
		w := httptest.NewRecorder()
		
		handler.QueryHandler(w, req)
		
		if w.Header().Get("Content-Type") != "application/json" {
			t.Errorf("Expected Content-Type header 'application/json', got '%s'", w.Header().Get("Content-Type"))
		}
		
		if w.Header().Get("X-Content-Type-Options") != "nosniff" {
			t.Errorf("Expected X-Content-Type-Options header 'nosniff', got '%s'", w.Header().Get("X-Content-Type-Options"))
		}
		
		if w.Header().Get("X-Frame-Options") != "DENY" {
			t.Errorf("Expected X-Frame-Options header 'DENY', got '%s'", w.Header().Get("X-Frame-Options"))
		}
		
		if w.Header().Get("X-XSS-Protection") != "1; mode=block" {
			t.Errorf("Expected X-XSS-Protection header '1; mode=block', got '%s'", w.Header().Get("X-XSS-Protection"))
		}
		
		if w.Header().Get("Cache-Control") != "no-store, no-cache, must-revalidate, max-age=0" {
			t.Errorf("Expected Cache-Control header 'no-store, no-cache, must-revalidate, max-age=0', got '%s'", w.Header().Get("Cache-Control"))
		}
	})
}
