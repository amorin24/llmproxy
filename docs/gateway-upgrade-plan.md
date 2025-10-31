# LLM Proxy to Gateway Upgrade Plan

## Executive Summary

This document outlines a phased approach to upgrade the current LLM Proxy system into a comprehensive LLM Gateway with enhanced cost visibility, observability, and developer experience.

### Key Priorities
1. **Cost Visibility** - Track and report costs per request, model, and provider
2. **Observability** - Comprehensive metrics, tracing, and monitoring
3. **Developer Experience** - Clear APIs, examples, and tooling
4. **Security & Advanced Routing** - Secondary priorities

### Scope
- **Target**: Internal use only (not SaaS)
- **Deployment**: Single region
- **Compliance**: None required initially
- **Multi-tenancy**: Not required (billing/chargeback excluded)
- **New Providers**: Vertex AI (priority), AWS Bedrock
- **New Capabilities**: Streaming (SSE/WebSocket), async jobs

---

## Current Architecture Overview

### Existing Components
- **Server**: Go-based HTTP server with Gorilla Mux
- **Providers**: OpenAI, Gemini, Mistral, Claude
- **Features**:
  - Basic routing with task-type awareness
  - In-memory caching
  - Rate limiting (per-client IP)
  - Fallback mechanisms
  - Prometheus metrics
  - Token tracking
  - Parallel query support
  - Web UI for testing

### Known Issues
- **Bug in `pkg/api/handlers.go:getEnvAsInt`**: Function lowercases the key string instead of reading the environment variable, preventing rate limit configuration via env vars
- **Dockerfile**: Uses Go 1.21, needs update to Go 1.25.3 (already updated in go.mod)
- **Limited cost visibility**: No cost tracking or reporting
- **Basic observability**: Limited metrics, no distributed tracing

---

## Target Gateway Architecture

### Core Architectural Changes

#### 1. Request Context Enhancement
Every request will carry a `RequestContext` object containing:
- `request_id` (UUID for tracing)
- `start_time` (for latency tracking)
- `tenant` (default: "internal")
- `budget_caps` (optional cost limits)
- `trace_context` (OpenTelemetry span context)

#### 2. Versioned API Structure
```
/v1/gateway/chat          - Chat completion (messages-based)
/v1/gateway/stream        - Streaming chat completion (SSE)
/v1/gateway/jobs          - Async job submission
/v1/gateway/jobs/{id}     - Job status and results
/v1/gateway/health        - Health check
/v1/gateway/status        - Provider availability
/v1/gateway/metrics       - Prometheus metrics endpoint
```

#### 3. Provider Interface Extension
```go
type Client interface {
    Query(ctx context.Context, query string, modelVersion string) (*QueryResult, error)
    QueryStream(ctx context.Context, query string, modelVersion string) (StreamReader, error)
    CheckAvailability() bool
    GetModelType() models.ModelType
    GetCapabilities() ProviderCapabilities
}

type ProviderCapabilities struct {
    SupportsStreaming   bool
    SupportsJSONSchema  bool
    MaxTokens           int
    SupportedRegions    []string
}

type StreamReader interface {
    Read() (token string, done bool, err error)
    Close() error
}
```

#### 4. Cost Model
```json
{
  "version": "1.0",
  "last_updated": "2025-10-31",
  "providers": {
    "openai": {
      "gpt-4o": {
        "input_per_1k_tokens": 0.005,
        "output_per_1k_tokens": 0.015
      }
    },
    "vertex_ai": {
      "gemini-2.0-flash": {
        "input_per_1k_tokens": 0.00025,
        "output_per_1k_tokens": 0.00075
      }
    }
  }
}
```

---

## Phased Implementation Plan

### Phase 0: Foundations and API Shape (Week 1-2)
**Goal**: Set a clean base for gateway work; define canonical API; fix obvious issues.

#### Deliverables
1. **Fix Critical Bug**
   - Fix `pkg/api/handlers.go:getEnvAsInt` to properly read environment variables
   - Add tests to verify env var parsing

