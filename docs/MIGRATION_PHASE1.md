# Migration Guide: Phase 1 Cost Visibility & Observability

This guide covers the changes introduced in Phase 1 of the LLM Gateway upgrade.

## Overview

Phase 1 implements comprehensive cost visibility and observability features:
- Price catalog with per-provider, per-model pricing
- Cost estimation service (pre-call and post-call)
- Prometheus metrics for cost tracking
- OpenTelemetry distributed tracing
- Grafana dashboards for cost visualization
- Integration with gateway API endpoints

## Breaking Changes

**None.** Phase 1 is fully backward compatible. All existing endpoints continue to work as before.

## New Features

### 1. Price Catalog System

The price catalog (`docs/price-catalog.json`) contains pricing for all LLM providers:
- OpenAI (gpt-4o, gpt-4-turbo, gpt-4, gpt-3.5-turbo, o3, o4-mini)
- Gemini (2.5 Flash/Pro, 2.0 Flash/Lite, 1.5 Flash/Pro)
- Mistral (Small, Medium, Large, Codestral)
- Claude (Haiku, Sonnet, Opus)
- Vertex AI (Gemini models via Vertex)
- Bedrock (Claude, Titan, Llama models)

**Pricing Structure:**
```json
{
  "providers": {
    "openai": {
      "gpt-4o": {
        "input_per_1k_tokens": 0.005,
        "output_per_1k_tokens": 0.015,
        "notes": "GPT-4 Optimized model"
      }
    }
  }
}
```

### 2. Cost Estimation Service

New `pkg/pricing` package provides cost estimation:

**Pre-call estimation:**
```go
import "github.com/amorin24/llmproxy/pkg/pricing"

catalogLoader, _ := pricing.NewCatalogLoader("docs/price-catalog.json")
estimator := pricing.NewCostEstimator(catalogLoader)

// Estimate cost before making a query
estimate, _ := estimator.EstimatePreCall("openai", "gpt-4o", 100, 50)
// estimate.EstimatedCostUSD = 0.00125
```

**Post-call cost calculation:**
```go
// Calculate actual cost after query completes
actualCost, _ := estimator.EstimatePostCall("openai", "gpt-4o", 120, 80)
// actualCost.EstimatedCostUSD = 0.0018
```

### 3. Enhanced Cost Estimate Endpoint

The `/v1/gateway/cost-estimate` endpoint now returns actual pricing data:

**Request:**
```bash
curl -X POST http://localhost:8080/v1/gateway/cost-estimate \
  -H "Content-Type: application/json" \
  -d '{
    "query": "Explain quantum computing in detail",
    "model": "openai",
    "model_version": "gpt-4o",
    "expected_response_tokens": 200
  }'
```

**Response:**
```json
{
  "model": "openai",
  "model_version": "gpt-4o",
  "input_tokens": 8,
  "output_tokens": 200,
  "estimated_cost_usd": 0.00304,
  "price_per_input_token": 0.005,
  "price_per_output_token": 0.015
}
```

### 4. Prometheus Metrics for Cost Tracking

New Prometheus metrics available at `/metrics`:

**Cost Metrics:**
- `llmproxy_cost_usd_total{provider, model}` - Total cost by provider and model
- `llmproxy_cost_per_request_usd{provider, model}` - Cost distribution per request
- `llmproxy_token_cost_usd_total{provider, model, token_type}` - Token cost by type (input/output)
- `llmproxy_cost_savings_from_cache_usd_total` - Total savings from cache hits
- `llmproxy_estimated_vs_actual_cost_ratio{provider, model}` - Estimation accuracy

**Example Queries:**
```promql
# Total cost per hour by provider
sum by (provider) (rate(llmproxy_cost_usd_total[1h]))

# Average cost per request
histogram_quantile(0.5, rate(llmproxy_cost_per_request_usd_bucket[5m]))

# Cost savings from caching
rate(llmproxy_cost_savings_from_cache_usd_total[1h])
```

### 5. OpenTelemetry Distributed Tracing

New `pkg/tracing` package provides distributed tracing:

