# Migration Guide: Phase 0 Gateway Foundations

This guide covers the changes introduced in Phase 0 of the LLM Gateway upgrade.

## Overview

Phase 0 establishes the foundational components for the gateway upgrade:
- Bug fixes for critical issues
- Go version upgrade to 1.25
- New versioned API endpoints (`/v1/gateway/*`)
- RequestContext structure for request-scoped data
- Foundation for cost tracking and observability

## Breaking Changes

**None.** Phase 0 is fully backward compatible. All existing `/api/*` endpoints continue to work as before.

## New Features

### 1. Versioned Gateway API

New API endpoints under `/v1/gateway/` provide enhanced functionality:

#### Query Endpoint: `POST /v1/gateway/query`

Enhanced query endpoint with cost controls and detailed metrics.

**Request:**
```json
{
  "query": "Explain quantum computing",
  "model": "openai",
  "model_version": "gpt-4o",
  "task_type": "text_generation",
  "max_cost_usd": 0.10,
  "dry_run": false,
  "tenant": "internal",
  "request_id": "optional-custom-id"
}
```

**Response:**
```json
{
  "request_id": "550e8400-e29b-41d4-a716-446655440000",
  "response": "Quantum computing is...",
  "model": "openai",
  "model_version": "gpt-4o",
  "cached": false,
  "response_time_ms": 1234,
  "num_tokens": 150,
  "cost_usd": 0.0045,
  "tenant": "internal"
}
```

**New Fields:**
- `max_cost_usd`: Optional cost limit for the request
- `dry_run`: If true, returns cost estimate without executing
- `tenant`: Tenant identifier (defaults to "internal")
- `cost_usd`: Estimated cost of the request (Phase 1+)

#### Cost Estimate Endpoint: `POST /v1/gateway/cost-estimate`

Estimate the cost of a query before execution.

**Request:**
```json
{
  "query": "Explain quantum computing",
  "model": "openai",
  "model_version": "gpt-4o",
  "expected_response_tokens": 150
}
```

**Response:**
```json
{
  "model": "openai",
  "model_version": "gpt-4o",
  "input_tokens": 4,
  "output_tokens": 150,
  "estimated_cost_usd": 0.0045,
  "price_per_input_token": 0.005,
  "price_per_output_token": 0.015
}
```

### 2. RequestContext Structure

New `pkg/context` package provides request-scoped context:

```go
import "github.com/amorin24/llmproxy/pkg/context"

// Create a new request context
reqCtx := context.NewRequestContext(r.Context())

// With custom request ID
reqCtx := context.NewRequestContextWithID(r.Context(), "custom-id")

// Set optional fields
reqCtx.WithMaxCost(0.10).WithTenant("team-alpha")

// Access fields
requestID := reqCtx.RequestID
elapsed := reqCtx.ElapsedMilliseconds()
```

### 3. Bug Fixes

**Fixed:** `getEnvAsInt` function in `pkg/api/handlers.go` was incorrectly lowercasing the key string instead of reading the environment variable. This prevented rate limiting configuration via environment variables.

**Impact:** Rate limiting can now be properly configured using `RATE_LIMIT_PER_MINUTE` and `RATE_LIMIT_BURST` environment variables.

## Upgrade Steps

### For Internal Users

1. **Update Go Version** (if building locally):
   ```bash
   # Verify Go version
   go version  # Should be 1.25 or higher
   
   # If needed, download Go 1.25+
   # https://go.dev/dl/
   ```

2. **Pull Latest Changes**:
   ```bash
   git pull origin main
   ```

3. **Rebuild** (if running locally):
   ```bash
   go build -o llmproxy ./cmd/server
   ./llmproxy
   ```

4. **Rebuild Docker** (if using Docker):
   ```bash
   docker-compose down
   docker-compose up --build
   ```

5. **Test Legacy Endpoints** (optional):
   ```bash
   # Verify existing endpoints still work
   curl -X POST http://localhost:8080/api/query \
     -H "Content-Type: application/json" \
     -d '{"query": "Hello", "model": "openai", "task_type": "text_generation"}'
   ```

