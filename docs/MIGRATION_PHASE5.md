# Phase 5: Developer Experience & Ergonomics - Migration Guide

## Overview

Phase 5 focuses on improving developer experience and making the LLM Gateway easier to use, test, and integrate. This phase introduces tools and features that streamline local development, API integration, and testing workflows.

**Timeline**: Week 13-14 (2 weeks)

**Status**: ✅ Complete

## What's New in Phase 5

### 1. OpenAPI 3.0 Specification
Complete API specification for all gateway endpoints with request/response schemas, error handling, and authentication schemes.

**Location**: `api/openapi.yaml`

**Features**:
- Complete endpoint documentation for all `/api/*` and `/v1/gateway/*` endpoints
- Request/response schemas with validation rules
- Error response schemas with error codes
- Authentication schemes
- Example requests and responses
- Support for all 6 providers (OpenAI, Gemini, Mistral, Claude, Vertex AI, Bedrock)

**Usage**:
```bash
# View the OpenAPI spec
cat api/openapi.yaml

# Generate API documentation (using openapi-generator or similar)
openapi-generator-cli generate -i api/openapi.yaml -g html2 -o docs/api

# Generate SDKs
openapi-generator-cli generate -i api/openapi.yaml -g go -o sdk/go
openapi-generator-cli generate -i api/openapi.yaml -g typescript-fetch -o sdk/typescript
```

### 2. Mock Providers for Local Development
Deterministic mock LLM providers for testing without real API keys or costs.

**Location**: `pkg/mock/provider.go`

**Features**:
- Mock providers for all 6 model types
- Deterministic mode for consistent responses
- Configurable latency simulation
- Configurable failure rate for testing error handling
- Fake token and cost calculations
- Request counting and statistics

**Usage**:
```go
import "github.com/amorin24/llmproxy/pkg/mock"

// Create mock factory in deterministic mode
factory := mock.NewMockFactory(true)

// Get a mock provider
provider, _ := factory.GetProvider(models.OpenAI)

// Query the mock provider
result, err := provider.Query(ctx, "Hello world", "gpt-4o")
// Returns: "[MOCK OpenAI] Hello! I'm a mock LLM provider..."

// Configure latency for all providers
factory.ConfigureLatency(200) // 200ms latency

// Configure failure rate (10% failures)
factory.ConfigureFailureRate(0.1)

// Get statistics
stats := factory.GetStats()
fmt.Printf("OpenAI requests: %d\n", stats[models.OpenAI])
```

**Environment Variable**:
```bash
# Enable mock mode (no real API keys needed)
MOCK_MODE=true
```

### 3. CLI Tool
Command-line interface for testing the gateway with support for chat, streaming, cost estimation, and job management.

**Location**: `cmd/cli/main.go`

**Installation**:
```bash
# Build the CLI
go build -o llmgateway ./cmd/cli

# Or install globally
go install ./cmd/cli
```

**Commands**:

#### Chat Command
```bash
# Basic chat
llmgateway chat "What is machine learning?" --model openai

# With streaming
llmgateway chat "Explain quantum computing" --model claude --stream

# Dry-run mode (estimate cost only)
llmgateway chat "Summarize this text" --model gemini --dry-run

# Show detailed metrics
llmgateway chat "Hello world" --model openai --show-metrics

# Custom gateway URL
llmgateway chat "Test query" --url http://localhost:8080 --model mistral
```

#### Cost Estimation Command
```bash
# Estimate cost before running
llmgateway cost "Long query text here..." --model openai

# Different model
llmgateway cost "Query text" --model vertexai
```

#### Status Command
```bash
# Check provider availability
llmgateway status

# Output:
# 🏥 Provider Status:
#    openai: ✅ Available
#    gemini: ✅ Available
#    mistral: ✅ Available
#    claude: ✅ Available
#    vertexai: ✅ Available
#    bedrock: ✅ Available
```

#### Job Commands
```bash
# Submit async job
llmgateway job submit "Long running query" --model openai
# Output: Job ID: 550e8400-e29b-41d4-a716-446655440000

# Check job status
llmgateway job status 550e8400-e29b-41d4-a716-446655440000

# Get job result
llmgateway job result 550e8400-e29b-41d4-a716-446655440000
```