**Usage:**
```go
import "github.com/amorin24/llmproxy/pkg/tracing"

// Initialize tracer (in main.go)
shutdown, _ := tracing.InitTracer("llmproxy")
defer shutdown(context.Background())

// Create spans
ctx, span := tracing.StartSpan(ctx, "operation_name",
    attribute.String("key", "value"))
defer span.End()

// Add attributes
tracing.AddSpanAttributes(span,
    attribute.Int("tokens", 100),
    attribute.Float64("cost", 0.001))

// Record errors
tracing.RecordError(span, err)
```

### 6. Grafana Dashboard

New cost visibility dashboard at `grafana/dashboards/cost_visibility_dashboard.json`:

**Panels:**
1. Total Cost by Provider (timeseries)
2. Total Cost by Model (timeseries)
3. Cost Per Request Distribution (heatmap)
4. Cost Savings from Cache (stat)
5. Token Cost by Type (timeseries)
6. Estimated vs Actual Cost Accuracy (timeseries)
7. Cumulative Cost Over Time (timeseries)

**Access:**
- URL: http://localhost:3000 (when running docker-compose)
- Default credentials: admin/admin
- Dashboard auto-provisioned on startup

## Upgrade Steps

### For Internal Users

1. **Pull Latest Changes:**
   ```bash
   git pull origin main
   ```

2. **Rebuild and Restart:**
   ```bash
   docker-compose down
   docker-compose up --build
   ```

3. **Verify Price Catalog:**
   ```bash
   # Check that price catalog exists
   cat docs/price-catalog.json
   ```

4. **Test Cost Estimation:**
   ```bash
   curl -X POST http://localhost:8080/v1/gateway/cost-estimate \
     -H "Content-Type: application/json" \
     -d '{"query": "Hello world", "model": "openai", "model_version": "gpt-4o"}'
   ```

5. **View Metrics:**
   ```bash
   # Check Prometheus metrics
   curl http://localhost:8080/metrics | grep llmproxy_cost
   ```

6. **Access Grafana Dashboard:**
   - Open http://localhost:3000
   - Login with admin/admin
   - Navigate to "LLM Gateway - Cost Visibility" dashboard

### For Developers

1. **Update Dependencies:**
   ```bash
   go mod download
   go mod tidy
   ```

2. **Review New Packages:**
   - `pkg/pricing`: Price catalog loader and cost estimator
   - `pkg/tracing`: OpenTelemetry distributed tracing
   - Updated `pkg/monitoring`: Extended Prometheus metrics
   - Updated `pkg/gateway/v1`: Cost-aware handlers

3. **Run Tests:**
   ```bash
   go test ./pkg/pricing/...
   go test ./pkg/tracing/...
   go test ./pkg/monitoring/...
   ```

## Configuration Changes

### New Environment Variables (Optional)

```bash
# Price catalog path (default: docs/price-catalog.json)
PRICE_CATALOG_PATH=docs/price-catalog.json

# Enable OpenTelemetry tracing (default: true)
OTEL_ENABLED=true

# Tracing service name (default: llmproxy)
OTEL_SERVICE_NAME=llmproxy
```

### No Changes Required

All existing environment variables continue to work as before.

## API Changes

### Enhanced `/v1/gateway/cost-estimate`

Now returns actual pricing data from the price catalog instead of placeholder values.

**Before (Phase 0):**
- Returned hardcoded sample values

**After (Phase 1):**
- Returns actual pricing from `docs/price-catalog.json`
- Estimates token counts from query text
- Supports custom `expected_response_tokens` parameter

### Future `/v1/gateway/query` Enhancement

The query endpoint will be enhanced in future commits to:
- Return actual `cost_usd` in responses
- Support `dry_run` mode for cost estimation
- Enforce `max_cost_usd` limits
- Record cost metrics to Prometheus

## Monitoring & Observability

### Prometheus Metrics

**Cost Tracking:**
```promql
# Total spend by provider (last hour)
sum by (provider) (increase(llmproxy_cost_usd_total[1h]))

# Most expensive models
topk(5, sum by (model) (rate(llmproxy_cost_usd_total[24h])))

# Cost per request percentiles
histogram_quantile(0.95, rate(llmproxy_cost_per_request_usd_bucket[5m]))
```

