# 人工测试文档（Copilot 集成与运营能力）

文档版本：v1  
适用分支：`yunyi_feature`（提交 `2ec29e89`）  
测试目标：验证 GitHub Copilot 渠道集成 + Seat/Tenant + 限流 + 计费/审计 + 运营看板相关能力，支撑测试团队系统回归。

---

## 1. 设计文档核对结论（先看）

本次对照文档：
- `feifan-docs/New-API 集成 GitHub Copilot 方案.pdf`
- `feifan-docs/技术方案.pdf`
- `feifan-docs/产品需求文档（PRD）.pdf`
- `feifan-docs/功能分类分析 V4.0.pdf`

结合当前代码实现，结论如下：

### 1.1 已落地（本轮重点）
- Copilot 新渠道接入（OpenAI/Anthropic 兼容路由复用）。
- GitHub Token -> Copilot Token Exchange（内存+Redis 两级缓存，带 refresh_in 刷新调度）。
- 动态 Base URL 解析（`proxy-ep` + account type fallback）。
- Copilot IDE Headers 注入（含 `x-request-id`、`X-Initiator`、vision 标记）。
- Claude 模型名规范化 + `max_tokens` 自动填充。
- Seat 绑定、tenant/user/seat 上下文注入、Seat+全局限流（Redis + 本地兜底）。
- 账单事件、审计事件、运营接口与看板联动。

### 1.2 发现的“部分实现/待补齐”项（建议测试重点关注）
- Token Exchange 错误分级（401/403/429/5xx）在当前实现中未做明确分支策略与 429 指数退避逻辑，返回为通用错误。
- `/v1/models` 的“网络失败 fallback 内置模型列表”未见显式兜底逻辑（仅有常量模型列表供 channel 元数据使用）。
- Seat 绑定“DB 唯一约束 + 分布式锁防并发重复绑定”未完整体现（当前主要依赖查询+FirstOrCreate逻辑）。
- 审计异步 producer 已实现，但主链路 `recordOpsEvents` 仍是同步写库。
- Copilot Token 健康检查接口当前只做渠道 key 是否为空判断，未实际调用 Token Exchange 探活。
- 文档中提到的告警能力（如 403 超阈值告警）未在本轮新增实现中体现。

> 说明：以上不阻塞本轮测试执行，但请在缺陷报告中标记为“设计一致性差异/后续需求”。

---

## 2. 测试范围

### 2.1 In Scope
- Relay 主链路：`/v1/chat/completions`、`/v1/messages`、`/v1/embeddings`、`/v1/models`
- Ops 管理接口：`/api/ops/*`
- Seat/Tenant 注入与限流中间件
- 账单/审计数据落库与查询导出
- Dashboard 管理员新增卡片展示

### 2.2 Out of Scope
- 生产构建性能压测
- KMS/外部审计归档系统联调
- 告警系统（企业微信/钉钉/邮件策略）联调

---

## 3. 两阶段测试执行模式

按“先基础能力，再运营能力”执行，避免一次性铺开导致反馈噪声。

### 3.1 第一阶段（基础能力，必须先过）
目标：先保证对外核心可用（可供研发/业务先行接入）。

覆盖用例：
- A 全部（渠道与鉴权）
- B 全部（Token Exchange 与 Headers）
- C 全部（模型规范化与参数补齐）
- D-03、D-04（上下文注入与回退行为）
- E-01、E-03、E-04（并发限流与降级）

第一阶段准入标准（建议）：
- 阻断性缺陷（P0/P1）为 0
- 核心接口 `chat/messages/embeddings/models` 全绿
- Redis 可用与不可用两种场景都可稳定返回

### 3.2 第二阶段（运营能力与管理面）
目标：验证管理员与运营可观测、可管理、可导出。

覆盖用例：
- D-01、D-02（Seat 绑定管理）
- E-02（Seat QPM）
- F 全部（账单/审计/健康检查）
- G 全部（运营看板）

第二阶段准入标准（建议）：
- 管理接口无阻断缺陷
- 账单与审计链路数据一致性通过
- 看板核心指标与数据库一致

---

## 4. 测试环境准备

## 4.1 基础服务
- 数据库：SQLite/MySQL/PostgreSQL 任一（建议至少 MySQL 或 PostgreSQL 做一次）
- Redis：建议开启（同时需覆盖“Redis 不可用”场景）
- new-api 服务：`yunyi_feature` 分支代码启动

## 4.2 必备配置
- 至少 1 个可用 Copilot 渠道（type=GitHubCopilot）
- 至少 1 个普通用户 Token（`sk-...`）
- 至少 1 组可用 GitHub Token（`ghp_` / `ghu_` / `gho_` / `github_pat_`）

