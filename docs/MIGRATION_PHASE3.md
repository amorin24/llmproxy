# Phase 3 Migration Guide: Streaming & Async Jobs

## Overview

Phase 3 of the LLM Gateway upgrade adds support for **streaming responses** and **asynchronous job processing**. This phase enables real-time token streaming via Server-Sent Events (SSE) and WebSocket, plus long-running operations with job status tracking.

**Timeline**: 3 weeks (according to gateway upgrade plan)

**Status**: ✅ Core Implementation Complete

## What's New in Phase 3

### New Capabilities

1. **Server-Sent Events (SSE) Streaming**
   - Real-time token streaming as responses are generated
   - Cost accumulation tracking during streaming
   - Backpressure handling and early termination support
   - Endpoint: `POST /v1/gateway/stream`

2. **WebSocket Streaming**
   - Bidirectional communication for multi-turn conversations
   - Session management with connection lifecycle
   - Real-time token streaming with cost tracking
   - Endpoint: `WS /v1/gateway/ws`

3. **Async Job System**
   - Submit long-running queries as background jobs
   - Job status tracking (pending → running → completed/failed)
   - Job result retrieval with cost and token information
   - Optional webhook delivery on completion
   - In-memory job store with configurable TTL (default: 1 hour)

4. **Circuit Breaker Pattern**
   - Per-provider circuit breakers to prevent cascading failures
   - Automatic recovery with half-open state
   - Configurable failure thresholds and timeouts
   - Protection against provider outages

## Breaking Changes

**None.** Phase 3 is fully backward compatible with all existing functionality.

## New Features

### 1. Server-Sent Events (SSE) Streaming

Stream LLM responses in real-time with cost tracking.

**Endpoint**: `POST /v1/gateway/stream`

**Request:**
```json
{
  "query": "Explain quantum computing in detail",
  "model": "openai",
  "model_version": "gpt-4o"
}
```

**Response Stream:**
```
data: {"token": "Quantum", "cost_usd": 0.0001}

data: {"token": " computing", "cost_usd": 0.0002}

data: {"token": " is", "cost_usd": 0.0003}

...

data: {"done": true, "total_cost_usd": 0.0052}
```

**Example Usage:**
```bash
curl -X POST http://localhost:8080/v1/gateway/stream \
  -H "Content-Type: application/json" \
  -d '{
    "query": "Explain quantum computing",
    "model": "openai"
  }' \
  --no-buffer
```

**JavaScript Client Example:**
```javascript
const eventSource = new EventSource('/v1/gateway/stream');

eventSource.onmessage = (event) => {
  const data = JSON.parse(event.data);
  
  if (data.token) {
    console.log('Token:', data.token);
    console.log('Accumulated Cost:', data.cost_usd);
  }
  
  if (data.done) {
    console.log('Total Cost:', data.total_cost_usd);
    eventSource.close();
  }
  
  if (data.error) {
    console.error('Error:', data.error);
    eventSource.close();
  }
};
```

### 2. WebSocket Streaming

Bidirectional streaming for multi-turn conversations.

**Endpoint**: `WS /v1/gateway/ws`

**Message Types:**

**Client → Server (Query):**
```json
{
  "type": "query",
  "query": "What is machine learning?",
  "model": "openai",
  "model_version": "gpt-4o"
}
```

**Server → Client (Token):**
```json
{
  "type": "token",
  "token": "Machine",
  "cost_usd": 0.0001
}
```

**Server → Client (Done):**
```json
{
  "type": "done",
  "done": true,
  "total_cost_usd": 0.0045
}
```

**Server → Client (Error):**
```json
{
  "type": "error",
  "error": "Provider unavailable"
}
```