6. **Test New Gateway Endpoints** (optional):
   ```bash
   # Test new gateway query endpoint
   curl -X POST http://localhost:8080/v1/gateway/query \
     -H "Content-Type: application/json" \
     -d '{"query": "Hello", "model": "openai", "task_type": "text_generation"}'
   
   # Test cost estimate endpoint
   curl -X POST http://localhost:8080/v1/gateway/cost-estimate \
     -H "Content-Type: application/json" \
     -d '{"query": "Hello", "model": "openai"}'
   ```

### For Developers

1. **Update Dependencies**:
   ```bash
   go mod download
   go mod tidy
   ```

2. **Run Tests**:
   ```bash
   go test ./...
   ```

3. **Review New Packages**:
   - `pkg/context`: Request context management
   - `pkg/gateway/v1`: Versioned gateway API types and handlers

4. **Update Integrations** (if using gateway programmatically):
   - Legacy `/api/*` endpoints remain unchanged
   - New `/v1/gateway/*` endpoints available for enhanced features
   - No changes required for existing integrations

## Configuration Changes

### New Environment Variables (Optional)

These variables were previously broken due to the `getEnvAsInt` bug and now work correctly:

```bash
# Rate limiting (now functional)
RATE_LIMIT_PER_MINUTE=60
RATE_LIMIT_BURST=10
```

### No Changes Required

All existing environment variables continue to work as before:
- `OPENAI_API_KEY`
- `GEMINI_API_KEY`
- `MISTRAL_API_KEY`
- `CLAUDE_API_KEY`
- `PORT`
- `LOG_LEVEL`
- `CACHE_ENABLED`
- `CACHE_TTL`
- `MAX_RETRIES`
- etc.

## Rollback Plan

If you need to rollback to the previous version:

```bash
# Rollback to previous commit
git checkout <previous-commit-hash>

# Rebuild
docker-compose down
docker-compose up --build
```

No data migration or cleanup is required as Phase 0 introduces no database or state changes.

## What's Next?

Phase 0 establishes the foundation. Upcoming phases will add:

- **Phase 1** (2 weeks): Cost visibility with price catalog and Prometheus metrics
- **Phase 2** (3 weeks): Vertex AI and AWS Bedrock provider integration
- **Phase 3** (3 weeks): Streaming support (SSE/WebSocket) and async jobs
- **Phase 4** (2 weeks): Advanced observability with OpenTelemetry and SLOs
- **Phase 5** (2 weeks): Developer experience improvements (CLI, SDKs, docs)
- **Phase 6** (2 weeks): Semantic caching and cost-optimized routing
- **Phase 7** (2 weeks): Guardrails and controlled fallbacks

See `docs/gateway-upgrade-plan.md` for the complete roadmap.

## Support

For questions or issues:
1. Check the main README.md for basic usage
2. Review the gateway upgrade plan in `docs/gateway-upgrade-plan.md`
3. Contact the development team

## Testing Recommendations

After upgrading, test the following:

1. **Legacy API compatibility**:
   - Verify existing `/api/query` requests work unchanged
   - Check `/api/status` endpoint functionality

2. **New gateway endpoints**:
   - Test `/v1/gateway/query` with various models
   - Test `/v1/gateway/cost-estimate` endpoint
   - Verify request ID tracking

3. **Rate limiting** (if configured):
   - Verify rate limits are enforced correctly
   - Test that `RATE_LIMIT_PER_MINUTE` and `RATE_LIMIT_BURST` work

4. **Error handling**:
   - Test invalid requests return proper error responses
   - Verify error responses include request IDs

## Known Limitations

Phase 0 is a foundation release with placeholder implementations:

1. **Cost tracking**: The `/v1/gateway/query` endpoint returns placeholder cost values. Full cost tracking will be implemented in Phase 1.

2. **Cost estimation**: The `/v1/gateway/cost-estimate` endpoint returns sample values. Actual price catalog integration comes in Phase 1.

3. **Dry run mode**: The `dry_run` parameter is accepted but not fully implemented. Full support comes in Phase 1.

These limitations are intentional and will be addressed in subsequent phases.
