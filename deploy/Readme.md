# Kubernetes Deployment

## Prerequisites

- kubectl v1.28+
- kustomize v5.0+ (or kubectl with kustomize support)
- Sealed Secrets controller (for production secrets)

## Quick Start

### Development

```bash
# Deploy to development
kubectl apply -k overlays/development
```

### Staging

```bash
# Deploy to staging
kubectl apply -k overlays/staging
```

### Production

```bash
# Deploy to production (requires sealed secrets)
kubectl apply -k overlays/production
```

### Secrets Management

For production, use Sealed Secrets or External Secrets Operator:

```bash
# Create sealed secret
kubeseal --format yaml < secrets.yaml > sealed-secrets.yaml
```

### Monitoring

The platform exposes Prometheus metrics on each service's metrics port.
ServiceMonitor resources are included for Prometheus Operator integration.
Troubleshooting

```bash
# Check pod status
kubectl -n logistics get pods

# View logs
kubectl -n logistics logs -l app=gateway-svc -f

# Describe deployment
kubectl -n logistics describe deployment gateway-svc
```
