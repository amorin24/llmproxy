package slo

import (
	"time"

	"github.com/amorin24/llmproxy/pkg/models"
)

type SLO struct {
	Provider         models.ModelType
	LatencyP95Target time.Duration // Target: < 2s
	LatencyP99Target time.Duration // Target: < 5s
	ErrorRateTarget  float64       // Target: < 1% (0.01)
	AvailabilityTarget float64     // Target: > 99.5% (0.995)
}

type SLOMetrics struct {
	Provider         models.ModelType
	LatencyP95       time.Duration
	LatencyP99       time.Duration
	ErrorRate        float64
	Availability     float64
	ErrorBudget      float64 // Remaining error budget (0-1)
	LastUpdated      time.Time
}

type SLOViolation struct {
	Provider    models.ModelType
	Metric      string // "latency_p95", "latency_p99", "error_rate", "availability"
	Target      float64
	Actual      float64
	Timestamp   time.Time
	Description string
}

type ErrorBudget struct {
	Provider       models.ModelType
	TotalRequests  int64
	FailedRequests int64
	BudgetRemaining float64 // 0-1, where 1 = 100% budget remaining
	WindowStart    time.Time
	WindowEnd      time.Time
}

func DefaultSLOs() map[models.ModelType]SLO {
	return map[models.ModelType]SLO{
		models.OpenAI: {
			Provider:           models.OpenAI,
			LatencyP95Target:   2 * time.Second,
			LatencyP99Target:   5 * time.Second,
			ErrorRateTarget:    0.01, // 1%
			AvailabilityTarget: 0.995, // 99.5%
		},
		models.Gemini: {
			Provider:           models.Gemini,
			LatencyP95Target:   2 * time.Second,
			LatencyP99Target:   5 * time.Second,
			ErrorRateTarget:    0.01,
			AvailabilityTarget: 0.995,
		},
		models.Mistral: {
			Provider:           models.Mistral,
			LatencyP95Target:   2 * time.Second,
			LatencyP99Target:   5 * time.Second,
			ErrorRateTarget:    0.01,
			AvailabilityTarget: 0.995,
		},
		models.Claude: {
			Provider:           models.Claude,
			LatencyP95Target:   2 * time.Second,
			LatencyP99Target:   5 * time.Second,
			ErrorRateTarget:    0.01,
			AvailabilityTarget: 0.995,
		},
		models.VertexAI: {
			Provider:           models.VertexAI,
			LatencyP95Target:   2 * time.Second,
			LatencyP99Target:   5 * time.Second,
			ErrorRateTarget:    0.01,
			AvailabilityTarget: 0.995,
		},
		models.Bedrock: {
			Provider:           models.Bedrock,
			LatencyP95Target:   2 * time.Second,
			LatencyP99Target:   5 * time.Second,
			ErrorRateTarget:    0.01,
			AvailabilityTarget: 0.995,
		},
	}
}
