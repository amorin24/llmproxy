package v1

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/amorin24/llmproxy/pkg/context"
	"github.com/amorin24/llmproxy/pkg/models"
	"github.com/amorin24/llmproxy/pkg/pricing"
	"github.com/amorin24/llmproxy/pkg/tracing"
	"go.opentelemetry.io/otel/attribute"
)

type GatewayHandler struct {
	catalogLoader *pricing.CatalogLoader
	costEstimator *pricing.CostEstimator
}

func NewGatewayHandler(catalogLoader *pricing.CatalogLoader) *GatewayHandler {
	return &GatewayHandler{
		catalogLoader: catalogLoader,
		costEstimator: pricing.NewCostEstimator(catalogLoader),
	}
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

	ctx, span := tracing.StartSpan(r.Context(), "gateway.cost_estimate",
		attribute.String("provider", pricing.MapModelTypeToProvider(req.Model)),
		attribute.String("model_version", req.ModelVersion),
	)
	defer span.End()

	provider := pricing.MapModelTypeToProvider(req.Model)
	modelVersion := req.ModelVersion
	if modelVersion == "" {
		modelVersion = pricing.GetDefaultModelVersion(req.Model)
	}

	inputTokens := pricing.EstimateTokenCount(req.Query)
	expectedOutputTokens := 100
	if req.ExpectedResponseTokens != nil {
		expectedOutputTokens = *req.ExpectedResponseTokens
	}

	estimate, err := h.costEstimator.EstimatePreCall(provider, modelVersion, inputTokens, expectedOutputTokens)
	if err != nil {
		tracing.RecordError(span, err)
		sendErrorResponse(w, http.StatusBadRequest, "Failed to estimate cost: "+err.Error(), "ESTIMATION_FAILED", "")
		return
	}

	response := CostEstimateResponse{
		Model:               req.Model,
		ModelVersion:        modelVersion,
		InputTokens:         estimate.InputTokens,
		OutputTokens:        estimate.OutputTokens,
		EstimatedCostUSD:    estimate.EstimatedCostUSD,
		PricePerInputToken:  estimate.PricePerInputToken,
		PricePerOutputToken: estimate.PricePerOutputToken,
	}

	tracing.AddSpanAttributes(span,
		attribute.Int("input_tokens", estimate.InputTokens),
		attribute.Int("output_tokens", estimate.OutputTokens),
		attribute.Float64("estimated_cost_usd", estimate.EstimatedCostUSD),
	)

	_ = ctx

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
