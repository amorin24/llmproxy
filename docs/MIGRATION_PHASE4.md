# Phase 4 Migration Guide: Advanced Observability and SLOs

## Overview

Phase 4 of the LLM Gateway upgrade adds **advanced observability** and **Service Level Objectives (SLOs)** to make the gateway easy to operate and tune. This phase enables comprehensive monitoring, alerting, and operational excellence.

**Timeline**: 2 weeks (according to gateway upgrade plan)

**Status**: ✅ Core Implementation Complete

## What's New in Phase 4

### New Capabilities

1. **Service Level Objectives (SLOs)**
   - Per-provider SLO definitions (latency, error rate, availability)
   - Error budget tracking and alerts
   - SLO violation detection and logging
   - Prometheus metrics for SLO compliance

2. **Cost Anomaly Detection**
   - Baseline cost tracking (hourly and daily)
   - Cost spike detection (>2x baseline)
   - Unusual model usage pattern detection
   - Prometheus metrics for cost anomalies

3. **Job Monitoring**
   - Stuck job detection (running > 5 minutes)
   - Job failure rate tracking
   - Queue depth monitoring
   - Job throughput metrics

4. **Trace Enrichment**
   - Cost and token attributes in spans
   - Fallback information tracking
   - Retry count tracking
   - Cache hit/miss tracking
   - Circuit breaker state tracking
   - SLO violation tracking in traces

5. **Log Hygiene**
   - API key redaction
   - PII pattern redaction (email, phone, SSN, credit cards)
   - Structured logging with consistent fields
   - Sensitive field sanitization

6. **Comprehensive Alerting**
   - 25+ Prometheus alerting rules
   - SLO violation alerts
   - Cost spike alerts
   - Job health alerts
   - Provider availability alerts
   - Cache performance alerts

## Breaking Changes

**None.** Phase 4 is fully backward compatible with all existing functionality.

## New Features

### 1. Service Level Objectives (SLOs)

Track and enforce SLOs for all providers with automatic violation detection.

**Default SLO Targets:**
- **Latency p95**: < 2 seconds
- **Latency p99**: < 5 seconds
- **Error Rate**: < 1%
- **Availability**: > 99.5%

**Implementation** (`pkg/slo/`):
- `types.go` - SLO definitions and structures
- `tracker.go` - SLO tracking and violation detection

**Usage Example:**
```go
import (
    "github.com/amorin24/llmproxy/pkg/slo"
    "time"
)

// Create SLO tracker with 24-hour window
tracker := slo.NewTracker(24 * time.Hour)

// Record a request
tracker.RecordRequest(models.OpenAI, 1500*time.Millisecond, true)

// Update latency percentiles
tracker.UpdateLatency(models.OpenAI, 1800*time.Millisecond, 4200*time.Millisecond)

// Get current metrics
metrics := tracker.GetMetrics(models.OpenAI)
fmt.Printf("Error Rate: %.2f%%\n", metrics.ErrorRate * 100)
fmt.Printf("Availability: %.2f%%\n", metrics.Availability * 100)
fmt.Printf("Error Budget: %.2f%%\n", metrics.ErrorBudget * 100)

// Get error budget
budget := tracker.GetErrorBudget(models.OpenAI)
fmt.Printf("Budget Remaining: %.2f%%\n", budget.BudgetRemaining * 100)

// Get recent violations
violations := tracker.GetViolations(time.Now().Add(-1 * time.Hour))
for _, v := range violations {
    fmt.Printf("Violation: %s - %s (target: %.2f, actual: %.2f)\n",
        v.Provider, v.Metric, v.Target, v.Actual)
}
```

**Prometheus Metrics:**
```promql
# Current SLO metrics
llmproxy_slo_latency_p95_seconds{provider="openai"}
llmproxy_slo_latency_p99_seconds{provider="openai"}
llmproxy_slo_error_rate{provider="openai"}
llmproxy_slo_availability{provider="openai"}
llmproxy_slo_error_budget_remaining{provider="openai"}

# SLO violations
llmproxy_slo_violations_total{provider="openai", metric="latency_p95"}
```