2. **Versioned Gateway API**
   - Introduce `/v1/gateway` endpoints
   - Define canonical request model:
     ```json
     {
       "messages": [{"role": "user", "content": "..."}],
       "model": "openai",
       "model_version": "gpt-4o",
       "parameters": {
         "temperature": 0.7,
         "max_tokens": 1000
       },
       "stream": false,
       "max_cost_usd": 0.10
     }
     ```
   - Keep existing endpoints temporarily for backward compatibility

3. **Request Context Structure**
   - Create `RequestContext` type
   - Add request_id generation and propagation
   - Add timing instrumentation

4. **Update Build Configuration**
   - Update Dockerfile to use Go 1.25.3
   - Update docker-compose.yml
   - Document Go version requirement in README

5. **Documentation**
   - Migration guide for new endpoints
   - API request/response examples

#### Acceptance Criteria
- [ ] getEnvAsInt bug fixed and tested
- [ ] New /v1/gateway endpoints respond correctly
- [ ] RequestContext propagates through all handlers
- [ ] Docker builds with Go 1.25.3
- [ ] Documentation updated

---

### Phase 1: Cost Visibility and Core Observability (Week 3-4)
**Goal**: Make cost a first-class signal and deepen telemetry.

#### Deliverables
1. **Price Catalog**
   - Create `docs/price-catalog.json` with provider/model pricing
   - Load at startup with hot-reload capability
   - Version tracking for price changes

2. **Cost Accounting**
   - Pre-call cost estimation based on input tokens
   - Post-call actual cost calculation using returned token counts
   - Include cost in response payload:
     ```json
     {
       "response": "...",
       "cost": {
         "estimated_usd": 0.0045,
         "actual_usd": 0.0052,
         "input_tokens": 150,
         "output_tokens": 200,
         "total_tokens": 350
       }
     }
     ```
   - Add cost to structured logs

3. **Prometheus Metrics Expansion**
   - `llm_cost_usd_total{provider, model, route}` - Total cost counter
   - `llm_tokens_total{provider, model, direction}` - Token usage (input/output)
   - `llm_requests_total{provider, model, route, success}` - Request counter
   - `llm_latency_ms{provider, model}` - Latency histogram
   - `llm_cache_hit_total{provider, model}` - Cache hits
   - `llm_stream_tokens_total{provider, model}` - Streaming token counter

4. **OpenTelemetry Tracing**
   - Add OTel SDK and exporter
   - Create spans for:
     - Request ingress
     - Routing decision
     - Provider call
     - Cache lookup/store
     - Response serialization
   - Propagate request_id in span attributes
   - Add cost and token counts to span attributes

5. **Grafana Dashboards**
   - Cost by model/provider over time
   - Latency percentiles (p50, p95, p99)
   - Error rates by provider
   - Token usage trends
   - Cache hit rate
   - Cost savings from caching

#### Acceptance Criteria
- [ ] Price catalog loads and hot-reloads
- [ ] Cost calculated and returned in responses
- [ ] All new Prometheus metrics exposed
- [ ] Traces visible in Jaeger/Tempo
- [ ] Grafana dashboards created and functional
- [ ] Cost metrics accurate within 5%

---

### Phase 2: Vertex AI and Bedrock Providers (Week 5-7)
**Goal**: Add Vertex AI first, then Bedrock, with consistent interfaces and streaming.

#### Deliverables
1. **Vertex AI Client** (Priority)
   - Authenticate via Application Default Credentials (ADC) or service account
   - Support text/chat completion
   - Support streaming responses
   - Configurable region (single region per requirement)
   - Model mapping:
     - `gemini-2.0-flash`
     - `gemini-2.0-flash-lite`
     - `gemini-1.5-flash`
     - `gemini-1.5-pro`
   - Token accounting from API responses
   - Error handling and retries

2. **Bedrock Client**
   - AWS SDK v2 Bedrock Runtime
   - Support non-streaming and event-streaming responses
   - Region and model selection via config
   - Models:
     - Claude 3 (Haiku, Sonnet, Opus)
     - Titan models
     - Llama models
   - Token accounting (estimate if not provided)

3. **Unified Provider Interface Updates**
   - Add `GetCapabilities()` method to Client interface
   - Add `QueryStream()` method for streaming
   - Normalize token accounting across providers
   - Handle providers that don't return token counts

