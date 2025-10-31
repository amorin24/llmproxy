# Phase 2 Migration Guide: Vertex AI & Bedrock Integration

## Overview

Phase 2 of the LLM Gateway upgrade adds support for two new LLM providers: **Google Vertex AI** (priority) and **AWS Bedrock**. This phase extends the gateway's capabilities to support enterprise-grade cloud AI platforms with proper authentication and cost tracking.

**Timeline**: 3 weeks (according to gateway upgrade plan)

**Status**: ✅ Complete

## What's New in Phase 2

### New Providers

1. **Vertex AI** (Google Cloud)
   - Gemini models via Google Cloud's Vertex AI platform
   - Enterprise-grade authentication with Google Cloud credentials
   - Regional deployment support (default: us-central1)
   - Cost tracking integrated with Phase 1 pricing system

2. **Bedrock** (AWS)
   - Claude, Titan, and Llama models via AWS Bedrock
   - AWS IAM authentication
   - Regional deployment support (default: us-east-1)
   - Cost tracking integrated with Phase 1 pricing system

### Supported Models

#### Vertex AI Models
- `gemini-2.0-flash` (default)
- `gemini-2.0-flash-lite`
- `gemini-1.5-flash`
- `gemini-1.5-pro`

#### Bedrock Models
- `claude-3-haiku-20240307` (default)
- `claude-3-sonnet-20240229`
- `claude-3-opus-20240229`
- `amazon.titan-text-express-v1`
- `meta.llama3-70b-instruct-v1`

## Breaking Changes

**None.** Phase 2 is fully backward compatible with all existing functionality.

## New Features

### 1. Vertex AI Provider

The Vertex AI provider enables access to Gemini models through Google Cloud's enterprise AI platform.

**Environment Variables:**
```bash
# Required
VERTEX_AI_API_KEY=your-google-cloud-api-key
VERTEX_AI_PROJECT_ID=your-gcp-project-id

# Optional (defaults to us-central1)
VERTEX_AI_LOCATION=us-central1
```

**Example Usage:**
```bash
curl -X POST http://localhost:8080/api/query \
  -H "Content-Type: application/json" \
  -d '{
    "query": "Explain quantum computing",
    "model": "vertex_ai",
    "model_version": "gemini-2.0-flash"
  }'
```

**Cost Tracking:**
Vertex AI pricing is automatically tracked through the Phase 1 cost estimation system. Pricing varies by region and is documented in `docs/price-catalog.json`.

### 2. Bedrock Provider

The Bedrock provider enables access to multiple foundation models through AWS's managed service.

**Environment Variables:**
```bash
# Required
AWS_ACCESS_KEY_ID=your-aws-access-key
AWS_SECRET_ACCESS_KEY=your-aws-secret-key

# Optional (defaults to us-east-1)
AWS_REGION=us-east-1
```

**Example Usage:**
```bash
# Claude via Bedrock
curl -X POST http://localhost:8080/api/query \
  -H "Content-Type: application/json" \
  -d '{
    "query": "Explain quantum computing",
    "model": "bedrock",
    "model_version": "claude-3-haiku-20240307"
  }'

# Titan via Bedrock
curl -X POST http://localhost:8080/api/query \
  -H "Content-Type: application/json" \
  -d '{
    "query": "Explain quantum computing",
    "model": "bedrock",
    "model_version": "amazon.titan-text-express-v1"
  }'

# Llama via Bedrock
curl -X POST http://localhost:8080/api/query \
  -H "Content-Type: application/json" \
  -d '{
    "query": "Explain quantum computing",
    "model": "bedrock",
    "model_version": "meta.llama3-70b-instruct-v1"
  }'
```

**Cost Tracking:**
Bedrock pricing is automatically tracked through the Phase 1 cost estimation system. Pricing varies by region and model family.

### 3. Enhanced Status Endpoint

The `/api/status` endpoint now includes availability status for Vertex AI and Bedrock:

```json
{
  "openai": true,
  "gemini": true,
  "mistral": true,
  "claude": true,
  "vertex_ai": true,
  "bedrock": false
}
```

### 4. Enhanced Router

The router now supports intelligent routing to Vertex AI and Bedrock providers:
- Random selection includes new providers
- Fallback logic includes new providers
- Availability checking for new providers
- Task-based routing (future enhancement)

### 5. Cost Estimation for New Providers

The Phase 1 cost estimation system automatically supports the new providers:

```bash
curl -X POST http://localhost:8080/v1/gateway/cost-estimate \
  -H "Content-Type: application/json" \
  -d '{
    "query": "Explain quantum computing in detail",
    "model": "vertex_ai",
    "model_version": "gemini-2.0-flash"
  }'
```

