#!/usr/bin/env bash
set -euo pipefail

NAMESPACE="${NAMESPACE:-new-api}"
DEPLOYMENT="${DEPLOYMENT:-new-api}"
LOG_DIR="${LOG_DIR:-./deploy/logs}"

if ! command -v kubectl >/dev/null 2>&1; then
  echo "kubectl not found"
  exit 1
fi

mkdir -p "${LOG_DIR}"
TS="$(date +%Y%m%d-%H%M%S)"

echo "[rollback] exporting current pod logs"
pods=$(kubectl -n "${NAMESPACE}" get pod -l app="${DEPLOYMENT}" -o jsonpath='{.items[*].metadata.name}')
for p in ${pods}; do
  kubectl -n "${NAMESPACE}" logs "${p}" --tail=2000 > "${LOG_DIR}/${p}-${TS}.log" || true
done

echo "[rollback] undo deployment"
kubectl -n "${NAMESPACE}" rollout undo deploy "${DEPLOYMENT}"
kubectl -n "${NAMESPACE}" rollout status deploy "${DEPLOYMENT}" --timeout=300s

echo "[rollback] done, logs exported to ${LOG_DIR}"
