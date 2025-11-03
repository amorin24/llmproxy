# Kubernetes Deployment for LLM Gateway

This directory contains Kubernetes manifests for deploying the LLM Gateway in production.

## Prerequisites

- Kubernetes cluster (1.24+)
- kubectl configured
- Helm (optional, for monitoring stack)
- cert-manager (for TLS certificates)
- nginx-ingress-controller (for ingress)

## Quick Start

### 1. Create Namespace

```bash
kubectl apply -f namespace.yaml
```

### 2. Create Secrets

**IMPORTANT**: Do NOT use the secrets.yaml template directly. Create secrets securely:

```bash
kubectl create secret generic llmgateway-secrets \
  --from-literal=encryption-key="$(openssl rand -base64 32)" \
  --from-literal=openai-api-key="sk-..." \
  --from-literal=gemini-api-key="..." \
  --from-literal=mistral-api-key="..." \
  --from-literal=claude-api-key="sk-ant-..." \
  --namespace=llmproxy
```

### 3. Create ConfigMap

```bash
kubectl apply -f configmap.yaml
```

### 4. Create ServiceAccount and RBAC

```bash
kubectl apply -f serviceaccount.yaml
```

### 5. Deploy Application

```bash
kubectl apply -f deployment.yaml
kubectl apply -f service.yaml
```

### 6. Configure Ingress

Update `ingress.yaml` with your domain name, then:

```bash
kubectl apply -f ingress.yaml
```

### 7. Enable Autoscaling

```bash
kubectl apply -f hpa.yaml
```

## Verification

### Check Deployment Status

```bash
kubectl get pods -n llmproxy
kubectl get svc -n llmproxy
kubectl get ingress -n llmproxy
```

### Check Logs

```bash
kubectl logs -n llmproxy -l app=llmgateway --tail=100 -f
```

### Test Health Endpoints

```bash
# Port-forward for local testing
kubectl port-forward -n llmproxy svc/llmgateway 8080:80

# Test health endpoints
curl http://localhost:8080/health/liveness
curl http://localhost:8080/health/readiness
curl http://localhost:8080/health/startup
```

### Test API

```bash
# Query endpoint
curl -X POST http://localhost:8080/v1/gateway/query \
  -H "Content-Type: application/json" \
  -d '{
    "query": "What is AI?",
    "model": "openai",
    "task_type": "chat"
  }'

# Cost estimate
curl -X POST http://localhost:8080/v1/gateway/cost-estimate \
  -H "Content-Type: application/json" \
  -d '{
    "query": "What is AI?",
    "model": "openai",
    "expected_output_tokens": 200
  }'
```

## Configuration

### Environment Variables

All configuration is done via environment variables in `deployment.yaml`. Key variables:

**Phase 1: Cost Visibility**
- `PRICE_CATALOG_PATH`: Path to price catalog JSON
- `TRACING_ENABLED`: Enable OpenTelemetry tracing
- `JAEGER_ENDPOINT`: Jaeger collector endpoint

**Phase 3: Streaming & Async Jobs**
- `JOB_TTL_SECONDS`: Job TTL (default: 3600)
- `MAX_JOB_WORKERS`: Max concurrent job workers (default: 10)
- `CIRCUIT_BREAKER_MAX_FAILURES`: Circuit breaker threshold (default: 5)

**Phase 6: Caching & Optimization**
- `COALESCING_ENABLED`: Enable request coalescing (default: true)
- `SEMANTIC_CACHE_ENABLED`: Enable semantic caching (default: true)
- `ROUTING_STRATEGY`: Routing strategy (balanced, cost_optimized, quality_first, latency_optimized)

**Phase 7: Security & Production**
- `ENCRYPTION_KEY`: Encryption key for API key management (from secret)
- `RATE_LIMIT_STRATEGY`: Rate limiting strategy (token_bucket, sliding_window, fixed_window)

### Resource Limits

Default resource limits in `deployment.yaml`:
- Requests: 500m CPU, 512Mi memory
- Limits: 2000m CPU, 2Gi memory

Adjust based on your workload.

### Autoscaling

HPA configuration in `hpa.yaml`:
- Min replicas: 3
- Max replicas: 10
- Target CPU: 70%
- Target Memory: 80%

## Monitoring

### Prometheus Metrics

Metrics are exposed at `/api/metrics` on port 8080. The deployment includes Prometheus annotations for automatic scraping.

### Grafana Dashboards

Import dashboards from `grafana/dashboards/` directory.

### Jaeger Tracing

Configure Jaeger endpoint in `deployment.yaml`:

```yaml
- name: JAEGER_ENDPOINT
  value: "http://jaeger-collector:14268/api/traces"
```

## Security

### Network Policies

Create network policies to restrict traffic:

```yaml
apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  name: llmgateway
  namespace: llmproxy
spec:
  podSelector:
    matchLabels:
      app: llmgateway
  policyTypes:
  - Ingress
  - Egress
  ingress:
  - from:
    - namespaceSelector:
        matchLabels:
          name: ingress-nginx
    ports:
    - protocol: TCP
      port: 8080
  egress:
  - to:
    - namespaceSelector: {}
    ports:
    - protocol: TCP
      port: 443  # HTTPS to LLM providers
```

### Pod Security Standards

The deployment uses restricted security context:
- `runAsNonRoot: true`
- `runAsUser: 1000`
- `fsGroup: 1000`

### Secret Management

For production, use external secret management:
- AWS Secrets Manager
- Google Secret Manager
- HashiCorp Vault
- Kubernetes External Secrets Operator

## Troubleshooting

### Pods Not Starting

```bash
kubectl describe pod -n llmproxy -l app=llmgateway
kubectl logs -n llmproxy -l app=llmgateway --previous
```

### Health Checks Failing

```bash
# Check startup probe
kubectl get events -n llmproxy --sort-by='.lastTimestamp'

# Check logs
kubectl logs -n llmproxy -l app=llmgateway --tail=100
```

### High Memory Usage

```bash
# Check memory usage
kubectl top pods -n llmproxy

# Increase memory limits in deployment.yaml
```

### Rate Limiting Issues

```bash
# Check quota usage
curl http://localhost:8080/api/metrics | grep quota

# Adjust quotas in application code or via API
```

## Rollback

### Rollback Deployment

```bash
# View rollout history
kubectl rollout history deployment/llmgateway -n llmproxy

# Rollback to previous version
kubectl rollout undo deployment/llmgateway -n llmproxy

# Rollback to specific revision
kubectl rollout undo deployment/llmgateway -n llmproxy --to-revision=2
```

## Cleanup

```bash
kubectl delete namespace llmproxy
```

## Production Checklist

- [ ] Secrets created securely (not from template)
- [ ] Resource limits configured appropriately
- [ ] HPA configured and tested
- [ ] Ingress configured with TLS
- [ ] Network policies applied
- [ ] Monitoring configured (Prometheus, Grafana, Jaeger)
- [ ] Alerting rules configured
- [ ] Backup strategy for persistent data
- [ ] Disaster recovery plan documented
- [ ] Load testing completed
- [ ] Security scanning completed
- [ ] Documentation updated

## Support

For issues or questions:
1. Check logs: `kubectl logs -n llmproxy -l app=llmgateway`
2. Check events: `kubectl get events -n llmproxy`
3. Check metrics: `curl http://localhost:8080/api/metrics`
4. Review migration guides in `docs/MIGRATION_PHASE*.md`
