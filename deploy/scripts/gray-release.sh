#!/usr/bin/env bash
set -euo pipefail

NAMESPACE="${NAMESPACE:-new-api}"
DEPLOYMENT="${DEPLOYMENT:-new-api}"
SERVICE="${SERVICE:-new-api}"
TARGET_REPLICAS="${TARGET_REPLICAS:-2}"

if ! command -v kubectl >/dev/null 2>&1; then
  echo "kubectl not found"
  exit 1
fi

echo "[gray] step 1/3: ensure deployment exists"
kubectl -n "${NAMESPACE}" get deploy "${DEPLOYMENT}" >/dev/null

echo "[gray] step 2/3: rollout latest image/config"
kubectl -n "${NAMESPACE}" rollout restart deploy "${DEPLOYMENT}"
kubectl -n "${NAMESPACE}" rollout status deploy "${DEPLOYMENT}" --timeout=300s

echo "[gray] step 3/3: canary checkpoints (10% -> 50% -> 100%)"
echo "Use your ingress/service mesh traffic policy to shift: 10% (30m), 50% (60m), 100%."
echo "Observe: 403 rate, 429 rate, p95 latency, fallback hits, SSE completion."

echo "[gray] enforce desired replicas=${TARGET_REPLICAS}"
kubectl -n "${NAMESPACE}" scale deploy "${DEPLOYMENT}" --replicas="${TARGET_REPLICAS}"
kubectl -n "${NAMESPACE}" rollout status deploy "${DEPLOYMENT}" --timeout=300s

echo "[gray] completed"
