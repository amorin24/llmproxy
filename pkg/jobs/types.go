package jobs

import (
	"time"

	"github.com/amorin24/llmproxy/pkg/models"
)

type JobStatus string

const (
	JobStatusPending   JobStatus = "pending"
	JobStatusRunning   JobStatus = "running"
	JobStatusCompleted JobStatus = "completed"
	JobStatusFailed    JobStatus = "failed"
)

type Job struct {
	ID                string
	Query             string
	Model             models.ModelType
	ModelVersion      string
	Status            JobStatus
	EstimatedCostUSD  float64
	ActualCostUSD     float64
	Result            string
	Error             string
	CallbackURL       string
	CreatedAt         time.Time
	StartedAt         *time.Time
	CompletedAt       *time.Time
	InputTokens       int
	OutputTokens      int
	TotalTokens       int
	Provider          string
	RequestID         string
}

type JobSubmitRequest struct {
	Query        string           `json:"query"`
	Model        models.ModelType `json:"model,omitempty"`
	ModelVersion string           `json:"model_version,omitempty"`
	CallbackURL  string           `json:"callback_url,omitempty"`
}

type JobSubmitResponse struct {
	JobID            string  `json:"job_id"`
	Status           string  `json:"status"`
	EstimatedCostUSD float64 `json:"estimated_cost_usd"`
}

type JobStatusResponse struct {
	JobID            string  `json:"job_id"`
	Status           string  `json:"status"`
	EstimatedCostUSD float64 `json:"estimated_cost_usd,omitempty"`
	ActualCostUSD    float64 `json:"actual_cost_usd,omitempty"`
	CreatedAt        string  `json:"created_at"`
	StartedAt        string  `json:"started_at,omitempty"`
	CompletedAt      string  `json:"completed_at,omitempty"`
	Error            string  `json:"error,omitempty"`
}

type JobResultResponse struct {
	JobID         string `json:"job_id"`
	Status        string `json:"status"`
	Result        string `json:"result,omitempty"`
	ActualCostUSD float64 `json:"actual_cost_usd,omitempty"`
	InputTokens   int    `json:"input_tokens,omitempty"`
	OutputTokens  int    `json:"output_tokens,omitempty"`
	TotalTokens   int    `json:"total_tokens,omitempty"`
	Provider      string `json:"provider,omitempty"`
	Error         string `json:"error,omitempty"`
}