### 2. Cost Anomaly Detection

Automatically detect cost spikes and unusual usage patterns.

**Implementation** (`pkg/anomaly/detector.go`):
- Baseline cost tracking with exponential moving average
- Cost spike detection (configurable threshold, default: 2x)
- Model usage pattern analysis
- Prometheus metrics for anomalies

**Usage Example:**
```go
import (
    "github.com/amorin24/llmproxy/pkg/anomaly"
)

// Create detector with 2x spike threshold
detector := anomaly.NewDetector(2.0)

// Record cost observations
detector.RecordCost(models.OpenAI, 0.05, "gpt-4o")
detector.RecordCost(models.Gemini, 0.02, "gemini-1.5-pro")

// Check for unusual usage patterns
alerts := detector.CheckUnusualUsage()
for _, alert := range alerts {
    fmt.Println(alert)
}

// Get baseline for a provider
baseline := detector.GetBaseline(models.OpenAI)
fmt.Printf("Hourly Baseline: $%.4f\n", baseline.HourlyBaseline)
fmt.Printf("Daily Baseline: $%.4f\n", baseline.DailyBaseline)

// Get recent anomalies
anomalies := detector.GetAnomalies(time.Now().Add(-1 * time.Hour))
for _, a := range anomalies {
    fmt.Printf("Anomaly: %s - $%.4f (%.2fx baseline)\n",
        a.Provider, a.ActualCost, a.Multiplier)
}

// Get usage patterns
patterns := detector.GetUsagePatterns()
for key, pattern := range patterns {
    fmt.Printf("%s: %d requests, $%.4f total\n",
        key, pattern.RequestCount, pattern.TotalCost)
}
```

**Prometheus Metrics:**
```promql
# Cost baselines and actuals
llmproxy_cost_baseline_usd_per_hour{provider="openai"}
llmproxy_cost_actual_usd_per_hour{provider="openai"}

# Anomaly detection
llmproxy_cost_anomalies_total{provider="openai"}

# Model usage
llmproxy_model_usage_requests_total{provider="openai", model_version="gpt-4o"}
```

### 3. Job Monitoring

Monitor job health with automatic detection of stuck jobs and high failure rates.

**Implementation** (`pkg/jobmonitor/monitor.go`):
- Stuck job detection (default: > 5 minutes)
- Failure rate tracking (default: 1-hour window)
- Queue depth monitoring (default: alert at 100)
- Prometheus metrics for job health

**Usage Example:**
```go
import (
    "github.com/amorin24/llmproxy/pkg/jobmonitor"
    "time"
)

// Create monitor
monitor := jobmonitor.NewMonitor(
    jobStore,
    5*time.Minute,  // stuck threshold
    1*time.Hour,    // failure rate window
    100,            // queue depth alert threshold
)

// Start monitoring
monitor.Start()

// Record job lifecycle
monitor.RecordJobStart("job-123")
monitor.RecordJobComplete("job-123", "openai", true, 2*time.Minute)

// Get current metrics
queueDepth := monitor.GetQueueDepth()
runningCount := monitor.GetRunningCount()
failureRate := monitor.GetFailureRate()

fmt.Printf("Queue Depth: %d\n", queueDepth)
fmt.Printf("Running Jobs: %d\n", runningCount)
fmt.Printf("Failure Rate: %.2f%%\n", failureRate * 100)
```

**Prometheus Metrics:**
```promql
# Job queue metrics
llmproxy_job_queue_depth
llmproxy_jobs_running
llmproxy_jobs_pending

# Job completion metrics
llmproxy_jobs_completed_total{provider="openai"}
llmproxy_jobs_failed_total{provider="openai"}

# Job duration
llmproxy_job_duration_seconds{provider="openai", status="completed"}

# Job health
llmproxy_jobs_stuck_total
llmproxy_job_failure_rate
```

### 4. Trace Enrichment

Enrich OpenTelemetry traces with comprehensive LLM-specific attributes.