## 4.3 推荐测试数据
- Tenant：`tenant-a`、`tenant-b`
- External User：`u-a1`、`u-a2`
- Seat：`seat-a1`、`seat-a2`
- Model：`gpt-4.1`、`claude-sonnet-4-5`、`gemini-2.5-flash`

---

## 5. 从分支代码本地启动与建表（详细实操）

本节是给测试同学“一步步照抄”的启动手册，默认 macOS/Linux Shell。  
如果你已经由研发启动好环境，可跳到第 6 节执行用例。

### 5.1 前置软件版本
- Git >= 2.30
- Go = 1.25.x（`go.mod` 当前为 1.25.1）
- Node.js >= 18（用于构建前端静态资源）
- bun（推荐）或 npm（可替代）
- Docker（可选，用于快速起 Redis）

### 5.2 拉取代码并切到测试分支
```bash
git clone https://github.com/FeifanTech/feifan-new-api.git
cd feifan-new-api
git fetch --all
git checkout yunyi_feature
```

### 5.2.1 获取 GitHub Token（用于 Copilot 渠道）
`GitHubCopilot` 渠道的 `API Key` 位置需要填写一个 GitHub Token。推荐使用 PAT（`ghp_` 或 `github_pat_`）。

#### A. 在 GitHub 网页创建 PAT（推荐）
1. 登录 GitHub 账号（该账号需要有有效 Copilot 订阅）。
2. 进入右上角头像 -> `Settings`。
3. 左侧进入 `Developer settings`。
4. 进入 `Personal access tokens`：
   - 推荐：`Fine-grained tokens`
   - 兼容：`Tokens (classic)`
5. 点击 `Generate new token`，设置：
   - **Expiration**：建议先设 30 天或 90 天（测试期足够）
   - **Repository access**：可选最小权限（一般无需 repo 写权限）
   - **Scopes/Permissions**：本方案主要用于调用 `copilot_internal/v2/token`，测试场景通常不需要额外高危权限
6. 生成后**立即复制并保存**（离开页面后无法再次查看明文）。

> 安全提示：Token 只用于测试环境，禁止发群、禁止贴工单明文。建议用密码管理器临时保存。

#### B. 使用 GitHub CLI 获取（备选）
如果本机已登录 `gh`，可直接查看当前 token：

```bash
gh auth token
```

输出通常是 `gho_` 前缀，也可用于渠道测试。

#### C. 校验 Token 是否可用（建议执行）
```bash
curl -i https://api.github.com/copilot_internal/v2/token \
  -H "Authorization: Bearer <YOUR_GITHUB_TOKEN>" \
  -H "Accept: application/json"
```

期望：
- `200`：可用，响应体包含 `token`、`expires_at`、`refresh_in`
- `401`：token 无效或过期
- `403`：账号无 Copilot 订阅或权限不足

### 5.3 准备 `.env`（最小可跑配置）
在项目根目录创建 `.env` 文件（若已有则按需修改）：

```bash
cat > .env <<'EOF'
PORT=3000
GIN_MODE=debug

# 1) 最简单：SQLite（本地文件）
SQLITE_PATH=./data/new-api.db

# 2) 如需 MySQL/PostgreSQL 联调，改用 SQL_DSN，示例：
# SQL_DSN=root:123456@tcp(127.0.0.1:3306)/new-api?parseTime=true
# SQL_DSN=postgresql://root:123456@127.0.0.1:5432/new-api

# Redis（建议开启，便于验证缓存和限流）
REDIS_CONN_STRING=redis://:123456@127.0.0.1:6379/0

# 必填：seat token 加解密依赖
CRYPTO_SECRET=change_me_to_a_long_random_string

# 可选：便于区分节点
NODE_NAME=local-test-node
EOF
```

### 5.4 准备目录与依赖
```bash
mkdir -p data logs
go mod download
```

### 5.5 启动 Redis（建议）
方式 A：本机已有 Redis，确保密码/连接串匹配即可。  
方式 B：Docker 一键启动：

```bash
docker run -d --name newapi-redis -p 6379:6379 redis:7 redis-server --requirepass 123456
```

验证 Redis 可用：
```bash
redis-cli -h 127.0.0.1 -p 6379 -a 123456 ping
# 预期：PONG
```

### 5.6 构建前端静态资源（必须）
`main.go` 使用 `go:embed web/dist`，所以本地运行前必须先产出 `web/dist`。

```bash
cd web
bun install
bun run build

# 若 bun 不可用，可替代：
# npm install --legacy-peer-deps
# npm run build
cd ..
```

验证产物：
```bash
ls web/dist
```

### 5.7 启动后端（本地源码）
```bash
go run main.go
```

