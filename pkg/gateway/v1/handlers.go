package v1

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/amorin24/llmproxy/pkg/context"
	"github.com/amorin24/llmproxy/pkg/models"
)

type GatewayHandler struct {
}

func NewGatewayHandler() *GatewayHandler {
	return &GatewayHandler{}
}

func (h *GatewayHandler) QueryHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("X-Frame-Options", "DENY")
	w.Header().Set("X-XSS-Protection", "1; mode=block")
	w.Header().Set("Content-Security-Policy", "default-src 'self'")
	w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
	w.Header().Set("Cache-Control", "no-store, no-cache, must-revalidate, private")
	w.Header().Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")

	if r.Method != http.MethodPost {
		sendErrorResponse(w, http.StatusMethodNotAllowed, "Method not allowed", "METHOD_NOT_ALLOWED", "")
		return
	}

	var req GatewayQueryRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		sendErrorResponse(w, http.StatusBadRequest, "Invalid JSON: "+err.Error(), "INVALID_JSON", "")
		return
	}

	var reqCtx *context.RequestContext
	if req.RequestID != "" {
		reqCtx = context.NewRequestContextWithID(r.Context(), req.RequestID)
	} else {
		reqCtx = context.NewRequestContext(r.Context())
	}

	if req.Tenant != "" {
		reqCtx.WithTenant(req.Tenant)
	}

	if req.MaxCostUSD != nil {
		reqCtx.WithMaxCost(*req.MaxCostUSD)
	}

	if err := validateGatewayQueryRequest(req); err != nil {
		sendErrorResponse(w, http.StatusBadRequest, err.Error(), "INVALID_REQUEST", reqCtx.RequestID)
		return
	}

	response := GatewayQueryResponse{
		RequestID:      reqCtx.RequestID,
		Response:       "Phase 0: Gateway API endpoint created. Full implementation in Phase 1+",
		Model:          req.Model,
		ModelVersion:   req.ModelVersion,
		Cached:         false,
		ResponseTimeMs: reqCtx.ElapsedMilliseconds(),
		Tenant:         reqCtx.Tenant,
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

func (h *GatewayHandler) CostEstimateHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if r.Method != http.MethodPost {
		sendErrorResponse(w, http.StatusMethodNotAllowed, "Method not allowed", "METHOD_NOT_ALLOWED", "")
		return
	}

	var req CostEstimateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		sendErrorResponse(w, http.StatusBadRequest, "Invalid JSON: "+err.Error(), "INVALID_JSON", "")
		return
	}

	response := CostEstimateResponse{
		Model:               req.Model,
		ModelVersion:        req.ModelVersion,
		InputTokens:         100,
		OutputTokens:        50,
		EstimatedCostUSD:    0.001,
		PricePerInputToken:  0.01,
		PricePerOutputToken: 0.03,
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

func validateGatewayQueryRequest(req GatewayQueryRequest) error {
	if strings.TrimSpace(req.Query) == "" {
		return models.ErrEmptyQuery
	}

	if len(req.Query) > 100000 {
		return models.ErrQueryTooLong
	}

	if req.Model == "" {
		return models.ErrInvalidModel
	}

	if req.TaskType == "" {
		return models.ErrInvalidTaskType
	}

	return nil
}

func sendErrorResponse(w http.ResponseWriter, statusCode int, message string, code string, requestID string) {
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(ErrorResponse{
		Error:     message,
		Code:      code,
		RequestID: requestID,
	})
}
