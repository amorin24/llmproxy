package tracing

import (
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

func EnrichWithCost(span trace.Span, costUSD float64, inputTokens, outputTokens int) {
	span.SetAttributes(
		attribute.Float64("llm.cost_usd", costUSD),
		attribute.Int("llm.input_tokens", inputTokens),
		attribute.Int("llm.output_tokens", outputTokens),
		attribute.Int("llm.total_tokens", inputTokens+outputTokens),
	)
}

func EnrichWithTokens(span trace.Span, inputTokens, outputTokens int) {
	span.SetAttributes(
		attribute.Int("llm.input_tokens", inputTokens),
		attribute.Int("llm.output_tokens", outputTokens),
		attribute.Int("llm.total_tokens", inputTokens+outputTokens),
	)
}

func EnrichWithFallback(span trace.Span, fallbackOccurred bool, originalProvider, fallbackProvider string) {
	span.SetAttributes(
		attribute.Bool("llm.fallback_occurred", fallbackOccurred),
	)

	if fallbackOccurred {
		span.SetAttributes(
			attribute.String("llm.original_provider", originalProvider),
			attribute.String("llm.fallback_provider", fallbackProvider),
		)
		span.AddEvent("fallback_triggered", trace.WithAttributes(
			attribute.String("from", originalProvider),
			attribute.String("to", fallbackProvider),
		))
	}
}

func EnrichWithRetry(span trace.Span, retryCount int, maxRetries int) {
	span.SetAttributes(
		attribute.Int("llm.retry_count", retryCount),
		attribute.Int("llm.max_retries", maxRetries),
		attribute.Bool("llm.retried", retryCount > 0),
	)

	if retryCount > 0 {
		span.AddEvent("retry_occurred", trace.WithAttributes(
			attribute.Int("attempt", retryCount),
		))
	}
}

func EnrichWithCache(span trace.Span, cacheHit bool, cacheKey string) {
	span.SetAttributes(
		attribute.Bool("llm.cache_hit", cacheHit),
		attribute.String("llm.cache_key", cacheKey),
	)

	if cacheHit {
		span.AddEvent("cache_hit", trace.WithAttributes(
			attribute.String("key", cacheKey),
		))
	} else {
		span.AddEvent("cache_miss", trace.WithAttributes(
			attribute.String("key", cacheKey),
		))
	}
}

func EnrichWithProvider(span trace.Span, provider, modelVersion string) {
	span.SetAttributes(
		attribute.String("llm.provider", provider),
		attribute.String("llm.model_version", modelVersion),
	)
}

func EnrichWithLatency(span trace.Span, latencyMs int64) {
	span.SetAttributes(
		attribute.Int64("llm.latency_ms", latencyMs),
	)
}

func EnrichWithCircuitBreaker(span trace.Span, state string, failureCount int) {
	span.SetAttributes(
		attribute.String("llm.circuit_breaker_state", state),
		attribute.Int("llm.circuit_breaker_failures", failureCount),
	)

	if state == "open" {
		span.AddEvent("circuit_breaker_open", trace.WithAttributes(
			attribute.Int("failures", failureCount),
		))
	}
}

func EnrichWithJob(span trace.Span, jobID string, jobStatus string) {
	span.SetAttributes(
		attribute.String("llm.job_id", jobID),
		attribute.String("llm.job_status", jobStatus),
	)
}

func EnrichWithRequestContext(span trace.Span, requestID, tenant string, maxCostUSD float64) {
	span.SetAttributes(
		attribute.String("llm.request_id", requestID),
	)

	if tenant != "" {
		span.SetAttributes(attribute.String("llm.tenant", tenant))
	}

	if maxCostUSD > 0 {
		span.SetAttributes(attribute.Float64("llm.max_cost_usd", maxCostUSD))
	}
}

func EnrichWithSLO(span trace.Span, sloViolated bool, metric string, target, actual float64) {
	span.SetAttributes(
		attribute.Bool("llm.slo_violated", sloViolated),
	)

	if sloViolated {
		span.SetAttributes(
			attribute.String("llm.slo_metric", metric),
			attribute.Float64("llm.slo_target", target),
			attribute.Float64("llm.slo_actual", actual),
		)
		span.AddEvent("slo_violation", trace.WithAttributes(
			attribute.String("metric", metric),
			attribute.Float64("target", target),
			attribute.Float64("actual", actual),
		))
	}
}

func EnrichWithAnomaly(span trace.Span, anomalyDetected bool, actualCost, baseline, multiplier float64) {
	span.SetAttributes(
		attribute.Bool("llm.cost_anomaly_detected", anomalyDetected),
	)

	if anomalyDetected {
		span.SetAttributes(
			attribute.Float64("llm.cost_actual", actualCost),
			attribute.Float64("llm.cost_baseline", baseline),
			attribute.Float64("llm.cost_multiplier", multiplier),
		)
		span.AddEvent("cost_anomaly", trace.WithAttributes(
			attribute.Float64("actual", actualCost),
			attribute.Float64("baseline", baseline),
			attribute.Float64("multiplier", multiplier),
		))
	}
}