**CLI Flags**:
- `--url`: Gateway URL (default: http://localhost:8080)
- `--model`: LLM model to use (openai, gemini, mistral, claude, vertexai, bedrock)
- `--stream`: Enable streaming mode
- `--dry-run`: Dry-run mode (estimate cost only)
- `--show-cost`: Show cost information (default: true)
- `--show-metrics`: Show detailed metrics

### 4. Prompt Templating System
Config-driven prompt templating with variable substitution for common use cases.

**Location**: `pkg/templates/engine.go`, `templates/examples.yaml`

**Features**:
- YAML-based template configuration
- Variable substitution with `{{variable}}` syntax
- Template validation
- Default templates for common tasks
- Model and parameter configuration per template

**Built-in Templates**:
1. **summarize** - Summarize text
2. **translate** - Translate between languages
3. **explain_code** - Explain code in any language
4. **qa** - Question answering with context
5. **creative_writing** - Generate creative content
6. **email** - Generate professional emails
7. **extract_data** - Extract structured data
8. **sentiment** - Sentiment analysis
9. **meeting_notes** - Generate meeting notes
10. **code_review** - Perform code reviews
11. **sql_generation** - Generate SQL queries
12. **product_description** - Write product descriptions

**Usage**:
```go
import "github.com/amorin24/llmproxy/pkg/templates"

// Create engine with default templates
engine := templates.DefaultTemplates()

// Or load from YAML file
engine := templates.NewEngine()
yamlData, _ := os.ReadFile("templates/examples.yaml")
engine.LoadFromYAML(yamlData)

// List available templates
templates := engine.ListTemplates()
// ["summarize", "translate", "explain_code", ...]

// Render a template
variables := map[string]string{
    "text": "Long article text here...",
}
prompt, _ := engine.Render("summarize", variables)
// Output: "Summarize the following text:\n\nLong article text here...\n\nSummary:"

// Validate template variables
err := engine.ValidateTemplate("translate", map[string]string{
    "text": "Hello",
    "source_lang": "English",
    // Missing: target_lang
})
// Returns error: "missing required variables: target_lang"

// Render with request context
req := templates.TemplateRequest{
    TemplateName: "summarize",
    Variables: map[string]string{
        "text": "Article text...",
    },
}
response, _ := engine.RenderRequest(req)
// response.Prompt: rendered prompt
// response.Model: claude (from template config)
// response.MaxTokens: 200 (from template config)
```

**Template YAML Format**:
```yaml
templates:
  my_template:
    prompt: "Do something with {{variable1}} and {{variable2}}"
    model: openai
    max_tokens: 300
    temperature: 0.7
    description: "Description of what this template does"
```

**Creating Custom Templates**:
```go
// Add a custom template programmatically
engine.AddTemplate("custom", templates.Template{
    Prompt:      "Custom prompt with {{var1}} and {{var2}}",
    Model:       models.OpenAI,
    MaxTokens:   500,
    Temperature: 0.8,
    Description: "My custom template",
})
```

### 5. Dry-Run Mode Endpoint
Validate requests and estimate costs without calling providers.

**Endpoint**: `POST /v1/gateway/dry-run`

**Features**:
- Request validation
- Cost estimation
- Budget limit checking
- No actual provider calls
- Fast response time

**Request**:
```json
{
  "query": "What is machine learning?",
  "model": "openai",
  "task_type": "chat",
  "max_cost_usd": 0.10
}
```

**Response**:
```json
{
  "valid": true,
  "estimated_cost_usd": 0.000234,
  "input_tokens": 5,
  "output_tokens": 100,
  "provider": "openai",
  "model_version": "gpt-4o",
  "within_budget": true,
  "errors": []
}
```

**Error Response** (invalid request):
```json
{
  "valid": false,
  "estimated_cost_usd": 0,
  "input_tokens": 0,
  "output_tokens": 0,
  "provider": "openai",
  "model_version": "",
  "within_budget": true,
  "errors": [
    "query cannot be empty",
    "task_type is required"
  ]
}
```

**Usage with CLI**:
```bash
llmgateway chat "Test query" --model openai --dry-run
```

**Usage with curl**:
```bash
curl -X POST http://localhost:8080/v1/gateway/dry-run \
  -H "Content-Type: application/json" \
  -d '{
    "query": "What is AI?",
    "model": "openai",
    "task_type": "chat",
    "max_cost_usd": 0.05
  }'
```

### 6. Docker Compose Setup for Local Development
Complete local development environment with all observability tools.

**Location**: `docker-compose.yml`

**Services**:
1. **llmproxy** - Main gateway service (port 8080)
2. **prometheus** - Metrics collection (port 9090)
3. **alertmanager** - Alert management (port 9093)
4. **grafana** - Dashboards and visualization (port 3000)
5. **jaeger** - Distributed tracing (port 16686)

**Quick Start**:
```bash
# Copy environment file
cp .env.example .env

# Edit .env with your API keys (or use MOCK_MODE=true)
nano .env

# Start all services
docker-compose up -d

# View logs
docker-compose logs -f llmproxy

# Stop all services
docker-compose down

# Stop and remove volumes
docker-compose down -v
```

**Service URLs**:
- Gateway API: http://localhost:8080
- Prometheus: http://localhost:9090
- Alertmanager: http://localhost:9093
- Grafana: http://localhost:3000 (admin/admin)
- Jaeger UI: http://localhost:16686

**Mock Mode for Local Development**:
```bash
# In .env file
MOCK_MODE=true

# No real API keys needed!
docker-compose up -d
```

**Health Checks**:
All services include health checks for reliability:
```bash
# Check service health
docker-compose ps

# Should show all services as "healthy"
```

**Volumes**:
- `prometheus-data` - Prometheus time-series data
- `alertmanager-data` - Alertmanager state
- `grafana-data` - Grafana dashboards and settings

**Networks**:
All services run on the `llm-gateway-network` bridge network for inter-service communication.

### 7. Enhanced Configuration
Updated environment configuration with all Phase 0-5 features.

**Location**: `.env.example`

**New Variables**:
```bash
# Mock Mode (Phase 5)
MOCK_MODE=false

# Vertex AI (Phase 2)
VERTEX_AI_API_KEY=...
VERTEX_AI_PROJECT_ID=...
VERTEX_AI_LOCATION=us-central1

# AWS Bedrock (Phase 2)
AWS_ACCESS_KEY_ID=...
AWS_SECRET_ACCESS_KEY=...
AWS_REGION=us-east-1

# Job Configuration (Phase 3)
JOB_TTL_SECONDS=3600
MAX_JOB_WORKERS=10

# Circuit Breaker (Phase 3)
CIRCUIT_BREAKER_MAX_FAILURES=5
CIRCUIT_BREAKER_TIMEOUT_SECONDS=60

# SLO Configuration (Phase 4)
SLO_WINDOW_HOURS=24

# Cost Anomaly Detection (Phase 4)
COST_SPIKE_THRESHOLD=2.0

# Job Monitoring (Phase 4)
JOB_STUCK_THRESHOLD_MINUTES=5
JOB_FAILURE_RATE_WINDOW_HOURS=1
JOB_QUEUE_DEPTH_ALERT=100

# Observability (Phase 1, 4)
OTEL_EXPORTER_OTLP_ENDPOINT=http://localhost:4318
OTEL_SERVICE_NAME=llm-gateway
```

## Breaking Changes

**None** - Phase 5 is fully backward compatible. All existing endpoints and functionality remain unchanged.

## New Features Summary

| Feature | Description | Location |
|---------|-------------|----------|
| OpenAPI Spec | Complete API documentation | `api/openapi.yaml` |
| Mock Providers | Testing without real API keys | `pkg/mock/provider.go` |
| CLI Tool | Command-line interface | `cmd/cli/main.go` |
| Prompt Templates | Config-driven templating | `pkg/templates/engine.go` |
| Dry-Run Mode | Cost estimation endpoint | `POST /v1/gateway/dry-run` |
| Docker Compose | Local dev environment | `docker-compose.yml` |
| Enhanced Config | Updated environment vars | `.env.example` |

## Upgrade Steps

### 1. Update Environment Configuration

```bash
# Copy new environment template
cp .env.example .env

# Add new Phase 5 variables
nano .env
```

### 2. Install CLI Tool (Optional)

```bash
# Build CLI
go build -o llmgateway ./cmd/cli

# Test CLI
./llmgateway status
```

### 3. Start Local Development Environment

```bash
# Start all services with Docker Compose
docker-compose up -d

# Verify all services are healthy
docker-compose ps

# Access services:
# - Gateway: http://localhost:8080
# - Grafana: http://localhost:3000
# - Prometheus: http://localhost:9090
# - Jaeger: http://localhost:16686
```

### 4. Test Mock Mode (Optional)

```bash
# Enable mock mode in .env
MOCK_MODE=true

# Restart gateway
docker-compose restart llmproxy

# Test with CLI (no real API keys needed)
./llmgateway chat "Hello world" --model openai
```

### 5. Explore Prompt Templates

```bash
# View example templates
cat templates/examples.yaml

# Use templates in your code
# See "Prompt Templating System" section above
```

### 6. Test Dry-Run Mode

```bash
# Using CLI
./llmgateway chat "Test query" --model openai --dry-run

# Using curl
curl -X POST http://localhost:8080/v1/gateway/dry-run \
  -H "Content-Type: application/json" \
  -d '{"query": "Test", "model": "openai", "task_type": "chat"}'
```

## Configuration Changes

### Docker Compose Enhancements

**New Services**:
- Alertmanager for alert management
- Jaeger for distributed tracing

**New Volumes**:
- `prometheus-data` - Persistent metrics storage
- `alertmanager-data` - Persistent alert state
- `grafana-data` - Persistent dashboard configuration

**New Environment Variables**:
- `MOCK_MODE` - Enable mock providers
- `OTEL_EXPORTER_OTLP_ENDPOINT` - OpenTelemetry endpoint
- All Phase 2-4 configuration variables

### Prometheus Configuration

**New Features**:
- Alertmanager integration
- Alert rules from `prometheus/alerts.yml`
- External labels for environment tracking

### Grafana Configuration

**New Features**:
- Automatic datasource provisioning
- Automatic dashboard provisioning
- Pre-configured Prometheus datasource

## Architecture

### Phase 5 Component Diagram

```
┌─────────────────────────────────────────────────────────────┐
│                     Developer Tools                          │
├─────────────────────────────────────────────────────────────┤
│                                                               │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐      │
│  │   CLI Tool   │  │   OpenAPI    │  │   Mock       │      │
│  │              │  │   Spec       │  │   Providers  │      │
│  │  - Chat      │  │              │  │              │      │
│  │  - Stream    │  │  - Schemas   │  │  - Determ.   │      │
│  │  - Cost      │  │  - Docs      │  │  - Latency   │      │
│  │  - Jobs      │  │  - SDK Gen   │  │  - Failures  │      │
│  └──────────────┘  └──────────────┘  └──────────────┘      │
│                                                               │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐      │
│  │   Prompt     │  │   Dry-Run    │  │   Docker     │      │
│  │   Templates  │  │   Mode       │  │   Compose    │      │
│  │              │  │              │  │              │      │
│  │  - YAML      │  │  - Validate  │  │  - Gateway   │      │
│  │  - Variables │  │  - Estimate  │  │  - Observ.   │      │
│  │  - Defaults  │  │  - Budget    │  │  - Network   │      │
│  └──────────────┘  └──────────────┘  └──────────────┘      │
│                                                               │
└─────────────────────────────────────────────────────────────┘
                            │
                            ▼
┌─────────────────────────────────────────────────────────────┐
│                   LLM Gateway (Phase 0-4)                    │
├─────────────────────────────────────────────────────────────┤
│  - Versioned API (/v1/gateway/*)                             │
│  - Cost Tracking & Estimation                                │
│  - Multi-Provider Support (6 providers)                      │
│  - Streaming (SSE, WebSocket)                                │
│  - Async Jobs                                                │
│  - Circuit Breakers                                          │
│  - SLO Tracking                                              │
│  - Cost Anomaly Detection                                    │
│  - Observability (Metrics, Traces, Logs)                     │
└─────────────────────────────────────────────────────────────┘
```

## Testing Recommendations

### Unit Tests

```bash
# Test mock providers
go test ./pkg/mock/... -v

# Test template engine
go test ./pkg/templates/... -v

# Test dry-run handler
go test ./pkg/gateway/v1/... -v -run TestDryRunHandler
```

### Integration Tests

```bash
# Start local environment
docker-compose up -d

# Wait for services to be healthy
sleep 30

# Test with CLI
./llmgateway status
./llmgateway chat "Test" --model openai --dry-run
./llmgateway cost "Test query" --model gemini

# Test dry-run endpoint
curl -X POST http://localhost:8080/v1/gateway/dry-run \
  -H "Content-Type: application/json" \
  -d '{"query": "Test", "model": "openai", "task_type": "chat"}'

# Test mock mode
MOCK_MODE=true docker-compose restart llmproxy
./llmgateway chat "Hello" --model openai
```

### Load Tests

```bash
# Test dry-run performance (no provider calls)
ab -n 1000 -c 10 -p dryrun.json -T application/json \
  http://localhost:8080/v1/gateway/dry-run

# Test mock provider performance
MOCK_MODE=true docker-compose restart llmproxy
ab -n 1000 -c 10 -p query.json -T application/json \
  http://localhost:8080/v1/gateway/query
```

## Monitoring & Observability

### Metrics

Phase 5 doesn't add new metrics but enhances the development experience for monitoring existing metrics:

**Grafana Dashboards**:
- Access: http://localhost:3000 (admin/admin)
- Pre-configured Prometheus datasource
- Import Phase 1 cost visibility dashboard

**Prometheus Queries**:
- Access: http://localhost:9090
- All Phase 1-4 metrics available
- Alert rules from Phase 4 active

### Tracing

**Jaeger UI**:
- Access: http://localhost:16686
- View distributed traces
- Analyze request flows
- Identify performance bottlenecks

**Dry-Run Traces**:
Dry-run requests are traced with span name `gateway.dry_run` and include:
- Provider selection
- Cost estimation
- Budget validation
- Validation errors

### Logs

**View Gateway Logs**:
```bash
# Follow logs
docker-compose logs -f llmproxy

# View last 100 lines
docker-compose logs --tail=100 llmproxy

# Search logs
docker-compose logs llmproxy | grep "dry-run"
```

## Rollback Plan

If you need to rollback Phase 5 changes:

### 1. Revert Code Changes

```bash
# Checkout previous version
git checkout main

# Or revert specific files
git checkout main -- docker-compose.yml
git checkout main -- .env.example
git checkout main -- prometheus/prometheus.yml
```

### 2. Remove Phase 5 Files

```bash
# Remove new directories
rm -rf api/
rm -rf cmd/cli/
rm -rf pkg/mock/
rm -rf pkg/templates/
rm -rf templates/

# Remove new files
rm -f grafana/dashboards/dashboard.yml
rm -rf grafana/datasources/
rm -f prometheus/alertmanager.yml
```

### 3. Restart Services

```bash
# Restart with old configuration
docker-compose down
docker-compose up -d
```

### 4. Verify Rollback

```bash
# Check gateway is running
curl http://localhost:8080/api/status

# Verify Phase 0-4 endpoints still work
curl -X POST http://localhost:8080/v1/gateway/query \
  -H "Content-Type: application/json" \
  -d '{"query": "Test", "model": "openai", "task_type": "chat"}'
```

## Known Limitations

1. **SDK Generation**: OpenAPI spec is provided, but SDKs must be generated manually using tools like `openapi-generator`
2. **Mock Provider Realism**: Mock responses are simplified and don't match real provider response formats exactly
3. **CLI Dependencies**: CLI requires `github.com/spf13/cobra` package
4. **Template Validation**: Template validation is basic and doesn't check for semantic correctness
5. **Dry-Run Accuracy**: Cost estimates are based on token heuristics and may not match actual costs exactly

## Support

For issues or questions about Phase 5:

1. Check this migration guide
2. Review the OpenAPI specification: `api/openapi.yaml`
3. Check example templates: `templates/examples.yaml`
4. Review Docker Compose logs: `docker-compose logs llmproxy`
5. Test with mock mode: `MOCK_MODE=true`

## What's Next

After Phase 5, the next phases are:

**Phase 6: Caching, Dedup, and Cost-Optimized Routing** (Week 15-16)
- Request coalescing (single-flight)
- Enhanced semantic caching
- Cost-aware routing
- Fallback strategies
- Provider health scoring

**Phase 7: Resilience and Advanced Routing** (Week 17-18)
- Hedging requests
- Adaptive timeouts
- Provider affinity
- Load shedding
- Graceful degradation

## Changelog

### Phase 5 (Week 13-14)

**Added**:
- OpenAPI 3.0 specification (`api/openapi.yaml`)
- Mock providers for testing (`pkg/mock/provider.go`)
- CLI tool for gateway interaction (`cmd/cli/main.go`)
- Prompt templating system (`pkg/templates/engine.go`)
- Dry-run mode endpoint (`POST /v1/gateway/dry-run`)
- Enhanced Docker Compose setup with Jaeger and Alertmanager
- Grafana datasource and dashboard provisioning
- Prometheus alerting configuration
- Example prompt templates (`templates/examples.yaml`)
- Enhanced environment configuration (`.env.example`)

**Changed**:
- Updated `docker-compose.yml` with Phase 2-5 features
- Updated `prometheus/prometheus.yml` with alerting
- Updated `.env.example` with all Phase 0-5 variables
- Enhanced `pkg/gateway/v1/handlers.go` with dry-run handler
- Enhanced `pkg/gateway/v1/types.go` with DryRunResponse

**Dependencies**:
- Added `github.com/spf13/cobra` for CLI (optional)
- Added `gopkg.in/yaml.v3` for template loading

---

**Phase 5 Complete** ✅

This phase significantly improves developer experience with comprehensive tooling, documentation, and local development capabilities. The gateway is now production-ready with excellent observability and developer ergonomics.
