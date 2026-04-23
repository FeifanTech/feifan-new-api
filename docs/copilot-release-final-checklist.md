# Copilot Gateway 上线前最终清单

## 一、构建与测试

- `go vet ./...` 通过
- `go test ./...` 通过
- `go build ./...` 通过
- `cd web && bun install --frozen-lockfile && bun run build` 通过

## 二、核心链路验收（E2E）

- 管理台创建 Seat 绑定成功（`/console/seat-binding`）
- 通过 Copilot 渠道发起 `/v1/chat/completions` 成功
- 限流策略生效（seat 并发/QPM、全局并发）
- 审计事件落库（含 `request_id`、seat、status、duration）
- 审计保留期清理任务在预期时间执行

## 三、灰度与监控

- 告警规则已部署（`deploy/monitoring/copilot-alert-rules.yaml` 或平台等价规则）
- 灰度节奏确认：10% → 50% → 100%
- 关键阈值确认：403、429、P95、fallback 命中
- 回滚演练完成至少 1 次（目标 < 15 分钟恢复）

## 四、发布与回滚资料

- Runbook 已更新且值班同学已阅读
- 当班联系人和升级路径明确
- 回滚命令与日志导出命令可直接执行

## 执行备注（2026-04-23）

- `go vet ./...` 已通过（已修复 unreachable/copy-lock 相关阻断项）。
- `go test ./...` 已通过（已修复 `relay/channel/claude` 与 `relay/helper` 失败项）。
- 前端构建已在本地通过，产物位于 `web/dist`。
- 已补充灰度/回滚脚本：
  - `deploy/scripts/gray-release.sh`
  - `deploy/scripts/rollback.sh`
- E2E 已完成项：
  - 初始化系统、登录 root、调用 `/api/seat_binding/` 创建并查询 Seat 绑定成功。
- 追加验证：
  - 连续 Copilot 请求触发 `429 Too Many Requests`，限流生效。
  - `audit_events` 已落库并包含 `request_id`、`seat_id`、`status_code`、`duration_ms`。
  - 通过 `AUDIT_CLEANUP_INTERVAL_MINUTES=1` + 过期样本记录验证保留期清理任务生效（样本被自动删除）。
  - 已修复 Copilot adaptor 请求路径为 `/chat/completions`（去掉 `/v1` 前缀）并重放验证，当前上游仍返回 404。
- E2E 待完成项的外部前置：
  - 需要可用的 Copilot 上游能力（当前实测上游返回 404），才能验证 `/v1/chat/completions` 成功链路。

