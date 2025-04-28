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
