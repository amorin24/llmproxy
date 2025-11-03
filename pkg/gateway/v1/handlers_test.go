package v1

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/amorin24/llmproxy/pkg/pricing"
)

func TestQueryHandler(t *testing.T) {
	catalogLoader, err := pricing.NewCatalogLoader("../../docs/price-catalog.json")
	if err != nil {
		t.Skipf("Skipping test: price catalog not found: %v", err)
	}

	handler := NewGatewayHandler(catalogLoader)

	tests := []struct {
		name           string
		method         string
		body           string
		expectedStatus int
		checkResponse  func(*testing.T, *httptest.ResponseRecorder)
	}{
		{
			name:           "Method not allowed",
			method:         http.MethodGet,
			body:           "",
			expectedStatus: http.StatusMethodNotAllowed,
			checkResponse: func(t *testing.T, w *httptest.ResponseRecorder) {
				var resp ErrorResponse
				if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
					t.Errorf("Failed to decode response: %v", err)
				}
				if resp.Code != "METHOD_NOT_ALLOWED" {
					t.Errorf("Expected error code METHOD_NOT_ALLOWED, got %s", resp.Code)
				}
			},
		},
		{
			name:           "Invalid JSON",
			method:         http.MethodPost,
			body:           `{invalid json}`,
			expectedStatus: http.StatusBadRequest,
			checkResponse: func(t *testing.T, w *httptest.ResponseRecorder) {
				var resp ErrorResponse
				if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
					t.Errorf("Failed to decode response: %v", err)
				}
				if resp.Code != "INVALID_JSON" {
					t.Errorf("Expected error code INVALID_JSON, got %s", resp.Code)
				}
			},
		},
		{
			name:           "Empty query",
			method:         http.MethodPost,
			body:           `{"query":"","model":"openai","task_type":"text_generation"}`,
			expectedStatus: http.StatusBadRequest,
			checkResponse: func(t *testing.T, w *httptest.ResponseRecorder) {
				var resp ErrorResponse
				if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
					t.Errorf("Failed to decode response: %v", err)
				}
				if resp.Code != "INVALID_REQUEST" {
					t.Errorf("Expected error code INVALID_REQUEST, got %s", resp.Code)
				}
			},
		},
		{
			name:           "Missing model",
			method:         http.MethodPost,
			body:           `{"query":"test query","task_type":"text_generation"}`,
			expectedStatus: http.StatusBadRequest,
			checkResponse: func(t *testing.T, w *httptest.ResponseRecorder) {
				var resp ErrorResponse
				if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
					t.Errorf("Failed to decode response: %v", err)
				}
				if resp.Code != "INVALID_REQUEST" {
					t.Errorf("Expected error code INVALID_REQUEST, got %s", resp.Code)
				}
			},
		},
		{
			name:           "Missing task type",
			method:         http.MethodPost,
			body:           `{"query":"test query","model":"openai"}`,
			expectedStatus: http.StatusBadRequest,
			checkResponse: func(t *testing.T, w *httptest.ResponseRecorder) {
				var resp ErrorResponse
				if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
					t.Errorf("Failed to decode response: %v", err)
				}
				if resp.Code != "INVALID_REQUEST" {
					t.Errorf("Expected error code INVALID_REQUEST, got %s", resp.Code)
				}
			},
		},
		{
			name:           "Valid request",
			method:         http.MethodPost,
			body:           `{"query":"test query","model":"openai","task_type":"text_generation"}`,
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, w *httptest.ResponseRecorder) {
				var resp GatewayQueryResponse
				if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
					t.Errorf("Failed to decode response: %v", err)
				}
				if resp.RequestID == "" {
					t.Errorf("Expected request ID to be set")
				}
				if resp.Model != "openai" {
					t.Errorf("Expected model openai, got %s", resp.Model)
				}
			},
		},
		{
			name:           "Valid request with custom request ID",
			method:         http.MethodPost,
			body:           `{"query":"test query","model":"openai","task_type":"text_generation","request_id":"custom-id"}`,
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, w *httptest.ResponseRecorder) {
				var resp GatewayQueryResponse
				if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
					t.Errorf("Failed to decode response: %v", err)
				}
				if resp.RequestID != "custom-id" {
					t.Errorf("Expected request ID custom-id, got %s", resp.RequestID)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, "/v1/gateway/query", bytes.NewBufferString(tt.body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			handler.QueryHandler(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, w.Code)
			}

			if tt.checkResponse != nil {
				tt.checkResponse(t, w)
			}
		})
	}
}

