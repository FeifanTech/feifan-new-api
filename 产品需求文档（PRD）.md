企业级 AI Agent 网关平台 · v3.0

版本：v3.0  
日期：2026-04-20  
核心调整：基于 copilot-api / openclaw / cc-switch 三项目综合方案，纯 HTTP 直调 Copilot API，支持完整双向协议转换  
范围：Provider 兼容模式，不含 Agent-Native

---

1. 产品概述

基于 new-api 二次开发的企业级 AI 模型网关，通过 GitHub Token → Copilot Token Exchange 机制，将 GitHub Copilot 订阅下的全部模型（GPT、Claude、Gemini 等）以 OpenAI 和 Anthropic 双协议对外透明提供，并叠加企业级 Seat 管理、多级限流和旁路审计。

放弃 Copilot SDK 的原因：  
Copilot SDK 本质是封装本地 Language Server 子进程，适合桌面工具，new-api 是多用户服务端网关，纯 HTTP 直调方案更轻量、更稳定、更适合高并发多租户场景。

核心价值：

- 企业客户零改造：OpenAI 客户端（Cursor、Continue 等）直接换 Base URL；Anthropic 客户端（Claude Code）同样零改造
- 一个网关覆盖 Copilot 订阅下所有模型（GPT-4.1、Claude 4、Gemini 2.5 等）
- 统一采购 Copilot Seat，1:1 用户映射，防滥用防风控
- 全链路旁路审计，满足企业合规要求
- 海外单集群 Kubernetes 部署，运维简单

---

1. 用户角色与使用方式

2.1 用户角色
角色
主要诉求
入口
企业开发者
现有工具换个 Base URL 即可用
任意 OpenAI / Anthropic 兼容客户端
企业内部系统
API 接入，集成到 CI/CD 等
REST API
企业管理员
管理 Seat、查看用量、审计
Web 管理控制台
平台运营
管理 GitHub Token 池、计费
运营后台

2.2 最终用户使用方式