**JavaScript Client Example:**
```javascript
const ws = new WebSocket('ws://localhost:8080/v1/gateway/ws');

ws.onopen = () => {
  console.log('WebSocket connected');
  
  // Send query
  ws.send(JSON.stringify({
    type: 'query',
    query: 'What is machine learning?',
    model: 'openai'
  }));
};

ws.onmessage = (event) => {
  const data = JSON.parse(event.data);
  
  if (data.type === 'connected') {
    console.log('Session ID:', data.session_id);
  } else if (data.type === 'token') {
    console.log('Token:', data.token);
    console.log('Cost:', data.cost_usd);
  } else if (data.type === 'done') {
    console.log('Total Cost:', data.total_cost_usd);
  } else if (data.type === 'error') {
    console.error('Error:', data.error);
  }
};

ws.onerror = (error) => {
  console.error('WebSocket error:', error);
};

ws.onclose = () => {
  console.log('WebSocket closed');
};
```

### 3. Async Job System

Submit long-running queries as background jobs with status tracking.

**Job Submission**: `POST /v1/gateway/jobs`

**Request:**
```json
{
  "query": "Write a comprehensive analysis of quantum computing",
  "model": "openai",
  "model_version": "gpt-4o",
  "callback_url": "https://your-service.com/webhook"
}
```

**Response:**
```json
{
  "job_id": "550e8400-e29b-41d4-a716-446655440000",
  "status": "pending",
  "estimated_cost_usd": 0.05
}
```

**Job Status**: `GET /v1/gateway/jobs/{id}`

**Response:**
```json
{
  "job_id": "550e8400-e29b-41d4-a716-446655440000",
  "status": "running",
  "estimated_cost_usd": 0.05,
  "created_at": "2025-11-03T13:00:00Z",
  "started_at": "2025-11-03T13:00:05Z"
}
```

**Job Result**: `GET /v1/gateway/jobs/{id}/result`

**Response:**
```json
{
  "job_id": "550e8400-e29b-41d4-a716-446655440000",
  "status": "completed",
  "result": "Quantum computing is a revolutionary...",
  "actual_cost_usd": 0.048,
  "input_tokens": 12,
  "output_tokens": 850,
  "total_tokens": 862,
  "provider": "openai"
}
```

**Example Usage:**
```bash
# Submit job
JOB_ID=$(curl -X POST http://localhost:8080/v1/gateway/jobs \
  -H "Content-Type: application/json" \
  -d '{
    "query": "Explain quantum computing",
    "model": "openai"
  }' | jq -r '.job_id')

# Check status
curl http://localhost:8080/v1/gateway/jobs/$JOB_ID

# Get result (when completed)
curl http://localhost:8080/v1/gateway/jobs/$JOB_ID/result
```

**Webhook Delivery:**

When a job completes (if `callback_url` was provided), the gateway will POST to the callback URL:

```json
{
  "job_id": "550e8400-e29b-41d4-a716-446655440000",
  "status": "completed",
  "result": "Quantum computing is...",
  "actual_cost_usd": 0.048,
  "input_tokens": 12,
  "output_tokens": 850,
  "provider": "openai"
}
```

### 4. Circuit Breaker Pattern

Automatic circuit breakers protect against cascading failures when providers are unavailable.

**States:**
- **Closed**: Normal operation, requests flow through
- **Open**: Provider is failing, requests are rejected immediately
- **Half-Open**: Testing if provider has recovered

**Configuration:**
- Max failures before opening: 5 (configurable)
- Timeout before retry: 60 seconds (configurable)
- Per-provider circuit breakers for all 6 providers

**Behavior:**
- After 5 consecutive failures, circuit opens
- Requests fail immediately with `ErrCircuitOpen`
- After 60 seconds, circuit transitions to half-open
- One successful request closes the circuit
- One failed request in half-open reopens the circuit

## Upgrade Steps

### 1. Pull Latest Changes

```bash
git pull origin main
```

### 2. Update Dependencies

The Phase 3 implementation requires the `gorilla/websocket` package:

```bash
go get github.com/gorilla/websocket
go get github.com/google/uuid
```

### 3. Configure Job System (Optional)

Set environment variables to customize job system behavior:

```bash
# Job TTL (default: 1 hour)
export JOB_TTL_SECONDS=3600

# Max concurrent job workers (default: 10)
export MAX_JOB_WORKERS=10

# Circuit breaker settings
export CIRCUIT_BREAKER_MAX_FAILURES=5
export CIRCUIT_BREAKER_TIMEOUT_SECONDS=60
```

