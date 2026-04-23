---
name: k8s-deploy-guard
description: 生成或修改 Kubernetes/Docker 部署配置时触发。当用户说「写 K8s 配置」「部署 YAML」「Dockerfile」「k8s deploy」时应用。
---

# 部署配置守卫 (K8s-Deploy-Guard)

在生成或修改 Kubernetes/Docker 部署配置时，按以下规范做检查与建议，实现运维左移。

## 触发条件

- 用户提到：写 K8s 配置、部署 YAML、Dockerfile、k8s deploy、Helm、容器部署。
- 用户意图：编写或审查部署相关配置，希望符合资源、安全与探针规范。

## 1. 资源限制 (Resource Limits)

- **强制**：所有 Container 必须定义 `resources.limits` 和 `resources.requests`。
- 默认建议值：
  ```yaml
  resources:
    requests:
      memory: "64Mi"
      cpu: "250m"
    limits:
      memory: "128Mi"
      cpu: "500m"
  ```
- 按实际负载可调大，但禁止不设 limits 导致节点被拖垮。

## 2. 安全与镜像

- Image 标签禁止使用 `latest`，必须使用具体版本号或 SHA。
- 禁止以 `root` 用户运行容器（建议设置 `securityContext.runAsUser` > 1000，或镜像内使用非 root 用户）。
- 敏感信息（密钥、证书）禁止写死在 YAML 中，应使用 Secret 或外部配置。

## 3. 存活与就绪探针

- 必须包含 `livenessProbe` 和 `readinessProbe`，避免误杀或误导流量。
- 探针路径与端口需与应用实际暴露的 health 接口一致。

## 与 memory/ 的联动（可选）

- 若当前仓库存在 `memory/` 目录，且本次产出了**已验证的部署命令或约定**（如常用 kubectl 命令、环境区分方式），可追加到 `memory/engineering.md`。
- 若本轮任务或会话结束，可将本次摘要追加到 `memory/changelog.md`：日期、本次做了什么、涉及部署或环境。
- 若不存在 `memory/`，则跳过本段，不报错。

## 使用说明

- 本 Skill 适用于使用 Kubernetes/Docker 部署的项目；未使用 K8s 的项目可忽略。
- 可与 **role-developer**、**development-principles** 串联：部署配置与代码一起纳入 Code Review 与质量门禁。