func TestCostEstimateHandler(t *testing.T) {
	catalogLoader, err := pricing.NewCatalogLoader("../../docs/price-catalog.json")
	if err != nil {
		t.Skipf("Skipping test: price catalog not found: %v", err)
	}

	handler := NewGatewayHandler(catalogLoader)

	tests := []struct {
		name           string
		method         string
		body           string
		expectedStatus int
		checkResponse  func(*testing.T, *httptest.ResponseRecorder)
	}{
		{
			name:           "Method not allowed",
			method:         http.MethodGet,
			body:           "",
			expectedStatus: http.StatusMethodNotAllowed,
		},
		{
			name:           "Invalid JSON",
			method:         http.MethodPost,
			body:           `{invalid json}`,
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "Valid request",
			method:         http.MethodPost,
			body:           `{"query":"test query","model":"openai","model_version":"gpt-4o"}`,
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, w *httptest.ResponseRecorder) {
				var resp CostEstimateResponse
				if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
					t.Errorf("Failed to decode response: %v", err)
				}
				if resp.Model != "openai" {
					t.Errorf("Expected model openai, got %s", resp.Model)
				}
				if resp.EstimatedCostUSD <= 0 {
					t.Errorf("Expected positive cost estimate, got %f", resp.EstimatedCostUSD)
				}
			},
		},
		{
			name:           "Valid request with custom output tokens",
			method:         http.MethodPost,
			body:           `{"query":"test query","model":"openai","expected_response_tokens":500}`,
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, w *httptest.ResponseRecorder) {
				var resp CostEstimateResponse
				if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
					t.Errorf("Failed to decode response: %v", err)
				}
				if resp.OutputTokens != 500 {
					t.Errorf("Expected output tokens 500, got %d", resp.OutputTokens)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, "/v1/gateway/cost-estimate", bytes.NewBufferString(tt.body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			handler.CostEstimateHandler(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, w.Code)
			}

			if tt.checkResponse != nil {
				tt.checkResponse(t, w)
			}
		})
	}
}

func TestDryRunHandler(t *testing.T) {
	catalogLoader, err := pricing.NewCatalogLoader("../../docs/price-catalog.json")
	if err != nil {
		t.Skipf("Skipping test: price catalog not found: %v", err)
	}

	handler := NewGatewayHandler(catalogLoader)

	tests := []struct {
		name           string
		method         string
		body           string
		expectedStatus int
		checkResponse  func(*testing.T, *httptest.ResponseRecorder)
	}{
		{
			name:           "Method not allowed",
			method:         http.MethodGet,
			body:           "",
			expectedStatus: http.StatusMethodNotAllowed,
		},
		{
			name:           "Invalid JSON",
			method:         http.MethodPost,
			body:           `{invalid json}`,
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "Valid request",
			method:         http.MethodPost,
			body:           `{"query":"test query","model":"openai","task_type":"text_generation"}`,
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, w *httptest.ResponseRecorder) {
				var resp DryRunResponse
				if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
					t.Errorf("Failed to decode response: %v", err)
				}
				if !resp.Valid {
					t.Errorf("Expected valid=true for valid request")
				}
				if resp.EstimatedCostUSD <= 0 {
					t.Errorf("Expected positive cost estimate, got %f", resp.EstimatedCostUSD)
				}
			},
		},
		{
			name:           "Invalid request - empty query",
			method:         http.MethodPost,
			body:           `{"query":"","model":"openai","task_type":"text_generation"}`,
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, w *httptest.ResponseRecorder) {
				var resp DryRunResponse
				if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
					t.Errorf("Failed to decode response: %v", err)
				}
				if resp.Valid {
					t.Errorf("Expected valid=false for empty query")
				}
				if len(resp.Errors) == 0 {
					t.Errorf("Expected validation errors")
				}
			},
		},
		{
			name:           "Request with max cost constraint",
			method:         http.MethodPost,
			body:           `{"query":"test query","model":"openai","task_type":"text_generation","max_cost_usd":0.001}`,
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, w *httptest.ResponseRecorder) {
				var resp DryRunResponse
				if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
					t.Errorf("Failed to decode response: %v", err)
				}
				if resp.EstimatedCostUSD > 0.001 && resp.WithinBudget {
					t.Errorf("Expected within_budget=false when cost exceeds max")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, "/v1/gateway/dry-run", bytes.NewBufferString(tt.body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			handler.DryRunHandler(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, w.Code)
			}

			if tt.checkResponse != nil {
				tt.checkResponse(t, w)
			}
		})
	}
}

func TestValidateGatewayQueryRequest(t *testing.T) {
	tests := []struct {
		name        string
		req         GatewayQueryRequest
		expectError bool
	}{
		{
			name: "Valid request",
			req: GatewayQueryRequest{
				Query:    "test query",
				Model:    "openai",
				TaskType: "text_generation",
			},
			expectError: false,
		},
		{
			name: "Empty query",
			req: GatewayQueryRequest{
				Query:    "",
				Model:    "openai",
				TaskType: "text_generation",
			},
			expectError: true,
		},
		{
			name: "Query too long",
			req: GatewayQueryRequest{
				Query:    string(make([]byte, 100001)),
				Model:    "openai",
				TaskType: "text_generation",
			},
			expectError: true,
		},
		{
			name: "Missing model",
			req: GatewayQueryRequest{
				Query:    "test query",
				TaskType: "text_generation",
			},
			expectError: true,
		},
		{
			name: "Missing task type",
			req: GatewayQueryRequest{
				Query: "test query",
				Model: "openai",
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateGatewayQueryRequest(tt.req)
			if tt.expectError && err == nil {
				t.Errorf("Expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Expected no error but got: %v", err)
			}
		})
	}
}
