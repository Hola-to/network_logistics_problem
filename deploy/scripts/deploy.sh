#!/bin/bash
# deploy/scripts/deploy.sh

set -euo pipefail

ENVIRONMENT=${1:-staging}
VERSION=${2:-latest}
NAMESPACE="logistics-${ENVIRONMENT}"

echo "🚀 Deploying to ${ENVIRONMENT} environment..."
echo "   Version: ${VERSION}"
echo "   Namespace: ${NAMESPACE}"

# Check prerequisites
command -v kubectl >/dev/null 2>&1 || { echo "kubectl is required"; exit 1; }
command -v helm >/dev/null 2>&1 || { echo "helm is required"; exit 1; }

# Set context based on environment
case $ENVIRONMENT in
  production)
    CONTEXT="production-cluster"
    VALUES_FILE="values-production.yaml"
    ;;
  staging)
    CONTEXT="staging-cluster"
    VALUES_FILE="values-staging.yaml"
    ;;
  *)
    echo "Unknown environment: ${ENVIRONMENT}"
    exit 1
    ;;
esac

kubectl config use-context $CONTEXT

# Update Helm dependencies
echo "📦 Updating Helm dependencies..."
helm dependency update ./deploy/helm/logistics-platform

# Deploy with Helm
echo "🎯 Deploying Helm chart..."
helm upgrade --install logistics ./deploy/helm/logistics-platform \
  --namespace $NAMESPACE \
  --create-namespace \
  -f ./deploy/helm/logistics-platform/values.yaml \
  -f ./deploy/helm/logistics-platform/${VALUES_FILE} \
  --set image.tag=${VERSION} \
  --wait \
  --timeout 10m

# Verify deployment
echo "✅ Verifying deployment..."
kubectl -n $NAMESPACE rollout status deployment/logistics-gateway --timeout=5m
kubectl -n $NAMESPACE rollout status deployment/logistics-solver --timeout=5m
kubectl -n $NAMESPACE rollout status deployment/logistics-auth --timeout=5m

echo "🎉 Deployment complete!"
kubectl -n $NAMESPACE get pods