4. **Configuration Updates**
   - Add to `.env.example`:
     ```
     # Vertex AI
     VERTEX_AI_PROJECT_ID=your-project-id
     VERTEX_AI_LOCATION=us-central1
     VERTEX_AI_CREDENTIALS_PATH=/path/to/service-account.json
     
     # AWS Bedrock
     AWS_REGION=us-east-1
     AWS_ACCESS_KEY_ID=your-access-key
     AWS_SECRET_ACCESS_KEY=your-secret-key
     ```
   - Feature toggles for enabling providers

5. **Testing**
   - Unit tests with mocked provider responses
   - Integration tests with real providers (manual)
   - Health checks reflected in `/api/status`

6. **Price Catalog Updates**
   - Add Vertex AI pricing
   - Add Bedrock pricing
   - Document pricing sources and update frequency

#### Acceptance Criteria
- [ ] Vertex AI client functional with all models
- [ ] Bedrock client functional with key models
- [ ] Streaming works for both providers
- [ ] Token accounting accurate or estimated
- [ ] Health checks show provider status
- [ ] Price catalog includes new providers
- [ ] Tests pass with >80% coverage

---

### Phase 3: Streaming and Async Jobs (Week 8-10)
**Goal**: Provide streaming UX and support long-running operations.

#### Deliverables
1. **Server-Sent Events (SSE) Streaming**
   - Endpoint: `POST /v1/gateway/stream`
   - Stream incremental tokens as they arrive
   - Include cost accumulation in stream
   - Format:
     ```
     data: {"token": "Hello", "cost_usd": 0.0001}
     data: {"token": " world", "cost_usd": 0.0002}
     data: {"done": true, "total_cost_usd": 0.0052}
     ```
   - Backpressure handling
   - Early termination support (client disconnect)

2. **WebSocket Streaming** (Optional)
   - Endpoint: `WS /v1/gateway/ws`
   - Bidirectional communication
   - Support for multi-turn conversations
   - Connection lifecycle management

3. **Async Job System**
   - Job submission: `POST /v1/gateway/jobs`
     ```json
     {
       "messages": [...],
       "model": "openai",
       "callback_url": "https://internal.service/webhook"
     }
     ```
   - Response:
     ```json
     {
       "job_id": "uuid",
       "status": "pending",
       "estimated_cost_usd": 0.05
     }
     ```
   - Job status: `GET /v1/gateway/jobs/{id}`
   - Job results: `GET /v1/gateway/jobs/{id}/result`
   - In-memory job store with TTL (1 hour default)
   - Job lifecycle: pending → running → completed/failed
   - Optional webhook delivery on completion

4. **Concurrency Controls**
   - Per-request timeout configuration
   - Per-provider timeout configuration
   - Circuit breakers at provider level
   - Max concurrent requests per provider

5. **Developer Experience**
   - Code examples for streaming in README
   - Code examples for async jobs
   - Postman collection with streaming examples
   - CLI tool for testing streaming locally

#### Acceptance Criteria
- [ ] SSE streaming works with all providers
- [ ] WebSocket streaming functional (if implemented)
- [ ] Async jobs can be submitted and retrieved
- [ ] Job status updates correctly
- [ ] Webhooks deliver results reliably
- [ ] Circuit breakers prevent cascading failures
- [ ] Examples and documentation complete

---

### Phase 4: Advanced Observability and SLOs (Week 11-12)
**Goal**: Make it easy to operate and tune the gateway.

#### Deliverables
1. **Service Level Objectives (SLOs)**
   - Define SLOs per provider:
     - Latency: p95 < 2s, p99 < 5s
     - Error rate: < 1%
     - Availability: > 99.5%
   - Error budget tracking
   - Alerts when error budget exhausted

2. **Cost Anomaly Detection**
   - Baseline cost per hour/day
   - Alert on cost spikes (>2x baseline)
   - Alert on unusual model usage patterns

3. **Job Monitoring**
   - Alert on stuck jobs (running > 5 minutes)
   - Alert on high job failure rate
   - Dashboard for job queue depth and throughput

4. **Trace Enrichment**
   - Add cost to span attributes
   - Add token counts to span attributes
   - Add fallback information (if fallback occurred)
   - Add retry counts to span attributes
   - Add cache hit/miss to span attributes