Response:
```json
{
  "provider": "vertex_ai",
  "model_version": "gemini-2.0-flash",
  "estimated_input_tokens": 6,
  "estimated_output_tokens": 500,
  "estimated_cost_usd": 0.000376,
  "price_per_1k_input_tokens": 0.00025,
  "price_per_1k_output_tokens": 0.00075
}
```

## Upgrade Steps

### 1. Pull Latest Changes

```bash
git pull origin main
```

### 2. Configure Authentication

**For Vertex AI:**
```bash
# Set up Google Cloud credentials
export VERTEX_AI_API_KEY="your-api-key"
export VERTEX_AI_PROJECT_ID="your-project-id"
export VERTEX_AI_LOCATION="us-central1"  # Optional
```

**For Bedrock:**
```bash
# Set up AWS credentials
export AWS_ACCESS_KEY_ID="your-access-key"
export AWS_SECRET_ACCESS_KEY="your-secret-key"
export AWS_REGION="us-east-1"  # Optional
```

### 3. Restart the Application

```bash
# Stop the current instance
pkill -f llmproxy

# Start with new configuration
go run main.go
```

### 4. Verify Provider Availability

```bash
curl http://localhost:8080/api/status
```

Expected response should include:
```json
{
  "vertex_ai": true,
  "bedrock": true
}
```

If a provider shows `false`, verify:
- Environment variables are set correctly
- API keys/credentials are valid
- Network connectivity to provider APIs
- Regional availability of models

### 5. Test New Providers

**Test Vertex AI:**
```bash
curl -X POST http://localhost:8080/api/query \
  -H "Content-Type: application/json" \
  -d '{
    "query": "Hello, world!",
    "model": "vertex_ai"
  }'
```

**Test Bedrock:**
```bash
curl -X POST http://localhost:8080/api/query \
  -H "Content-Type: application/json" \
  -d '{
    "query": "Hello, world!",
    "model": "bedrock"
  }'
```

## Configuration Changes

### Environment Variables

**New Required Variables (per provider):**

For Vertex AI:
- `VERTEX_AI_API_KEY` - Google Cloud API key or service account token
- `VERTEX_AI_PROJECT_ID` - GCP project ID

For Bedrock:
- `AWS_ACCESS_KEY_ID` - AWS access key
- `AWS_SECRET_ACCESS_KEY` - AWS secret access key

**New Optional Variables:**
- `VERTEX_AI_LOCATION` - Vertex AI region (default: us-central1)
- `AWS_REGION` - AWS region (default: us-east-1)

### Price Catalog

The price catalog (`docs/price-catalog.json`) has been updated with pricing for:
- Vertex AI models (all supported Gemini versions)
- Bedrock models (Claude, Titan, Llama families)

Pricing is region-specific and should be updated monthly or when providers announce changes.

## Authentication Setup

### Vertex AI Authentication

**Option 1: API Key (Development)**
```bash
export VERTEX_AI_API_KEY="your-api-key"
export VERTEX_AI_PROJECT_ID="your-project-id"
```

**Option 2: Service Account (Production)**
```bash
# Create service account with Vertex AI permissions
gcloud iam service-accounts create llmproxy-vertex \
  --display-name="LLM Proxy Vertex AI"

# Grant necessary permissions
gcloud projects add-iam-policy-binding YOUR_PROJECT_ID \
  --member="serviceAccount:llmproxy-vertex@YOUR_PROJECT_ID.iam.gserviceaccount.com" \
  --role="roles/aiplatform.user"

# Generate key and set environment variable
gcloud iam service-accounts keys create vertex-key.json \
  --iam-account=llmproxy-vertex@YOUR_PROJECT_ID.iam.gserviceaccount.com

export VERTEX_AI_API_KEY=$(cat vertex-key.json | jq -r '.private_key')
export VERTEX_AI_PROJECT_ID="YOUR_PROJECT_ID"
```

### Bedrock Authentication

**Option 1: IAM User (Development)**
```bash
# Create IAM user with Bedrock permissions
aws iam create-user --user-name llmproxy-bedrock

# Attach Bedrock policy
aws iam attach-user-policy \
  --user-name llmproxy-bedrock \
  --policy-arn arn:aws:iam::aws:policy/AmazonBedrockFullAccess

# Create access key
aws iam create-access-key --user-name llmproxy-bedrock

export AWS_ACCESS_KEY_ID="your-access-key"
export AWS_SECRET_ACCESS_KEY="your-secret-key"
```

**Option 2: IAM Role (Production)**
```bash
# Use EC2 instance role or ECS task role with Bedrock permissions
# No explicit credentials needed - AWS SDK will use instance metadata
```

## Monitoring & Observability

### Prometheus Metrics

All Phase 1 cost tracking metrics automatically support the new providers:

