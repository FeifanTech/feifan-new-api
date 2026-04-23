# Copilot MVP Execution Baseline

## Scope Lock (MVP)
- Included:
  - Copilot channel type and adaptor
  - Seat binding model/service/controller/admin routes
  - CI workflow and Kubernetes deployment manifest
  - Launch runbook (gray + rollback)
- Excluded:
  - Subscription billing, invoice, contract, ARR/MRR dashboard
  - Advanced token exchange scheduler and KMS integration

## WBS Baseline (Implemented in Repository)
- Epic E1:
  - Channel/API type constants
  - Copilot adaptor registration and base header strategy
- Epic E2:
  - `seat_bindings` schema via GORM model migration
  - seat binding CRUD endpoints
- Epic E5:
  - CI workflow
  - K8s manifest
  - runbook

## Progress Status (D1-D4)
- D1: Copilot 基础接入与 Seat 基础能力已完成。
- D2: Token 交换、缓存与刷新机制已完成。
- D3: 限流与审计主链路已完成。
- D4: 审计保留、关键指标告警、request-id 串联已完成。

## Remaining Work (Ops/Release Closure)
- 前端构建在 CI 中强制执行（已补）。
- 监控告警规则模板化并部署到目标环境（模板已补）。
- 按最终清单执行一次上线前全流程验收与回滚演练。

## Sprint Baseline (6 weeks)
- W1-W2: relay core and seat mapping backend
- W3-W4: rate-limit/audit hardening + admin panel integration
- W5: CI/K8s/gray scripts
- W6: UAT, rehearsal, production rollout

## Release Strategy Baseline
- Health gate: `/api/status`
- Gray: 10% -> 50% -> 100%
- Rollback objective: service restore within 15 min