**Implementation** (`pkg/tracing/enrichment.go`):
- Cost and token attributes
- Fallback tracking
- Retry tracking
- Cache hit/miss tracking
- Circuit breaker state
- SLO violation tracking
- Cost anomaly tracking

**Usage Example:**
```go
import (
    "github.com/amorin24/llmproxy/pkg/tracing"
)

// Start a span
ctx, span := tracing.StartSpan(ctx, "llm.query")
defer span.End()

// Enrich with cost and tokens
tracing.EnrichWithCost(span, 0.05, 100, 500)

// Enrich with provider info
tracing.EnrichWithProvider(span, "openai", "gpt-4o")

// Enrich with fallback info (if fallback occurred)
tracing.EnrichWithFallback(span, true, "openai", "gemini")

// Enrich with retry info
tracing.EnrichWithRetry(span, 2, 3)

// Enrich with cache info
tracing.EnrichWithCache(span, true, "cache-key-123")

// Enrich with circuit breaker info
tracing.EnrichWithCircuitBreaker(span, "closed", 0)

// Enrich with SLO info (if violated)
tracing.EnrichWithSLO(span, true, "latency_p95", 2.0, 3.5)

// Enrich with cost anomaly info (if detected)
tracing.EnrichWithAnomaly(span, true, 0.10, 0.04, 2.5)

// Enrich with request context
tracing.EnrichWithRequestContext(span, "req-123", "tenant-abc", 1.0)

// Enrich with job info
tracing.EnrichWithJob(span, "job-456", "completed")
```

**Trace Attributes:**
```
llm.cost_usd: 0.05
llm.input_tokens: 100
llm.output_tokens: 500
llm.total_tokens: 600
llm.provider: openai
llm.model_version: gpt-4o
llm.fallback_occurred: true
llm.original_provider: openai
llm.fallback_provider: gemini
llm.retry_count: 2
llm.max_retries: 3
llm.retried: true
llm.cache_hit: true
llm.cache_key: cache-key-123
llm.circuit_breaker_state: closed
llm.circuit_breaker_failures: 0
llm.slo_violated: true
llm.slo_metric: latency_p95
llm.slo_target: 2.0
llm.slo_actual: 3.5
llm.cost_anomaly_detected: true
llm.cost_actual: 0.10
llm.cost_baseline: 0.04
llm.cost_multiplier: 2.5
llm.request_id: req-123
llm.tenant: tenant-abc
llm.max_cost_usd: 1.0
llm.job_id: job-456
llm.job_status: completed
```

### 5. Log Hygiene

Automatically sanitize logs to prevent leaking sensitive data.

**Implementation** (`pkg/logging/sanitizer.go`):
- API key pattern detection and redaction
- PII pattern detection and redaction (email, phone, SSN, credit cards)
- Sensitive field sanitization
- Map and nested structure sanitization

**Usage Example:**
```go
import (
    "github.com/amorin24/llmproxy/pkg/logging"
)

// Create sanitizer
sanitizer := logging.NewSanitizer()

// Sanitize a log message
message := "API key: sk-abc123... for user john@example.com"
sanitized := sanitizer.Sanitize(message)
// Output: "API key: [REDACTED] for user [REDACTED]"

// Sanitize API keys only
apiKeySanitized := sanitizer.SanitizeAPIKeys("Bearer sk-abc123...")
// Output: "Bearer [REDACTED]"

// Sanitize PII only
piiSanitized := sanitizer.SanitizePII("Call 555-123-4567")
// Output: "Call [REDACTED]"

// Sanitize a map
data := map[string]interface{}{
    "api_key": "sk-abc123...",
    "email": "user@example.com",
    "query": "What is AI?",
}
sanitizedMap := sanitizer.SanitizeFields(data)
// Output: {"api_key": "[REDACTED]", "email": "[REDACTED]", "query": "What is AI?"}

// Check if a value is sensitive
isAPIKey := sanitizer.IsAPIKey("sk-abc123...")  // true
isPII := sanitizer.IsPII("john@example.com")    // true
```