5. **Log Hygiene**
   - Ensure no API keys leak in logs
   - Redact PII-like patterns by default
   - Structured logging with consistent fields
   - Log sampling for high-volume endpoints

6. **Alerting Rules**
   - High error rate (>5% for 5 minutes)
   - High latency (p95 > 5s for 5 minutes)
   - Cost spike (>2x baseline)
   - Provider unavailable (>1 minute)
   - Job queue backing up (>100 pending jobs)

#### Acceptance Criteria
- [ ] SLOs defined and tracked
- [ ] Alerts configured in Prometheus/Alertmanager
- [ ] Cost anomaly detection working
- [ ] Traces include all relevant attributes
- [ ] Logs are clean and structured
- [ ] No secrets in logs verified

---

### Phase 5: Developer Experience and Ergonomics (Week 13-14)
**Goal**: Smooth local dev, clear interfaces, and faster iteration.

#### Deliverables
1. **OpenAPI Specification**
   - Complete OpenAPI 3.0 spec for all endpoints
   - Request/response schemas
   - Error response schemas
   - Authentication schemes

2. **SDK Generation**
   - Generate Go SDK from OpenAPI spec
   - Generate TypeScript SDK from OpenAPI spec
   - Publish to internal package registry

3. **Local Development Tools**
   - Mock providers with deterministic responses
   - Fake token/cost calculations for testing
   - Docker Compose setup for local development
   - Seed data for testing

4. **CLI Tool**
   - Simple CLI for sending requests
   - Stream results to terminal
   - Inspect cost/metrics locally
   - Format: `llmgateway chat "Hello world" --model openai --stream`

5. **Prompt Templating**
   - Lightweight config-driven templating
   - Variable substitution
   - Example:
     ```yaml
     templates:
       summarize:
         prompt: "Summarize the following text:\n\n{{text}}\n\nSummary:"
         model: claude
         max_tokens: 200
     ```

6. **Dry-Run Mode**
   - Estimate cost without calling providers
   - Validate request format
   - Check quota/budget limits
   - Endpoint: `POST /v1/gateway/dry-run`

7. **Documentation**
   - Getting started guide
   - API reference (from OpenAPI)
   - Code examples for common use cases
   - Troubleshooting guide
   - Cost optimization tips

#### Acceptance Criteria
- [ ] OpenAPI spec complete and validated
- [ ] SDKs generated and functional
- [ ] Mock providers work for local testing
- [ ] CLI tool functional and documented
- [ ] Prompt templates work correctly
- [ ] Dry-run mode accurate
- [ ] Documentation comprehensive

---

### Phase 6: Caching, Dedup, and Cost-Optimized Routing (Week 15-16)
**Goal**: Save cost without adding multi-tenancy complexity.

#### Deliverables
1. **Request Coalescing (Single-Flight)**
   - Detect duplicate in-flight requests
   - Coalesce identical requests to single provider call
   - Distribute response to all waiting clients
   - Metrics: `llm_coalesced_requests_total`

2. **Enhanced Caching**
   - Structured cache keys:
     - Normalized prompt
     - Model family (not specific version)
     - Parameters (temperature, max_tokens)
   - Configurable TTL per model family
   - Cache size limits with LRU eviction
   - Safety checks to avoid caching sensitive content
   - Cache warming for common queries

3. **Semantic Caching** (Advanced)
   - Embedding-based similarity matching
   - Cache hits for similar (not identical) prompts
   - Configurable similarity threshold
   - Per-model-family embeddings
   - Note: Requires embedding model (can use Vertex AI)

4. **Cost-Aware Routing**
   - Simple policy: if `prefer_low_cost` flag set, route to cheaper models
   - Config-driven routing rules:
     ```yaml
     routing_policies:
       low_cost:
         prefer: [gemini, mistral, claude, openai]
         max_cost_per_request: 0.01
       high_quality:
         prefer: [openai, claude, gemini, mistral]
     ```
   - Apply policy based on request parameters
   - Fallback to higher-cost models if low-cost unavailable