启动成功标志：
- 控制台出现服务启动日志
- 可访问 `http://localhost:3000`

### 5.8 建表与迁移说明（重点）
- 本项目启动时会自动执行 `model.InitDB()` -> `migrateDB()` -> `AutoMigrate(...)`。
- 不需要手工执行 SQL 建表脚本。
- 本次新增的关键表会在启动时自动创建：
  - `tenants`
  - `seat_bindings`
  - `billing_events`
  - `audit_events`

#### 5.8.1 如何确认表已创建（SQLite）
```bash
sqlite3 ./data/new-api.db ".tables"
```
预期至少包含：`tenants`、`seat_bindings`、`billing_events`、`audit_events`。

#### 5.8.2 如何确认表已创建（MySQL）
```sql
SHOW TABLES LIKE 'tenants';
SHOW TABLES LIKE 'seat_bindings';
SHOW TABLES LIKE 'billing_events';
SHOW TABLES LIKE 'audit_events';
```

#### 5.8.3 如何确认表已创建（PostgreSQL）
```sql
\dt
-- 或
SELECT tablename FROM pg_tables WHERE tablename IN ('tenants','seat_bindings','billing_events','audit_events');
```

### 5.9 首次初始化管理员账号
服务首次启动后，浏览器打开 `http://localhost:3000`，按引导完成初始化：
- 设置 root 用户名和密码（密码 >= 8）
- 完成后可登录管理后台

### 5.10 测试前最小准备清单（建议按顺序）
1. 登录后台 -> 创建/确认一个用户 token（`sk-...`）
2. 渠道管理 -> 新增 `GitHubCopilot` 渠道（填写 GitHub token，来源见 5.2.1）
3. 使用管理员接口绑定 Seat（或在 UI 对应入口）：
   - `tenant_id`
   - `user_id`
   - `seat_id`
   - `github_token`
4. 调用业务接口时带请求头：
   - `Authorization: Bearer sk-...`
   - `X-Tenant-Id: tenant-a`
   - `X-User-Id: u-a1`

### 5.11 常见启动失败排查

**问题 1：`web/dist` 不存在，go run 报错**
- 处理：先执行第 5.6 节前端构建。

**问题 2：数据库连接失败**
- 处理：检查 `.env` 的 `SQL_DSN`/`SQLITE_PATH` 是否正确，或改回 SQLite 最小方案。

**问题 3：Redis 连接失败**
- 处理：确认 Redis 已启动且密码一致；若暂不测 Redis 场景，可注释 `REDIS_CONN_STRING`，先跑基础能力。

**问题 4：Seat 解密失败**
- 处理：确认 `CRYPTO_SECRET` 已设置，且测试期间不要频繁变更该值。

---

## 6. 人工测试用例

以下每条都建议记录：请求参数、响应体、日志截图、数据库核对结果。

### A. Copilot 渠道与鉴权（第一阶段）

**A-01 渠道创建校验（GitHub Token 前缀）**
- 步骤：在渠道管理新增 GitHubCopilot 渠道，分别输入合法/非法 key。
- 期望：合法 key 可保存；非法 key 被拒绝并返回明确错误。

**A-02 OpenAI 路由可用性**
- 步骤：`POST /v1/chat/completions`，模型 `gpt-4.1`，`stream=false`。
- 期望：返回成功，响应结构符合 OpenAI 协议。

**A-03 Anthropic 路由可用性**
- 步骤：`POST /v1/messages`，模型 `claude-sonnet-4`，`stream=false`。
- 期望：返回 Anthropic 协议结构，消息可正常转换。

**A-04 Embedding 可用性**
- 步骤：`POST /v1/embeddings`。
- 期望：返回 embedding 数据；无协议转换报错。

**A-05 动态模型列表**
- 步骤：`GET /v1/models`。
- 期望：返回 Copilot 上游模型集合；异常时记录当前行为（是否报错/是否兜底）。

---

### B. Token Exchange 与 Header 行为（第一阶段）

**B-01 Token Exchange 成功链路**
- 步骤：首次请求触发 exchange；重复请求命中缓存。
- 期望：首次耗时较高，后续请求明显降低；业务请求成功。

**B-02 Base URL 动态解析**
- 步骤：验证 individual/business/enterprise 账号类型下请求路由。
- 期望：下游目标域名符合预期（`api.githubcopilot.com` / `api.business...` / `api.enterprise...`）。

**B-03 请求头注入完整性**
- 步骤：抓包/代理查看下游请求头。
- 期望：含 `editor-version`、`editor-plugin-version`、`x-request-id`、`x-vscode-user-agent-library-version` 等关键头。