### 4. Restart the Application

```bash
# Stop the current instance
pkill -f llmproxy

# Start with new configuration
go run main.go
```

### 5. Verify New Endpoints

```bash
# Test SSE streaming
curl -X POST http://localhost:8080/v1/gateway/stream \
  -H "Content-Type: application/json" \
  -d '{"query": "Hello", "model": "openai"}' \
  --no-buffer

# Test async jobs
curl -X POST http://localhost:8080/v1/gateway/jobs \
  -H "Content-Type: application/json" \
  -d '{"query": "Hello", "model": "openai"}'

# Test WebSocket (requires WebSocket client)
wscat -c ws://localhost:8080/v1/gateway/ws
```

## Configuration Changes

### New Environment Variables

**Job System:**
- `JOB_TTL_SECONDS` - How long to keep completed jobs (default: 3600)
- `MAX_JOB_WORKERS` - Maximum concurrent job workers (default: 10)

**Circuit Breaker:**
- `CIRCUIT_BREAKER_MAX_FAILURES` - Failures before opening (default: 5)
- `CIRCUIT_BREAKER_TIMEOUT_SECONDS` - Timeout before retry (default: 60)

### No Breaking Changes

All existing endpoints continue to work unchanged:
- `/api/query` - Standard query endpoint
- `/api/status` - Provider status
- `/v1/gateway/query` - Gateway query (Phase 0)
- `/v1/gateway/cost-estimate` - Cost estimation (Phase 1)

## Architecture

### Job System Architecture

```
Client → Submit Job → Job Store (in-memory)
                          ↓
                     Job Queue (channel)
                          ↓
                   Worker Pool (goroutines)
                          ↓
                   LLM Provider Query
                          ↓
                   Update Job Result
                          ↓
                   Webhook Delivery (optional)
```

**Components:**
- **Job Store**: Thread-safe in-memory storage with TTL cleanup
- **Job Queue**: Buffered channel (capacity: 100)
- **Worker Pool**: Configurable number of goroutines (default: 10)
- **Job Lifecycle**: pending → running → completed/failed

### Streaming Architecture

**SSE Streaming:**
```
Client → POST /v1/gateway/stream
           ↓
      Route Request
           ↓
      Query LLM Provider
           ↓
      Split Response into Tokens
           ↓
      Stream Tokens with Cost
           ↓
      Send Final Message
```

**WebSocket Streaming:**
```
Client ←→ WS /v1/gateway/ws
           ↓
      Bidirectional Messages
           ↓
      Query Processing
           ↓
      Token Streaming
           ↓
      Multi-turn Conversations
```

### Circuit Breaker Architecture

```
Request → Circuit Breaker Check
              ↓
         State: Closed?
              ↓
         Execute Request
              ↓
         Success/Failure
              ↓
         Update Circuit State
```

## Monitoring & Observability

### Job Metrics

Monitor job system health:

```promql
# Pending jobs
llmproxy_jobs_pending_total

# Running jobs
llmproxy_jobs_running_total

# Completed jobs
llmproxy_jobs_completed_total

# Failed jobs
llmproxy_jobs_failed_total

# Job processing time
llmproxy_job_duration_seconds
```

### Circuit Breaker Metrics

Monitor circuit breaker state:

```promql
# Circuit breaker state by provider
llmproxy_circuit_breaker_state{provider="openai"}

# Circuit breaker failures
llmproxy_circuit_breaker_failures_total{provider="openai"}

# Circuit breaker opens
llmproxy_circuit_breaker_opens_total{provider="openai"}
```

### Streaming Metrics

Monitor streaming performance:

```promql
# Active SSE connections
llmproxy_sse_connections_active

# Active WebSocket connections
llmproxy_websocket_connections_active

# Streaming errors
llmproxy_streaming_errors_total{type="sse"}
llmproxy_streaming_errors_total{type="websocket"}
```