**Patterns Detected:**
- **API Keys**: OpenAI keys (sk-...), generic API keys, bearer tokens, AWS access keys
- **Secrets**: Generic secret patterns, authorization headers
- **PII**: Email addresses, phone numbers (US format), credit card numbers, SSN (US format)

**Sensitive Field Names:**
- api_key, apikey, api-key
- secret, password, token
- authorization, auth
- access_key, secret_key

### 6. Comprehensive Alerting

25+ Prometheus alerting rules covering all aspects of gateway health.

**Implementation** (`prometheus/alerts.yml`):
- SLO violation alerts (6 rules)
- Cost anomaly alerts (3 rules)
- Job health alerts (4 rules)
- Provider availability alerts (3 rules)
- Cache performance alerts (2 rules)
- System health alerts (2 rules)

**Alert Groups:**

**SLO Alerts:**
- `HighErrorRate` - Error rate > 5% for 5 minutes
- `HighLatencyP95` - p95 latency > 5s for 5 minutes
- `SLOViolationLatencyP95` - p95 latency > 2s (SLO target)
- `SLOViolationLatencyP99` - p99 latency > 5s (SLO target)
- `SLOViolationErrorRate` - Error rate > 1% (SLO target)
- `SLOViolationAvailability` - Availability < 99.5% (SLO target)
- `ErrorBudgetExhausted` - Error budget completely exhausted
- `ErrorBudgetLow` - Error budget < 20% remaining

**Cost Alerts:**
- `CostSpike` - Actual cost > 2x baseline for 5 minutes
- `CostAnomaly` - Cost anomaly detected
- `HighDailyCost` - Daily cost > $100

**Job Alerts:**
- `JobQueueBackingUp` - Queue depth > 100 for 5 minutes
- `HighJobFailureRate` - Failure rate > 20% for 5 minutes
- `StuckJobsDetected` - Jobs running > 5 minutes
- `NoJobWorkersRunning` - No workers but queue has pending jobs

**Provider Alerts:**
- `ProviderUnavailable` - Provider unavailable for > 1 minute
- `CircuitBreakerOpen` - Circuit breaker opened for provider
- `HighProviderErrorRate` - Provider error rate > 10%

**Cache Alerts:**
- `LowCacheHitRate` - Cache hit rate < 10% for 10 minutes
- `CacheSavingsLow` - Cache savings < $0.10 in last hour

**System Alerts:**
- `HighRequestRate` - Request rate > 1000 req/s for 5 minutes
- `GatewayRestart` - Gateway restarted recently

## Upgrade Steps

### 1. Pull Latest Changes

```bash
git pull origin main
```

### 2. Update Dependencies

No new dependencies required for Phase 4.

### 3. Configure Prometheus Alerting

**Option A: Using Prometheus with Alertmanager**

```bash
# Copy alerts configuration
cp prometheus/alerts.yml /etc/prometheus/alerts.yml

# Update prometheus.yml to include alerts
cat >> /etc/prometheus/prometheus.yml <<EOF
rule_files:
  - "alerts.yml"

alerting:
  alertmanagers:
    - static_configs:
        - targets:
            - localhost:9093
EOF

# Reload Prometheus
curl -X POST http://localhost:9090/-/reload
```

**Option B: Using Docker Compose**

```yaml
# docker-compose.yml
version: '3.8'
services:
  prometheus:
    image: prom/prometheus:latest
    volumes:
      - ./prometheus/alerts.yml:/etc/prometheus/alerts.yml
      - ./prometheus/prometheus.yml:/etc/prometheus/prometheus.yml
    ports:
      - "9090:9090"
  
  alertmanager:
    image: prom/alertmanager:latest
    volumes:
      - ./prometheus/alertmanager.yml:/etc/alertmanager/alertmanager.yml
    ports:
      - "9093:9093"
```

### 4. Configure Environment Variables (Optional)