5. **Cost Metrics**
   - `llm_cache_savings_usd_total` - Cost saved by cache hits
   - `llm_coalesced_savings_usd_total` - Cost saved by coalescing
   - `llm_routing_cost_optimized_total` - Requests routed to cheaper models

6. **Cache Management API**
   - `DELETE /v1/gateway/cache` - Clear cache
   - `GET /v1/gateway/cache/stats` - Cache statistics
   - `POST /v1/gateway/cache/warm` - Warm cache with common queries

#### Acceptance Criteria
- [ ] Request coalescing works correctly
- [ ] Cache hit rate >30% for typical workloads
- [ ] Semantic caching functional (if implemented)
- [ ] Cost-aware routing reduces costs by >20%
- [ ] Cost savings metrics accurate
- [ ] Cache management API functional

---

### Phase 7: Guardrails and Controlled Fallbacks (Week 17-18)
**Goal**: Increase reliability and quality while respecting cost targets.

#### Deliverables
1. **Circuit Breakers**
   - Per-provider circuit breakers
   - States: closed, open, half-open
   - Configurable thresholds:
     - Error rate: >50% in 1 minute → open
     - Timeout rate: >30% in 1 minute → open
   - Recovery: half-open after 30 seconds
   - Metrics: `llm_circuit_breaker_state{provider}`

2. **Advanced Retry Logic**
   - Exponential backoff with jitter
   - Per-error-type retry policies
   - Max retries configurable per provider
   - Retry budget to prevent retry storms

3. **Hedged Requests**
   - Send duplicate request if first is slow
   - Cancel slower request when first completes
   - Cost guard: only hedge if under budget
   - Configurable hedging delay (e.g., p95 latency)
   - Metrics: `llm_hedged_requests_total`

4. **Content Moderation Hooks** (Minimal)
   - Pre-request hook for input validation
   - Post-response hook for output validation
   - Pluggable moderation functions
   - Default: no-op (can be enabled later)
   - Example: block requests with PII patterns

5. **Weighted A/B Routing**
   - Route percentage of traffic to different models
   - Config-driven weights:
     ```yaml
     experiments:
       gpt4_vs_claude:
         enabled: true
         variants:
           - model: openai/gpt-4o
             weight: 50
           - model: claude/claude-3-opus
             weight: 50
     ```
   - Track metrics per variant
   - Automatic winner selection based on cost/quality

6. **Fallback Chains**
   - Define fallback sequences:
     ```yaml
     fallback_chains:
       default:
         - openai/gpt-4o
         - claude/claude-3-sonnet
         - gemini/gemini-2.0-flash
     ```
   - Try next in chain on failure
   - Cost budget enforcement across chain
   - Stop if budget exceeded

#### Acceptance Criteria
- [ ] Circuit breakers prevent cascading failures
- [ ] Retry logic reduces transient errors
- [ ] Hedged requests improve p99 latency
- [ ] Content moderation hooks functional
- [ ] A/B routing works correctly
- [ ] Fallback chains respect cost budgets

---

## Architecture Diagrams

### Current Proxy Architecture
```
┌─────────┐
│  Client │
└────┬────┘
     │
     ▼
┌─────────────────┐
│   HTTP Server   │
│  (Gorilla Mux)  │
└────┬────────────┘
     │
     ▼
┌─────────────────┐
│     Router      │
│  (Task-based)   │
└────┬────────────┘
     │
     ▼
┌─────────────────┐      ┌──────────┐
│  LLM Clients    │─────▶│  Cache   │
│ (4 providers)   │      └──────────┘
└────┬────────────┘
     │
     ▼
┌─────────────────┐
│   Providers     │
│ OpenAI, Gemini, │
│ Mistral, Claude │
└─────────────────┘
```

