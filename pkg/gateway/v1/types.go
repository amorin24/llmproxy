package v1

import (
	"github.com/amorin24/llmproxy/pkg/models"
)

type GatewayQueryRequest struct {
	Query string `json:"query"`
	
	Model models.ModelType `json:"model"`
	
	ModelVersion string `json:"model_version,omitempty"`
	
	TaskType models.TaskType `json:"task_type"`
	
	MaxCostUSD *float64 `json:"max_cost_usd,omitempty"`
	
	DryRun bool `json:"dry_run,omitempty"`
	
	Tenant string `json:"tenant,omitempty"`
	
	RequestID string `json:"request_id,omitempty"`
}

type GatewayQueryResponse struct {
	RequestID string `json:"request_id"`
	
	Response string `json:"response"`
	
	Model models.ModelType `json:"model"`
	
	ModelVersion string `json:"model_version"`
	
	Cached bool `json:"cached"`
	
	ResponseTimeMs int64 `json:"response_time_ms"`
	
	NumTokens int `json:"num_tokens,omitempty"`
	
	CostUSD *float64 `json:"cost_usd,omitempty"`
	
	Tenant string `json:"tenant,omitempty"`
}

type CostEstimateRequest struct {
	Query string `json:"query"`
	
	Model models.ModelType `json:"model"`
	
	ModelVersion string `json:"model_version,omitempty"`
	
	ExpectedResponseTokens *int `json:"expected_response_tokens,omitempty"`
}

type CostEstimateResponse struct {
	Model models.ModelType `json:"model"`
	
	ModelVersion string `json:"model_version"`
	
	InputTokens int `json:"input_tokens"`
	
	OutputTokens int `json:"output_tokens"`
	
	EstimatedCostUSD float64 `json:"estimated_cost_usd"`
	
	PricePerInputToken float64 `json:"price_per_input_token"`
	
	PricePerOutputToken float64 `json:"price_per_output_token"`
}

type ErrorResponse struct {
	Error string `json:"error"`
	
	Code string `json:"code,omitempty"`
	
	RequestID string `json:"request_id,omitempty"`
}