```bash
# SLO tracking window (default: 24 hours)
export SLO_WINDOW_HOURS=24

# Cost anomaly spike threshold (default: 2.0x)
export COST_SPIKE_THRESHOLD=2.0

# Job monitoring thresholds
export JOB_STUCK_THRESHOLD_MINUTES=5
export JOB_FAILURE_RATE_WINDOW_HOURS=1
export JOB_QUEUE_DEPTH_ALERT=100
```

### 5. Integrate SLO Tracking

```go
// In your main.go or initialization code
import (
    "github.com/amorin24/llmproxy/pkg/slo"
    "time"
)

// Create SLO tracker
sloTracker := slo.NewTracker(24 * time.Hour)

// In your request handler
func handleRequest(w http.ResponseWriter, r *http.Request) {
    start := time.Now()
    
    // ... process request ...
    
    latency := time.Since(start)
    success := err == nil
    
    // Record for SLO tracking
    sloTracker.RecordRequest(provider, latency, success)
}
```

### 6. Integrate Cost Anomaly Detection

```go
// In your main.go or initialization code
import (
    "github.com/amorin24/llmproxy/pkg/anomaly"
)

// Create anomaly detector
anomalyDetector := anomaly.NewDetector(2.0)

// After processing a request with cost
anomalyDetector.RecordCost(provider, costUSD, modelVersion)

// Periodically check for unusual usage
go func() {
    ticker := time.NewTicker(1 * time.Hour)
    for range ticker.C {
        alerts := anomalyDetector.CheckUnusualUsage()
        for _, alert := range alerts {
            logrus.Warn(alert)
        }
    }
}()
```

### 7. Integrate Job Monitoring

```go
// In your main.go or initialization code
import (
    "github.com/amorin24/llmproxy/pkg/jobmonitor"
)

// Create job monitor
jobMonitor := jobmonitor.NewMonitor(
    jobStore,
    5*time.Minute,
    1*time.Hour,
    100,
)

// Start monitoring
jobMonitor.Start()

// In your job worker
jobMonitor.RecordJobStart(jobID)
// ... process job ...
jobMonitor.RecordJobComplete(jobID, provider, success, duration)
```

### 8. Integrate Log Sanitization

```go
// In your logging setup
import (
    "github.com/amorin24/llmproxy/pkg/logging"
    "github.com/sirupsen/logrus"
)

// Create sanitizer
sanitizer := logging.NewSanitizer()

// Create custom logrus hook
type SanitizingHook struct {
    sanitizer *logging.Sanitizer
}

func (h *SanitizingHook) Levels() []logrus.Level {
    return logrus.AllLevels
}

func (h *SanitizingHook) Fire(entry *logrus.Entry) error {
    // Sanitize message
    entry.Message = h.sanitizer.Sanitize(entry.Message)
    
    // Sanitize fields
    entry.Data = h.sanitizer.SanitizeFields(entry.Data)
    
    return nil
}

// Add hook to logrus
logrus.AddHook(&SanitizingHook{sanitizer: sanitizer})
```

### 9. Restart the Application

```bash
# Stop the current instance
pkill -f llmproxy

# Start with new configuration
go run cmd/server/main.go
```

### 10. Verify Phase 4 Features

```bash
# Check SLO metrics
curl http://localhost:9090/api/v1/query?query=llmproxy_slo_error_rate

# Check cost anomaly metrics
curl http://localhost:9090/api/v1/query?query=llmproxy_cost_baseline_usd_per_hour

# Check job monitoring metrics
curl http://localhost:9090/api/v1/query?query=llmproxy_job_queue_depth

# Check alerts
curl http://localhost:9090/api/v1/alerts
```

## Configuration Changes

### New Environment Variables

**SLO Configuration:**
- `SLO_WINDOW_HOURS` - SLO tracking window in hours (default: 24)

**Cost Anomaly Detection:**
- `COST_SPIKE_THRESHOLD` - Multiplier for cost spike detection (default: 2.0)