**B-04 X-Initiator / vision 标识**
- 步骤：发送含 assistant/tool 历史与 image 内容的请求。
- 期望：`X-Initiator=agent`；图像请求带 `copilot-vision-request=true`。

---

### C. 模型行为与参数处理（第一阶段）

**C-01 Claude 模型规范化**
- 步骤：请求模型 `claude-sonnet-4-5`。
- 期望：下游使用规范化模型名 `claude-sonnet-4`，请求成功。

**C-02 max_tokens 自动补齐**
- 步骤：不传 `max_tokens` 发起请求。
- 期望：可正常请求；根据模型能力自动填充（无能力数据时 fallback）。

---

### D. Seat/Tenant 绑定与分配（D-03/D-04 第一阶段，D-01/D-02 第二阶段）

**D-01 Seat 绑定接口**
- 步骤：`POST /api/ops/seat/bind` 绑定 tenant/user/seat/github_token。
- 期望：绑定成功，返回绑定记录；加密字段不明文回显。

**D-02 Seat 绑定列表脱敏**
- 步骤：`GET /api/ops/seat/bindings`。
- 期望：返回记录，不包含明文 token。

**D-03 Header 注入映射**
- 步骤：业务请求时带 `X-Tenant-Id`、`X-User-Id`。
- 期望：系统能解析到对应 seat，并使用 seat 关联 GitHub token。

**D-04 无映射回退**
- 步骤：不传 tenant/user 头，直接请求 Copilot 渠道。
- 期望：走渠道 key 默认逻辑；若 key 不可用应有明确报错。

---

### E. 限流与降级（E-01/E-03/E-04 第一阶段，E-02 第二阶段）

**E-01 Seat 并发限流**
- 步骤：同 seat 并发压测超过阈值（默认 3）。
- 期望：部分请求返回 `429`，错误码 `SEAT_CONCURRENT`。

**E-02 Seat QPM 限流**
- 步骤：同 seat 1 分钟内连续请求超阈值。
- 期望：返回 `429`，错误码 `SEAT_QPM`。

**E-03 全局并发保护**
- 步骤：高并发超过全局阈值。
- 期望：返回 `503`，错误码 `GLOBAL_OVERLOAD`。

**E-04 Redis 异常兜底**
- 步骤：关闭 Redis 后重试并发请求。
- 期望：服务不崩溃，限流逻辑退化到本地模式（容量下降但可用）。

---

### F. 账单与审计（第二阶段）

**F-01 账单聚合查询**
- 步骤：调用业务请求后，`GET /api/ops/billing/statements`。
- 期望：按 `billing_cycle + tenant_id` 聚合展示请求数与 token。

**F-02 账单 CSV 导出**
- 步骤：`GET /api/ops/billing/statements/export`。
- 期望：返回 CSV 文件，字段完整且内容可读。

**F-03 审计事件查询**
- 步骤：`GET /api/ops/audit/events?tenant_id=...`。
- 期望：可按租户查询，返回模型/状态码/耗时/IP 哈希等元信息。

**F-04 审计清理**
- 步骤：`DELETE /api/ops/audit/events?days=...`。
- 期望：返回删除条数与 cutoff；历史数据确实减少。

**F-05 Token 健康检查**
- 步骤：`GET /api/ops/copilot/token-health`，构造有 key/空 key 渠道。
- 期望：有 key 显示 `ok`，空 key 显示 `missing_token`。

---

### G. 运营看板（前端，第二阶段）

**G-01 管理员概览卡片展示**
- 步骤：管理员登录 Dashboard。
- 期望：展示 `tenant_count`、`seat_count`、`active_seat_count`、`copilot_channel_count` 四张卡片。

**G-02 数据一致性**
- 步骤：新增/变更 seat、tenant 后刷新看板。
- 期望：卡片数值与数据库统计一致。

---

## 7. 回归重点（高风险）

- `/v1/messages` 流式场景：长文本 + tool call + image 组合。
- Redis 抖动时限流行为是否稳定，是否有误伤。
- Seat 未绑定/绑定失效时的错误提示是否可定位。
- 账单与审计在高并发下是否出现漏记/重复记录。

---

## 8. 缺陷提交流程建议

缺陷单建议字段：
- 用例编号（如 A-03）
- 环境信息（DB/Redis/分支/commit）
- 请求参数与响应体
- 日志片段（服务端 + 网关）
- 是否阻断（Blocker/Critical/Major/Minor）
- 复现概率（必现/偶现）

---

## 9. 测试退出标准（建议）

- P0/P1 缺陷全部关闭。
- 所有 In Scope 用例执行完成且通过率 >= 95%。
- 高风险回归项（第 5 节）全部通过。
- 账单导出与审计查询在目标环境完成至少 1 轮端到端验证。