```promql
# Total cost by provider (includes vertex_ai and bedrock)
llmproxy_cost_usd_total{provider="vertex_ai"}
llmproxy_cost_usd_total{provider="bedrock"}

# Cost per request
llmproxy_cost_per_request_usd{provider="vertex_ai", model="gemini-2.0-flash"}
llmproxy_cost_per_request_usd{provider="bedrock", model="claude-3-haiku-20240307"}

# Token costs
llmproxy_token_cost_usd_total{provider="vertex_ai", model="gemini-2.0-flash", token_type="input"}
llmproxy_token_cost_usd_total{provider="bedrock", model="amazon.titan-text-express-v1", token_type="output"}
```

### OpenTelemetry Tracing

Distributed tracing automatically includes spans for Vertex AI and Bedrock requests with attributes:
- `provider`: "vertex_ai" or "bedrock"
- `model_version`: Specific model version
- `input_tokens`: Token count
- `output_tokens`: Token count
- `cost_usd`: Actual cost

### Grafana Dashboards

The Phase 1 cost visibility dashboard automatically displays data for the new providers. No configuration changes needed.

## Rollback Plan

If you need to rollback Phase 2:

### Option 1: Disable Providers

Simply remove the environment variables:
```bash
unset VERTEX_AI_API_KEY
unset VERTEX_AI_PROJECT_ID
unset AWS_ACCESS_KEY_ID
unset AWS_SECRET_ACCESS_KEY
```

The providers will show as unavailable in `/api/status` but all other functionality continues to work.

### Option 2: Revert to Phase 1

```bash
# Checkout Phase 1 commit
git checkout 3f30020

# Rebuild and restart
go build -o llmproxy main.go
./llmproxy
```

## What's Next: Phase 3

Phase 3 (Streaming & Async Jobs) will add:
- Server-Sent Events (SSE) streaming support
- WebSocket streaming support
- Async job system with status tracking
- Long-running operation management
- Job queuing and prioritization

Estimated timeline: 2 weeks

## Testing Recommendations

### Unit Tests

```bash
# Run all tests
go test ./...

# Test specific providers
go test ./pkg/llm -run TestVertexAI
go test ./pkg/llm -run TestBedrock
```

### Integration Tests

```bash
# Test Vertex AI integration
curl -X POST http://localhost:8080/api/query \
  -H "Content-Type: application/json" \
  -d '{
    "query": "Test query",
    "model": "vertex_ai",
    "model_version": "gemini-2.0-flash"
  }'

# Test Bedrock integration
curl -X POST http://localhost:8080/api/query \
  -H "Content-Type: application/json" \
  -d '{
    "query": "Test query",
    "model": "bedrock",
    "model_version": "claude-3-haiku-20240307"
  }'

# Test cost estimation
curl -X POST http://localhost:8080/v1/gateway/cost-estimate \
  -H "Content-Type: application/json" \
  -d '{
    "query": "Test query",
    "model": "vertex_ai"
  }'
```

### Load Tests

```bash
# Test Vertex AI under load
ab -n 100 -c 10 -p vertex_query.json -T application/json \
  http://localhost:8080/api/query

# Test Bedrock under load
ab -n 100 -c 10 -p bedrock_query.json -T application/json \
  http://localhost:8080/api/query
```

## Known Limitations

1. **Vertex AI Authentication**: Currently uses API key authentication. Service account authentication with automatic token refresh will be added in a future phase.

2. **Bedrock Authentication**: Currently uses static AWS credentials. IAM role-based authentication with automatic credential refresh will be added in a future phase.

3. **Regional Pricing**: Price catalog uses default region pricing (us-central1 for Vertex AI, us-east-1 for Bedrock). Multi-region pricing support will be added in Phase 4.

4. **Model Availability**: Not all Bedrock models are available in all regions. Check AWS Bedrock documentation for regional availability.

5. **Rate Limiting**: Provider-specific rate limiting is not yet implemented. This will be added in Phase 4.

6. **Streaming**: Streaming responses are not yet supported for Vertex AI and Bedrock. This will be added in Phase 3.

## Support

For issues or questions:
1. Check the main README.md for general setup
2. Review Phase 1 migration guide (MIGRATION_PHASE1.md) for cost tracking setup
3. Check provider documentation:
   - Vertex AI: https://cloud.google.com/vertex-ai/docs
   - Bedrock: https://docs.aws.amazon.com/bedrock/
4. Review the gateway upgrade plan (docs/gateway-upgrade-plan.md)

## Changelog

### Phase 2 (Current)
- Added Vertex AI provider with Gemini model support
- Added Bedrock provider with Claude, Titan, and Llama support
- Extended router to support new providers
- Updated status endpoint to include new providers
- Integrated new providers with Phase 1 cost tracking
- Updated price catalog with Vertex AI and Bedrock pricing
- Added comprehensive authentication documentation

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