**Job Monitoring:**
- `JOB_STUCK_THRESHOLD_MINUTES` - Minutes before job is considered stuck (default: 5)
- `JOB_FAILURE_RATE_WINDOW_HOURS` - Window for failure rate calculation (default: 1)
- `JOB_QUEUE_DEPTH_ALERT` - Queue depth threshold for alerts (default: 100)

### No Breaking Changes

All existing endpoints continue to work unchanged:
- `/api/query` - Standard query endpoint
- `/api/status` - Provider status
- `/v1/gateway/query` - Gateway query (Phase 0)
- `/v1/gateway/cost-estimate` - Cost estimation (Phase 1)
- `/v1/gateway/stream` - SSE streaming (Phase 3)
- `/v1/gateway/ws` - WebSocket streaming (Phase 3)
- `/v1/gateway/jobs` - Async jobs (Phase 3)

## Architecture

### SLO Tracking Architecture

```
Request → SLO Tracker
            ↓
       Record Metrics
            ↓
       Check Violations
            ↓
       Update Prometheus
            ↓
       Log Violations
```

### Cost Anomaly Detection Architecture

```
Cost Observation → Anomaly Detector
                      ↓
                 Update Baseline (EMA)
                      ↓
                 Check for Spike
                      ↓
                 Update Prometheus
                      ↓
                 Log Anomaly
```

### Job Monitoring Architecture

```
Job Lifecycle → Job Monitor
                   ↓
              Check Health
                   ↓
              Detect Issues
                   ↓
              Update Prometheus
                   ↓
              Log Alerts
```

### Trace Enrichment Architecture

```
Span Creation → Enrichment Functions
                    ↓
               Add Attributes
                    ↓
               Add Events
                    ↓
               Export Trace
```

## Monitoring & Observability

### SLO Dashboards

**Grafana Dashboard Queries:**

```promql
# SLO Compliance
llmproxy_slo_error_rate{provider="openai"}
llmproxy_slo_availability{provider="openai"}
llmproxy_slo_latency_p95_seconds{provider="openai"}
llmproxy_slo_latency_p99_seconds{provider="openai"}

# Error Budget
llmproxy_slo_error_budget_remaining{provider="openai"}

# SLO Violations
rate(llmproxy_slo_violations_total[1h])
```

### Cost Anomaly Dashboards

```promql
# Cost Baselines
llmproxy_cost_baseline_usd_per_hour{provider="openai"}
llmproxy_cost_actual_usd_per_hour{provider="openai"}

# Cost Anomalies
rate(llmproxy_cost_anomalies_total[1h])

# Model Usage
llmproxy_model_usage_requests_total{provider="openai", model_version="gpt-4o"}
```

### Job Monitoring Dashboards

```promql
# Queue Health
llmproxy_job_queue_depth
llmproxy_jobs_running
llmproxy_jobs_pending

# Job Success Rate
(
  rate(llmproxy_jobs_completed_total[5m])
  /
  (rate(llmproxy_jobs_completed_total[5m]) + rate(llmproxy_jobs_failed_total[5m]))
)

# Job Duration
histogram_quantile(0.95, rate(llmproxy_job_duration_seconds_bucket[5m]))

# Stuck Jobs
rate(llmproxy_jobs_stuck_total[5m])
```

## Rollback Plan

If you need to rollback Phase 4:

### Option 1: Disable New Features

Phase 4 features are additive and can be disabled by simply not using them:
- Don't initialize SLO tracker
- Don't initialize anomaly detector
- Don't initialize job monitor
- Don't add sanitizing hook to logrus
- Don't load Prometheus alerts

All existing functionality continues to work unchanged.

### Option 2: Revert to Phase 3

```bash
# Checkout Phase 3 commit
git checkout c9d251f

# Rebuild and restart
go build -o llmproxy cmd/server/main.go
./llmproxy
```

## What's Next: Phase 5

Phase 5 (Developer Experience and Ergonomics) will add:
- Complete OpenAPI 3.0 specification
- Generated SDKs (Go, TypeScript)
- Mock providers for local development
- CLI tool for testing
- Docker Compose setup
- Comprehensive documentation