### Target Gateway Architecture
```
┌─────────┐
│  Client │
└────┬────┘
     │
     ▼
┌──────────────────────────────────────┐
│         Gateway API Layer            │
│  /v1/gateway/* (versioned endpoints) │
└────┬─────────────────────────────────┘
     │
     ▼
┌──────────────────────────────────────┐
│       Request Context Layer          │
│  (request_id, timing, budget, trace) │
└────┬─────────────────────────────────┘
     │
     ▼
┌──────────────────────────────────────┐
│      Authentication & Rate Limit     │
└────┬─────────────────────────────────┘
     │
     ▼
┌──────────────────────────────────────┐
│         Cost Estimation              │
│    (pre-call budget check)           │
└────┬─────────────────────────────────┘
     │
     ▼
┌──────────────────────────────────────┐
│    Cache & Request Coalescing        │
│  (semantic cache, single-flight)     │
└────┬─────────────────────────────────┘
     │
     ▼
┌──────────────────────────────────────┐
│      Policy-Driven Router            │
│  (cost-aware, A/B, fallback chains)  │
└────┬─────────────────────────────────┘
     │
     ▼
┌──────────────────────────────────────┐
│     Provider Orchestration           │
│  (circuit breakers, retries, hedge)  │
└────┬─────────────────────────────────┘
     │
     ▼
┌──────────────────────────────────────┐
│         LLM Clients (8+)             │
│  OpenAI, Gemini, Mistral, Claude,    │
│  Vertex AI, Bedrock, ...             │
└────┬─────────────────────────────────┘
     │
     ▼
┌──────────────────────────────────────┐
│      Streaming & Async Jobs          │
│    (SSE, WebSocket, job queue)       │
└────┬─────────────────────────────────┘
     │
     ▼
┌──────────────────────────────────────┐
│    Observability & Cost Tracking     │
│  (OTel traces, Prometheus, Grafana)  │
└──────────────────────────────────────┘
```

---

## Risk Assessment and Mitigations

### Technical Risks

1. **Streaming API Differences**
   - **Risk**: Vertex AI and Bedrock have different streaming semantics
   - **Mitigation**: Spike both providers early; create adapter layer to normalize streaming events

2. **Token Accounting Inconsistencies**
   - **Risk**: Providers report tokens differently (or not at all)
   - **Mitigation**: Prefer provider-reported tokens; fall back to estimation; mark estimates in logs/metrics

3. **Price Catalog Drift**
   - **Risk**: Provider pricing changes frequently
   - **Mitigation**: Version the catalog; add health check for missing prices; alert on unknown models

4. **Hedging Cost Explosion**
   - **Risk**: Hedged requests can double costs
   - **Mitigation**: Per-request max_cost guard; disable hedging when near budget; monitor hedging metrics

5. **Circuit Breaker False Positives**
   - **Risk**: Transient errors trigger circuit breaker unnecessarily
   - **Mitigation**: Tune thresholds carefully; implement half-open state; add manual override

### Operational Risks

1. **Job Store Memory Limits**
   - **Risk**: In-memory job store can exhaust memory
   - **Mitigation**: Set max job count; implement TTL; document Redis migration path

2. **Secrets in Logs**
   - **Risk**: API keys or sensitive data logged accidentally
   - **Mitigation**: Structured logging with redaction; audit logs regularly; add tests

3. **Go Version Mismatch**
   - **Risk**: Local dev uses different Go version than production
   - **Mitigation**: Update Dockerfile; document version requirement; add CI check

### Business Risks

1. **Cost Overruns**
   - **Risk**: Unexpected cost increases from new features
   - **Mitigation**: Cost alerts; budget enforcement; cost dashboards; dry-run mode

2. **Provider Outages**
   - **Risk**: Single provider outage impacts all users
   - **Mitigation**: Fallback chains; circuit breakers; multi-provider routing

---

## Success Metrics

### Phase 0-1 (Foundations & Cost Visibility)
- [ ] Cost tracked for 100% of requests
- [ ] Cost accuracy within 5% of actual
- [ ] All requests have trace IDs
- [ ] Grafana dashboards show cost trends

### Phase 2-3 (New Providers & Streaming)
- [ ] Vertex AI handles 30%+ of traffic
- [ ] Streaming latency <100ms per token
- [ ] Async jobs complete successfully >95%

### Phase 4-5 (Observability & DX)
- [ ] SLO compliance >99%
- [ ] Developer onboarding time <1 hour
- [ ] API documentation rated 4+/5 by users

### Phase 6-7 (Optimization & Reliability)
- [ ] Cache hit rate >30%
- [ ] Cost reduction >20% from optimizations
- [ ] Error rate <1%
- [ ] Circuit breakers prevent 100% of cascading failures

