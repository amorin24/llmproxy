package monitoring

import (
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	RequestsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "llmproxy_requests_total",
			Help: "The total number of requests processed by model and status",
		},
		[]string{"model", "status"},
	)

	RequestDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "llmproxy_request_duration_seconds",
			Help:    "The duration of requests in seconds by model",
			Buckets: prometheus.ExponentialBuckets(0.1, 2, 10), // 0.1s to ~102.4s
		},
		[]string{"model"},
	)

	TokensProcessed = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "llmproxy_tokens_processed_total",
			Help: "The total number of tokens processed by model and type",
		},
		[]string{"model", "type"},
	)

	CacheHits = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "llmproxy_cache_hits_total",
			Help: "The total number of cache hits and misses",
		},
		[]string{"result"},
	)

	ActiveRequests = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "llmproxy_active_requests",
			Help: "The number of currently active requests by model",
		},
		[]string{"model"},
	)

	ModelAvailability = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "llmproxy_model_availability",
			Help: "The availability status of each model (1=available, 0=unavailable)",
		},
		[]string{"model"},
	)

	CostTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "llmproxy_cost_usd_total",
			Help: "The total cost in USD by provider and model",
		},
		[]string{"provider", "model"},
	)

	CostPerRequest = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "llmproxy_cost_per_request_usd",
			Help:    "The cost per request in USD by provider and model",
			Buckets: prometheus.ExponentialBuckets(0.0001, 2, 15), // $0.0001 to ~$1.64
		},
		[]string{"provider", "model"},
	)

	TokenCostTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "llmproxy_token_cost_usd_total",
			Help: "The total token cost in USD by provider, model, and token type",
		},
		[]string{"provider", "model", "token_type"},
	)

	CostSavingsFromCache = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "llmproxy_cost_savings_from_cache_usd_total",
			Help: "The total cost savings in USD from cache hits",
		},
	)

	EstimatedVsActualCost = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "llmproxy_estimated_vs_actual_cost_ratio",
			Help:    "The ratio of estimated cost to actual cost",
			Buckets: prometheus.LinearBuckets(0.5, 0.1, 11), // 0.5 to 1.5
		},
		[]string{"provider", "model"},
	)
)

func RecordRequest(model string, status int, duration time.Duration) {
	RequestsTotal.WithLabelValues(model, http.StatusText(status)).Inc()
	RequestDuration.WithLabelValues(model).Observe(duration.Seconds())
}

func RecordTokens(model string, inputTokens, outputTokens int) {
	TokensProcessed.WithLabelValues(model, "input").Add(float64(inputTokens))
	TokensProcessed.WithLabelValues(model, "output").Add(float64(outputTokens))
}

func RecordCacheHit() {
	CacheHits.WithLabelValues("hit").Inc()
}

func RecordCacheMiss() {
	CacheHits.WithLabelValues("miss").Inc()
}

func IncreaseActiveRequests(model string) {
	ActiveRequests.WithLabelValues(model).Inc()
}

func DecreaseActiveRequests(model string) {
	ActiveRequests.WithLabelValues(model).Dec()
}

func SetModelAvailability(model string, available bool) {
	value := 0.0
	if available {
		value = 1.0
	}
	ModelAvailability.WithLabelValues(model).Set(value)
}

func RecordCost(provider string, model string, costUSD float64) {
	CostTotal.WithLabelValues(provider, model).Add(costUSD)
	CostPerRequest.WithLabelValues(provider, model).Observe(costUSD)
}

func RecordTokenCost(provider string, model string, tokenType string, costUSD float64) {
	TokenCostTotal.WithLabelValues(provider, model, tokenType).Add(costUSD)
}

func RecordCostSavingsFromCache(costUSD float64) {
	CostSavingsFromCache.Add(costUSD)
}

func RecordEstimatedVsActualCost(provider string, model string, estimatedCost float64, actualCost float64) {
	if estimatedCost > 0 {
		ratio := actualCost / estimatedCost
		EstimatedVsActualCost.WithLabelValues(provider, model).Observe(ratio)
	}
}
