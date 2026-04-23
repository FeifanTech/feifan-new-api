---
name: using-skills
description: 核心元技能。在任何响应前必须检查是否有相关 Skill；若任务涉及架构/设计/代码/需求/调试等，必须加载并遵循对应 Skill。当用户发起任何开发、设计、评审、调试类请求时，本 Skill 先于其他 Skill 生效。
---

# 使用 Skills 的强制规则

## 定位与原则

本 Skill 是**元 Skill**：定义「何时必须用哪些 Skill、如何执行」。与 AI-DLC 一致：执行任一 **Stage**（需求/设计/实现/测试/发布）时，必须加载该 Stage 对应的 Skill；若已有 **Level 1 Plan**，则按 Plan 中「建议使用的 Skills」加载。

**若任务与某 Skill 相关，必须使用该 Skill。这不是建议，是强制。**

---

## 响应前检查流程

### Step 1：收到用户消息

无论消息是什么（需求、Bug、代码、架构、PRD、Code Review 等），在给出实质性响应前，先进入 Skill 检查流程。

### Step 2：相关性检查

问自己并执行：

- 涉及**架构/技术方案/选型/接口设计/ADR**？→ 加载 **architecture-principles**
- 涉及**写代码/Code Review/重构/提交前自检**？→ 加载 **code-standards**（及 **development-principles**）
- 涉及**开发流程/迭代/交付质量**？→ 加载 **development-principles**
- 涉及**PRD/需求/产品设计/用户故事**？→ 加载 **product-design-principles**
- 涉及**Bug/故障/异常/调试**？→ 加载 **systematic-debugging**
- 涉及**学习或开发/优化 Skill**？→ 加载 **skill-learner-developer**
- 涉及**多步 Agent / 工具调用链 / 生产级可靠性 / Harness Engineering（缰绳工程）**（上下文装配、工具编排、验证闭环、成本中止、可观测）？→ 加载 **harness-engineering**

**规则**：只要存在与上述任一类相关的可能，就必须加载对应 Skill，不能以「可能不相关」为由跳过。

### Step 3：若有 Level 1 Plan

若当前任务已有 AI-DLC 的 **Level 1 Plan**，则按 Plan 中「建议使用的 Skills / 文档」加载对应 Skill，并在执行每个 **Stage** 时确保该 Stage 绑定的 Skill 已加载。**Human Gate** 节点须由人确认，不能跳过。

### Step 4：宣布使用的 Skill

在给出实质性输出前，明确说明：

- 「正在使用 [Skill 名称] 以 [目的]」

例如：

- 「正在使用 architecture-principles 以确保方案符合边界与可观测性要求」
- 「正在使用 code-standards 做 Code Review」
- 「正在使用 harness-engineering 对照 AI-DLC Stage 做缰绳检查（上下文/工具/验证/成本/记录）」

### Step 5：按 Skill 执行

- 若 Skill 有**红旗清单**：触达任一条即停止，将决策交给人，不自行合理化绕过。
- 若 Skill 有**检查清单**：逐项执行或对照，输出中须体现检查结论（满足/不满足/待补充）。
- 不因「这次情况特殊」「用户很急」「小改动」而跳过或简化；若确需例外，由人明确同意后再继续。

### Step 6：响应用户

完成上述检查与执行后，再给出最终回答或代码/文档。

---

## 禁止的合理化借口

以下说法**不能**作为跳过或弱化 Skill 的理由：

- ❌ 「这次情况特殊，不需要遵循」
- ❌ 「我已经很了解最佳实践了」
- ❌ 「用户很急，没时间检查」
- ❌ 「这只是个小改动，不重要」
- ❌ 「我理解精神，不需要照搬」

若出现上述想法，仍须按 Skill 执行；若用户明确要求「跳过检查」，应说明「跳过可能导致不符合规范、返工风险」，并建议至少做最小必要检查。

---

## 自检（响应前必过）

在响应用户前，确认：

- [ ] 已检查所有可能相关的 Skill（只要有一类相关即加载）
- [ ] 已宣布正在使用的 Skill
- [ ] 已按 Skill 的检查清单或红旗清单执行（或明确不适用）
- [ ] 未用上述「禁止的合理化借口」绕过任何步骤

任一项为「否」时，先完成该步再继续响应。

---

## 与 AI-DLC 的对应

- **Stage 执行前**：必须加载该 Stage 对应 Skill；若尚无 Plan，至少声明本任务将使用的 Skill。
- **Human Gate**：Plan 中标注的审批节点须由人确认，AI 不替代人做「通过/驳回」决策。
- **可追溯**：若任务带 Plan-ID，在 Commit/PR 或决策记录中建议引用该 Plan-ID。

---

## 工作结束后的默认动作

- **提交 GitHub**：任务或迭代完成后，建议用户执行提交（`git add` → `git commit` → `git push`）；commit 信息需清晰、可与需求/Plan 关联。

---

## 使用说明

- 本 Skill 应在 CURSOR.md 或项目规则中被引用，使「每次响应前执行 using-skills 的检查」成为默认行为。
- 与 **architecture-principles**、**code-standards**、**development-principles**、**product-design-principles**、**systematic-debugging**、**skill-learner-developer**、**harness-engineering** 等配合：本 Skill 决定「用谁」，后者决定「怎么做」。