Estimated timeline: 2 weeks

## Testing Recommendations

### Unit Tests

```bash
# Run all tests
go test ./...

# Test specific packages
go test ./pkg/slo
go test ./pkg/anomaly
go test ./pkg/jobmonitor
go test ./pkg/logging
```

### Integration Tests

**Test SLO Tracking:**
```bash
# Generate load with varying latencies
for i in {1..100}; do
  curl -X POST http://localhost:8080/v1/gateway/query \
    -H "Content-Type: application/json" \
    -d '{"query": "Test", "model": "openai"}'
  sleep 0.1
done

# Check SLO metrics
curl http://localhost:9090/api/v1/query?query=llmproxy_slo_error_rate
```

**Test Cost Anomaly Detection:**
```bash
# Generate requests with varying costs
# Check for anomaly detection in logs and Prometheus
curl http://localhost:9090/api/v1/query?query=llmproxy_cost_anomalies_total
```

**Test Job Monitoring:**
```bash
# Submit multiple jobs
for i in {1..50}; do
  curl -X POST http://localhost:8080/v1/gateway/jobs \
    -H "Content-Type: application/json" \
    -d '{"query": "Test", "model": "openai"}'
done

# Check job metrics
curl http://localhost:9090/api/v1/query?query=llmproxy_job_queue_depth
```

**Test Log Sanitization:**
```bash
# Check logs for redacted sensitive data
tail -f /var/log/llmproxy.log | grep REDACTED
```

### Load Tests

**Test SLO Tracking Under Load:**
```bash
ab -n 10000 -c 100 -p query.json -T application/json \
  http://localhost:8080/v1/gateway/query
```

**Test Alerting:**
```bash
# Trigger high error rate
# Trigger cost spike
# Trigger job queue backup
# Verify alerts fire in Prometheus/Alertmanager
```

## Known Limitations

1. **In-Memory SLO Tracking**: SLO metrics are stored in memory and will be lost on restart. Persistent storage will be added in Phase 6.

2. **Baseline Initialization**: Cost baselines require at least 10 samples before anomaly detection is effective.

3. **Log Sanitization Performance**: Sanitization adds overhead to logging. Consider log sampling for high-volume endpoints.

4. **Alert Fatigue**: With 25+ alerting rules, tune thresholds to avoid alert fatigue. Start with higher thresholds and adjust based on your environment.

5. **Trace Storage**: Enriched traces can be large. Configure trace sampling in production (e.g., 10% sampling rate).

6. **Job Monitor Overhead**: Job monitoring runs every 30 seconds. For very high job volumes, consider increasing the interval.

## Support

For issues or questions:
1. Check the main README.md for general setup
2. Review Phase 1 migration guide (MIGRATION_PHASE1.md) for cost tracking
3. Review Phase 2 migration guide (MIGRATION_PHASE2.md) for provider setup
4. Review Phase 3 migration guide (MIGRATION_PHASE3.md) for streaming and jobs
5. Review the gateway upgrade plan (docs/gateway-upgrade-plan.md)

## Changelog

### Phase 4 (Current)
- Added SLO definitions and tracking per provider
- Added cost anomaly detection with baseline tracking
- Added job monitoring with stuck job detection
- Added trace enrichment with cost, tokens, and operational attributes
- Added log hygiene with API key and PII redaction
- Added 25+ Prometheus alerting rules
- Added comprehensive observability and operational excellence

### Phase 3 (Previous)
- Added SSE streaming endpoint
- Added WebSocket streaming endpoint
- Added async job system
- Added circuit breaker pattern

### Phase 2 (Previous)
- Added Vertex AI provider
- Added Bedrock provider
- Extended router for new providers
- Integrated cost tracking

### Phase 1 (Previous)
- Price catalog system
- Cost estimation service
- Extended Prometheus metrics
- OpenTelemetry distributed tracing
- Grafana cost visibility dashboard

### Phase 0 (Foundation)
- RequestContext structure
- Versioned gateway API
- Bug fixes and Go 1.25 upgrade