方式 A：OpenAI 兼容客户端（零改造）
Base URL:  [https://gateway.your-domain.com](https://gateway.your-domain.com)
API Key:   sk-{企业分配的用户 Key}
Model:     gpt-4.1 / claude-sonnet-4 / gemini-2.5-flash（任选）

适用：Cursor、Continue、OpenCode、任意 OpenAI SDK 项目。

方式 B：Anthropic 兼容客户端（零改造）

ANTHROPIC_BASE_URL=[https://gateway.your-domain.com](https://gateway.your-domain.com)
ANTHROPIC_API_KEY=sk-{企业分配的用户 Key}

适用：Claude Code、任意 Anthropic SDK 项目。  
网关自动处理 Anthropic ↔ OpenAI 协议双向转换，用户无感知。

方式 C：直接 API 调用

# OpenAI 格式

POST /v1/chat/completions
Authorization: Bearer sk-user-key
{"model": "claude-sonnet-4", "messages": [...], "stream": true}

# Anthropic 格式

POST /v1/messages
Authorization: Bearer sk-user-key
{"model": "claude-sonnet-4", "messages": [...], "stream": true}

# Embedding

POST /v1/embeddings
Authorization: Bearer sk-user-key
{"model": "text-embedding-3-small", "input": "..."}

方式 D：Web 管理控制台（管理员）

- Seat 绑定管理（用户 ↔ GitHub Token）
- 实时用量仪表盘（Token 消耗、请求数、限流命中率）
- 审计日志查询
- API Key 管理

---

1. 对外接口清单

接口
方法
协议
说明
/v1/chat/completions
POST
OpenAI
所有模型透传（Claude 模型内部转换，对外仍是 OpenAI 格式）
/v1/messages
POST
Anthropic
完整 Anthropic 协议支持，内部转 OpenAI 再转回
/v1/embeddings
POST
OpenAI
Embedding 透传
/v1/models
GET
OpenAI
动态从 Copilot 拉取可用模型列表

---

1. 核心功能需求

4.1 GitHub Copilot 渠道接入

Token Exchange（最核心）：

- 用户 API Key → Seat 映射 → GitHub Token → Copilot Token（短期，约 1 小时有效）
- 按服务端返回的 refresh_in 字段主动刷新（提前 60 秒），而非被动等过期
- 两级缓存：进程内 sync.Map（单实例） + Redis（多实例共享）

动态 Base URL：

- 从 Copilot Token 的 proxy-ep 字段解析（proxy.xxx.com → api.xxx.com）
- 支持 individual / business / enterprise 三种账号类型 fallback

IDE Headers 注入：  
每次请求注入完整的 Copilot IDE 标识头（共 10 个 Header），缺少任何一个可能导致 403 或行为异常。关键 Header 包括：

- editor-version：动态从 AUR 拉取最新 VSCode 版本（24h 缓存），无需手动维护
- x-request-id：每次请求生成唯一 UUID
- X-Initiator：根据消息历史动态判断 user / agent

协议路由：

请求来源
模型前缀
Copilot 端点
转换
/v1/chat/completions
全部
/chat/completions
透传（Claude 需模型名规范化）
/v1/messages
全部
/chat/completions
请求：Anthropic→OpenAI；响应：OpenAI→Anthropic
/v1/embeddings
全部
/embeddings
透传

模型名规范化：  
Copilot 不支持带子版本号的 Claude 模型名（如 claude-sonnet-4-20250514 / claude-sonnet-4-5），必须截断为 claude-sonnet-4。

4.2 Anthropic ↔ OpenAI 双向协议转换

这是本次集成与上一版本最大的升级，支持 Claude Code 等 Anthropic 客户端零改造接入。

请求转换（Anthropic → OpenAI）：

- system 字段提取（string / TextBlock 数组两种形式）
- tool_result block → role: "tool"（必须排在用户消息之前）
- tool_use block → tool_calls[]
- thinking block → 合并进 content 文本（OpenAI 无对应概念）
- image block → image_url ContentPart
- tool_choice 映射（any → required，tool → function 指定）
- max_tokens 动态填充（从模型 capabilities 读取，fallback 8192）

响应转换（OpenAI → Anthropic）：

- 文本 → text block
- tool_calls → tool_use block
- finish_reason 映射（stop → end_turn，tool_calls → tool_use 等）
- usage 字段映射（含 cache_read_input_tokens）

流式转换（OpenAI SSE → Anthropic SSE 事件序列）：  
使用有状态状态机转换，维护 block 开关状态：

- 首 chunk → message_start
- text delta → content_block_start + content_block_delta (text_delta)
- tool_calls delta → content_block_start (tool_use) + content_block_delta (input_json_delta)
- finish_reason → content_block_stop + message_delta + message_stop

4.3 Seat 1:1 映射

- 一个企业用户（tenant_id + user_id）绑定唯一一个 GitHub Token（对应一个 Copilot Seat）
- DB 唯一约束 + 分布式锁防并发重复绑定
- GitHub Token 加密存储（AES-256-GCM + KMS）
- 管理员可通过控制台解绑 / 重新绑定

4.4 多级限流

- 单 Seat 并发：默认 1–3 并发请求（可按租户配置），超限返回 429
- 单 Seat QPM：防止 GitHub 风控（Token Exchange 频率）
- 全局并发：整体防雪崩上限
- Fail-open：Redis 不可用时本地内存兜底（50% 容量）

4.5 旁路审计

- 异步写入，不阻塞主链路
- 记录：模型、Token 用量、响应状态、耗时、IP Hash（不记录请求内容）
- 存储：PostgreSQL 热数据（30 天） + 对象存储冷归档

4.6 动态模型列表

- 从 Copilot /models 接口实时拉取（30 分钟缓存）
- 自动适配不同 GitHub Plan 的可用模型
- 内置 fallback 模型列表（网络不可用时使用）

---

1. 内置模型列表（Fallback）

GPT 系列:    gpt-4.1, gpt-4.1-mini, gpt-4.1-nano, gpt-4o, gpt-4o-mini
o 系列:      o3, o3-mini, o4-mini
Claude 系列: claude-opus-4, claude-opus-4-5, claude-sonnet-4, claude-sonnet-4-5, claude-haiku-4-5
Gemini 系列: gemini-2.5-pro, gemini-2.5-flash, gemini-2.0-flash
其他:        xai/grok-3, xai/grok-3-mini, mistral-large, cohere-command-r-plus

---

1. 部署架构

用户 / 企业内部系统
   │ HTTPS（OpenAI 或 Anthropic 格式）
   ▼
Load Balancer（AWS ALB / GCP LB）
   │
   ▼
new-api 网关（Go，Kubernetes，2-4 副本 HPA）
┌─────────────────────────────────────────────────────┐
│  鉴权中间件（API Key → tenant/user）                  │
│  Seat 映射（user → GitHub Token，Redis 缓存）         │
│  多级限流（Redis，Fail-open 本地兜底）                 │
│                                                     │
│  Copilot Adaptor                                    │
│   ├── Token Exchange（refresh_in 主动刷�� + 缓存）    │
│   ├── 动态 Base URL 解析（proxy-ep + 账号类型）        │
│   ├── 完整 IDE Headers 注入（动态 VSCode 版本）        │
│   ├── /v1/chat/completions → 透传                   │
│   ├── /v1/messages → 双向协议转换（含流式状态机）       │
│   └── /v1/embeddings → 透传                         │
│                                                     │
│  旁路审计 Producer（异步）                             │
│  计费事件写入                                         │
└─────────────────────────────────────────────────────┘
   │                    │
   ▼                    ▼
PostgreSQL           Redis
（seat/billing/audit）（限流/token缓存）
   │
   ▼（出站 HTTPS）
GitHub Copilot API（api.*.githubcopilot.com）

技术选型：

- Kubernetes：AWS EKS / GCP GKE，单 Region，多 AZ
- PostgreSQL：RDS / Cloud SQL（托管，Multi-AZ）
- Redis：ElastiCache / Memorystore（托管）
- 对象存储：S3 / GCS（审计冷归档）
- CI/CD：GitHub Actions

---

1. 风险

风险
级别
应对
Copilot Internal API 是非公开接口，随时可能变更
P1
跟踪三个参考项目更新；Adaptor 单层隔离
IDE Headers 版本过时被拒（全量 403）
P1
动态从 AUR 拉取版本；监控 403 错误率告警；版本号可配置化热更新
单账号并发过高触发 GitHub 风控
P0
单 Seat QPM + 并发限流；多 Token 池分散压力
GitHub Token 泄露
P0
AES-256-GCM 加密 + KMS；Token 只在运行时解密，不出现在日志
Copilot Seat 共享条款合规
P0
提前与 GitHub 销售确认；严格 1:1 映射
new-api AGPL-3.0 许可证
P1
内部部署通常满足合规；正式上线前法务确认
stream 状态机边缘 case
P2
充分测试 tool_use / thinking / 并发 chunk 场景