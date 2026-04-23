# Copilot Gateway MVP Runbook

## Scope
- Copilot channel enabled for OpenAI/Anthropic relay.
- Seat binding admin APIs available at `/api/seat_binding`.
- Deploy manifest: `deploy/k8s/copilot-gateway.yaml`.
- Alert template: `deploy/monitoring/copilot-alert-rules.yaml`.

## Pre-Deployment Checklist
- `CRYPTO_SECRET`, `SESSION_SECRET`, `SQL_DSN`, `REDIS_CONN_STRING` are set.
- At least 1 admin user can call `/api/seat_binding`.
- Health check `/api/status` is reachable from cluster.
- CI passes:
  - backend: `go vet ./... && go test ./... && go build ./...`
  - frontend: `cd web && bun install --frozen-lockfile && bun run build`
- Alert rules are applied (or equivalent monitor rules are configured in your platform).

## Deployment Steps
1. Apply namespace/config:
   - `kubectl apply -f deploy/k8s/copilot-gateway.yaml`
2. Wait until rollout is complete:
   - `kubectl -n new-api rollout status deploy/new-api`
3. Verify basic APIs:
   - `GET /api/status`
   - `POST /api/seat_binding`
   - `POST /v1/chat/completions` via a Copilot channel

## Gray Strategy
- Step 1: route 10% traffic for 30 minutes.
- Step 2: route 50% traffic for 60 minutes.
- Step 3: route 100% traffic and observe for 24 hours.

Observe:
- 403 ratio
- 429 ratio
- p95 latency
- SSE completion success
- fallback hit count (`copilot_rate_limit_fallback`)

## Rollback
- Trigger:
  - 403 > 3%
  - p95 latency > 2x baseline
  - P0 incident
- Action:
  1. shift traffic to previous deployment
  2. rollback deployment:
     - `kubectl -n new-api rollout undo deploy/new-api`
  3. keep failed pod logs for RCA
  4. export the last 30 minutes of `request-id`-correlated audit data

## Ownership
- Dev A: backend/runtime incident handling
- Dev B: console verification and smoke test