---

## Timeline Summary

| Phase | Duration | Key Deliverables |
|-------|----------|------------------|
| Phase 0 | 2 weeks | Bug fixes, versioned API, Go 1.25.3 |
| Phase 1 | 2 weeks | Cost tracking, OTel, Grafana dashboards |
| Phase 2 | 3 weeks | Vertex AI, Bedrock clients |
| Phase 3 | 3 weeks | Streaming, async jobs |
| Phase 4 | 2 weeks | SLOs, alerts, trace enrichment |
| Phase 5 | 2 weeks | OpenAPI, SDKs, CLI, docs |
| Phase 6 | 2 weeks | Caching, coalescing, cost routing |
| Phase 7 | 2 weeks | Circuit breakers, hedging, A/B |
| **Total** | **18 weeks** | **Full gateway capabilities** |

---

## Next Steps

1. **Review and Approve Plan**: Stakeholder review of this document
2. **Set Up Project Tracking**: Create tickets for each phase
3. **Provision Resources**: Set up Vertex AI and Bedrock accounts
4. **Kick Off Phase 0**: Begin with bug fixes and API design
5. **Weekly Reviews**: Track progress and adjust timeline as needed

---

## Appendix

### A. Environment Variables

See `.env.example` for complete list. Key additions:

```bash
# Cost Tracking
PRICE_CATALOG_PATH=./docs/price-catalog.json
COST_TRACKING_ENABLED=true

# OpenTelemetry
OTEL_EXPORTER_OTLP_ENDPOINT=http://localhost:4318
OTEL_SERVICE_NAME=llm-gateway

# Vertex AI
VERTEX_AI_PROJECT_ID=your-project
VERTEX_AI_LOCATION=us-central1
VERTEX_AI_CREDENTIALS_PATH=/path/to/creds.json

# AWS Bedrock
AWS_REGION=us-east-1
AWS_ACCESS_KEY_ID=your-key
AWS_SECRET_ACCESS_KEY=your-secret

# Streaming
STREAM_BUFFER_SIZE=100
STREAM_TIMEOUT_SECONDS=300

# Async Jobs
JOB_STORE_TTL_SECONDS=3600
JOB_MAX_CONCURRENT=50
```

### B. Sample Requests

#### Chat Completion
```bash
curl -X POST http://localhost:8080/v1/gateway/chat \
  -H "Content-Type: application/json" \
  -d '{
    "messages": [
      {"role": "user", "content": "Hello, world!"}
    ],
    "model": "openai",
    "model_version": "gpt-4o",
    "parameters": {
      "temperature": 0.7,
      "max_tokens": 100
    }
  }'
```

#### Streaming
```bash
curl -X POST http://localhost:8080/v1/gateway/stream \
  -H "Content-Type: application/json" \
  -d '{
    "messages": [
      {"role": "user", "content": "Tell me a story"}
    ],
    "model": "vertex_ai",
    "model_version": "gemini-2.0-flash"
  }'
```

#### Async Job
```bash
# Submit job
curl -X POST http://localhost:8080/v1/gateway/jobs \
  -H "Content-Type: application/json" \
  -d '{
    "messages": [
      {"role": "user", "content": "Analyze this large document..."}
    ],
    "model": "claude",
    "callback_url": "https://internal.service/webhook"
  }'

# Check status
curl http://localhost:8080/v1/gateway/jobs/{job_id}

# Get result
curl http://localhost:8080/v1/gateway/jobs/{job_id}/result
```

### C. References

- [OpenTelemetry Go SDK](https://opentelemetry.io/docs/instrumentation/go/)
- [Vertex AI Go Client](https://cloud.google.com/vertex-ai/docs/reference/go)
- [AWS Bedrock Go SDK](https://pkg.go.dev/github.com/aws/aws-sdk-go-v2/service/bedrockruntime)
- [Prometheus Best Practices](https://prometheus.io/docs/practices/naming/)
- [Server-Sent Events Spec](https://html.spec.whatwg.org/multipage/server-sent-events.html)

---

**Document Version**: 1.0  
**Last Updated**: October 31, 2025  
**Author**: Devin AI  
**Status**: Draft for Review