## Rollback Plan

If you need to rollback Phase 3:

### Option 1: Disable New Features

Simply don't use the new endpoints:
- `/v1/gateway/stream` (SSE)
- `/v1/gateway/ws` (WebSocket)
- `/v1/gateway/jobs` (Async jobs)

All existing functionality continues to work unchanged.

### Option 2: Revert to Phase 2

```bash
# Checkout Phase 2 commit
git checkout a0ed3df

# Rebuild and restart
go build -o llmproxy main.go
./llmproxy
```

## What's Next: Phase 4

Phase 4 (Advanced Observability and SLOs) will add:
- Service Level Objectives (SLOs) per provider
- Cost anomaly detection
- Job monitoring and alerting
- Trace enrichment with cost/token data
- Log hygiene and PII redaction
- Comprehensive alerting rules

Estimated timeline: 2 weeks

## Testing Recommendations

### Unit Tests

```bash
# Run all tests
go test ./...

# Test specific packages
go test ./pkg/jobs
go test ./pkg/streaming
go test ./pkg/circuitbreaker
```

### Integration Tests

**Test SSE Streaming:**
```bash
curl -X POST http://localhost:8080/v1/gateway/stream \
  -H "Content-Type: application/json" \
  -d '{
    "query": "Count to 10",
    "model": "openai"
  }' \
  --no-buffer
```

**Test Async Jobs:**
```bash
# Submit job
JOB_ID=$(curl -X POST http://localhost:8080/v1/gateway/jobs \
  -H "Content-Type: application/json" \
  -d '{"query": "Test", "model": "openai"}' \
  | jq -r '.job_id')

# Wait a few seconds
sleep 5

# Check result
curl http://localhost:8080/v1/gateway/jobs/$JOB_ID/result
```

**Test WebSocket:**
```bash
# Install wscat if needed
npm install -g wscat

# Connect and send message
wscat -c ws://localhost:8080/v1/gateway/ws
> {"type": "query", "query": "Hello", "model": "openai"}
```

### Load Tests

**Test SSE under load:**
```bash
ab -n 100 -c 10 -p sse_query.json -T application/json \
  http://localhost:8080/v1/gateway/stream
```

**Test async jobs under load:**
```bash
ab -n 1000 -c 50 -p job_query.json -T application/json \
  http://localhost:8080/v1/gateway/jobs
```

## Known Limitations

1. **In-Memory Job Store**: Jobs are stored in memory and will be lost on restart. Persistent storage (Redis/PostgreSQL) will be added in Phase 6.

2. **Job Queue Capacity**: Job queue has a fixed capacity of 100. Jobs submitted when queue is full will be rejected. This will be improved in Phase 4.

3. **No Job Cancellation**: Once submitted, jobs cannot be cancelled. This will be added in Phase 4.

4. **Webhook Retries**: Webhook delivery has no retry logic. Failed webhooks are logged but not retried. This will be improved in Phase 4.

5. **Streaming Token Simulation**: Current implementation simulates streaming by splitting the complete response. Native provider streaming will be added in Phase 5.

6. **Circuit Breaker Persistence**: Circuit breaker state is not persisted across restarts. This will be improved in Phase 6.

7. **WebSocket Authentication**: WebSocket connections have no authentication. This will be added in Phase 4.

## Support

For issues or questions:
1. Check the main README.md for general setup
2. Review Phase 1 migration guide (MIGRATION_PHASE1.md) for cost tracking
3. Review Phase 2 migration guide (MIGRATION_PHASE2.md) for provider setup
4. Review the gateway upgrade plan (docs/gateway-upgrade-plan.md)

## Changelog

### Phase 3 (Current)
- Added SSE streaming endpoint (`/v1/gateway/stream`)
- Added WebSocket streaming endpoint (`/v1/gateway/ws`)
- Added async job system (`/v1/gateway/jobs`)
- Added circuit breaker pattern for provider protection
- Added job status tracking and result retrieval
- Added webhook delivery for job completion
- Added streaming cost tracking

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