**Cost Optimization:**
```promql
# Cache hit savings
rate(llmproxy_cost_savings_from_cache_usd_total[1h])

# Estimation accuracy (should be close to 1.0)
histogram_quantile(0.5, rate(llmproxy_estimated_vs_actual_cost_ratio_bucket[5m]))
```

### Grafana Dashboards

The cost visibility dashboard provides:
- Real-time cost tracking by provider and model
- Cost distribution analysis
- Cache savings visualization
- Estimation accuracy monitoring
- Cumulative cost trends

### OpenTelemetry Traces

Distributed traces include:
- Request ID correlation
- Cost estimation spans
- Provider and model attributes
- Token count attributes
- Error recording

## Price Catalog Management

### Updating Prices

1. Edit `docs/price-catalog.json`
2. Update pricing for specific models
3. Update `last_updated` timestamp
4. Restart the service to reload catalog

**Example:**
```json
{
  "version": "1.0",
  "last_updated": "2025-11-01T00:00:00Z",
  "providers": {
    "openai": {
      "gpt-4o": {
        "input_per_1k_tokens": 0.006,
        "output_per_1k_tokens": 0.018
      }
    }
  }
}
```

### Adding New Models

1. Add model entry to appropriate provider section
2. Include `input_per_1k_tokens` and `output_per_1k_tokens`
3. Add descriptive `notes`
4. Update pricing sources if needed

## Rollback Plan

If you need to rollback to Phase 0:

```bash
# Rollback to Phase 0 commit
git checkout <phase-0-commit-hash>

# Rebuild
docker-compose down
docker-compose up --build
```

No data migration or cleanup required. The price catalog is a static configuration file.

## What's Next?

Phase 1 establishes cost visibility. Upcoming phases will add:

- **Phase 2** (3 weeks): Vertex AI and AWS Bedrock provider integration
- **Phase 3** (3 weeks): Streaming support (SSE/WebSocket) and async jobs
- **Phase 4** (2 weeks): Advanced observability with SLOs and alerting
- **Phase 5** (2 weeks): Developer experience improvements (CLI, SDKs)
- **Phase 6** (2 weeks): Semantic caching and cost-optimized routing
- **Phase 7** (2 weeks): Guardrails and controlled fallbacks

See `docs/gateway-upgrade-plan.md` for the complete roadmap.

## Testing Recommendations

After upgrading, test the following:

1. **Cost Estimation:**
   - Test `/v1/gateway/cost-estimate` with various models
   - Verify pricing matches `docs/price-catalog.json`
   - Test with different query lengths

2. **Prometheus Metrics:**
   - Verify cost metrics are exposed at `/metrics`
   - Check that metrics update correctly
   - Test metric labels (provider, model, token_type)

3. **Grafana Dashboard:**
   - Access dashboard at http://localhost:3000
   - Verify all panels load correctly
   - Check that data appears after making requests

4. **Tracing:**
   - Verify traces are generated for requests
   - Check that spans include cost attributes
   - Verify error recording works

## Known Limitations

Phase 1 limitations:

1. **Query endpoint cost tracking**: The `/v1/gateway/query` endpoint does not yet return actual costs or enforce cost limits. This will be implemented in a future commit.

2. **Token estimation**: Token counts are estimated using a simple heuristic (~4 characters per token). More accurate tokenization will be added in future phases.

3. **Tracing export**: Traces currently use stdout exporter. Production-ready exporters (Jaeger, Zipkin) will be added in Phase 4.

4. **Price catalog hot reload**: Catalog changes require service restart. Hot reload will be added in a future update.

These limitations will be addressed in subsequent phases and updates.

## Support

For questions or issues:
1. Check the main README.md for basic usage
2. Review the gateway upgrade plan in `docs/gateway-upgrade-plan.md`
3. Check Prometheus metrics for cost data
4. Review Grafana dashboards for visualization
5. Contact the development team
