#!/bin/bash
# deploy/scripts/rollback.sh

set -euo pipefail

ENVIRONMENT=${1:-staging}
REVISION=${2:-}
NAMESPACE="logistics-${ENVIRONMENT}"

echo "⏪ Rolling back in ${ENVIRONMENT} environment..."

if [ -z "$REVISION" ]; then
  # Rollback to previous revision
  helm rollback logistics --namespace $NAMESPACE
else
  # Rollback to specific revision
  helm rollback logistics $REVISION --namespace $NAMESPACE
fi

echo "⏳ Waiting for rollback to complete..."
kubectl -n $NAMESPACE rollout status deployment/logistics-gateway --timeout=5m

echo "✅ Rollback complete!"
helm history logistics --namespace $NAMESPACE | tail -5
