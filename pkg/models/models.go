package models

import "time"

type ModelType string

const (
	OpenAI  ModelType = "openai"
	Gemini  ModelType = "gemini"
	Mistral ModelType = "mistral"
	Claude  ModelType = "claude"
)

type TaskType string

const (
	TextGeneration   TaskType = "text_generation"
	Summarization    TaskType = "summarization"
	SentimentAnalysis TaskType = "sentiment_analysis"
	QuestionAnswering TaskType = "question_answering"
	Other            TaskType = "other"
)

type QueryRequest struct {
	Query     string    `json:"query"`
	Model     ModelType `json:"model,omitempty"`     // Optional - if not provided, will be determined by the proxy
	TaskType  TaskType  `json:"task_type,omitempty"` // Optional - helps with model selection
	RequestID string    `json:"request_id,omitempty"` // Optional - for tracking requests
}

type QueryResponse struct {
	Response      string    `json:"response"`
	Model         ModelType `json:"model"`
	ResponseTime  int64     `json:"response_time_ms"`
	Timestamp     time.Time `json:"timestamp"`
	Cached        bool      `json:"cached"`
	Error         string    `json:"error,omitempty"`
	ErrorType     string    `json:"error_type,omitempty"`
	InputTokens   int       `json:"input_tokens,omitempty"`
	OutputTokens  int       `json:"output_tokens,omitempty"`
	TotalTokens   int       `json:"total_tokens,omitempty"`
	NumTokens     int       `json:"num_tokens,omitempty"` // Deprecated: Use TotalTokens instead
	NumRetries    int       `json:"num_retries,omitempty"`
	RequestID     string    `json:"request_id,omitempty"`
	OriginalModel ModelType `json:"original_model,omitempty"` // If fallback occurred
}

type StatusResponse struct {
	OpenAI  bool `json:"openai"`
	Gemini  bool `json:"gemini"`
	Mistral bool `json:"mistral"`
	Claude  bool `json:"claude"`
}
