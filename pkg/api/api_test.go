package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/amorin24/llmproxy/pkg/models"
)

func TestAPIStatusHandler(t *testing.T) {
	handler := NewHandler()
	
	req := httptest.NewRequest(http.MethodGet, "/status", nil)
	w := httptest.NewRecorder()
	
	handler.StatusHandler(w, req)
	
	if w.Code != http.StatusOK {
		t.Errorf("Expected status code %d, got %d", http.StatusOK, w.Code)
	}
	
	var resp models.StatusResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("Error decoding response: %v", err)
	}
}

func TestQueryHandlerValidation(t *testing.T) {
	handler := NewHandler()
	
	testCases := []struct {
		name       string
		requestBody string
		expectedStatus int
		expectError bool
	}{
		{
			name:          "Empty request",
			requestBody:   `{}`,
			expectedStatus: http.StatusBadRequest,
			expectError:   true,
		},
		{
			name:          "Missing query",
			requestBody:   `{"model":"openai"}`,
			expectedStatus: http.StatusBadRequest,
			expectError:   true,
		},
		{
			name:          "Query too long",
			requestBody:   `{"query":"` + string(make([]byte, 50000)) + `"}`,
			expectedStatus: http.StatusBadRequest,
			expectError:   true,
		},
		{
			name:          "Invalid model",
			requestBody:   `{"query":"test query", "model":"invalid_model"}`,
			expectedStatus: http.StatusBadRequest,
			expectError:   true,
		},
		{
			name:          "Invalid task type",
			requestBody:   `{"query":"test query", "taskType":"invalid_task"}`,
			expectedStatus: http.StatusBadRequest,
			expectError:   true,
		},
	}
	
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/query", bytes.NewBufferString(tc.requestBody))
			w := httptest.NewRecorder()
			
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
			}
		})
	}
}

func TestAPIHealthHandler(t *testing.T) {
	handler := NewHandler()
	
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	w := httptest.NewRecorder()
	
	handler.StatusHandler(w, req)
	
	if w.Code != http.StatusOK {
		t.Errorf("Expected status code %d, got %d", http.StatusOK, w.Code)
	}
	
	var resp map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("Error decoding response: %v", err)
	}
}

func TestAPISecurityHeaders(t *testing.T) {
	handler := NewHandler()
	
	req := httptest.NewRequest(http.MethodGet, "/status", nil)
	w := httptest.NewRecorder()
	
	handler.StatusHandler(w, req)
	
	expectedHeaders := map[string]string{
		"Content-Type":           "application/json",
		"X-Content-Type-Options": "nosniff",
		"X-Frame-Options":        "DENY",
		"X-XSS-Protection":       "1; mode=block",
	}
	
	for header, expectedValue := range expectedHeaders {
		if value := w.Header().Get(header); value != expectedValue {
			t.Errorf("Expected header %s to be '%s', got '%s'", header, expectedValue, value)
		}
	}
	
	cacheControl := w.Header().Get("Cache-Control")
	expectedCacheValues := []string{"no-store", "no-cache", "must-revalidate"}
	for _, expected := range expectedCacheValues {
		if !bytes.Contains([]byte(cacheControl), []byte(expected)) {
			t.Errorf("Expected Cache-Control header to contain '%s', got '%s'", expected, cacheControl)
		}
	}
}

func TestRateLimiting(t *testing.T) {
	handler := NewHandler()
	
	for i := 0; i < 20; i++ {
		req := httptest.NewRequest(http.MethodGet, "/health", nil)
		w := httptest.NewRecorder()
		
		handler.HealthHandler(w, req)
		
		if i < 10 && w.Code != http.StatusOK {
			t.Errorf("Expected status code %d for request %d, got %d", http.StatusOK, i, w.Code)
		}
		
		if w.Code == http.StatusTooManyRequests {
			var resp map[string]interface{}
			if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
				t.Fatalf("Error decoding response: %v", err)
			}
			
			if _, ok := resp["error"]; !ok {
				t.Errorf("Expected error in rate-limited response, got %+v", resp)
			}
			
			break
		}
	}
}
