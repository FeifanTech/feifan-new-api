---
name: decisions-summary
description: 根据 memory/decisions.md 或 ADR 文件汇总决策摘要（本 release/本季度/指定范围）。当用户说「汇总决策」「decisions summary」「决策摘要」时应用。
---

# 决策汇总 (Decisions-Summary)

根据已采纳的决策（memory/decisions.md 或 docs/architecture 下 ADR）生成决策摘要，便于评审、复盘或写入 Release Notes 的「重要变更」一节。

## 触发条件

- 用户提到：汇总决策、decisions summary、决策摘要、本季度/本 release 的决策。
- 用户意图：将分散的 ADR 或 decisions 条目汇总为可读的摘要，供团队或发布说明使用。

## 输入

- **决策来源**：`memory/decisions.md` 的 **Accepted** 区，或 `docs/architecture/` 下 ADR 文件（如 `ADR-*.md`）。
- **可选**：时间范围（如「本季度」「自 tag v1.0 以来」）、仅汇总与某主题相关的决策（如「架构」「数据」）。

## 产出结构

生成 Markdown 格式的决策摘要，建议包含：

1. **标题**：如「决策摘要 (YYYY-MM 或 v1.0..v1.1)」
2. **按时间或主题列出的决策条目**，每条含：
   - **标题/结论**：一句话结论。
   - **背景/约束（可选）**：1～2 句背景或约束。
   - **状态**：已采纳 (Accepted)。
3. **可选**：仅汇总 **本 release 周期内新增** 的决策（需根据文件修改时间或 git 历史推断）。

## 规则

- 只汇总 **已采纳 (Accepted)** 的决策，不包含 Proposed 或待确认项，除非用户明确要求「含待确认」。
- 若 decisions 文件不存在或为空，明确提示「未找到已采纳决策」并建议检查路径。

## 与 memory/ 的联动（可选）

- 若本次汇总结果写入仓库，可同时将「本次汇总的时间范围 + 条目数」追加到 `memory/changelog.md`，便于追溯。
- 若不存在 `memory/`，则跳过本段，不报错。

## 使用说明

- **手动**：用户说「汇总最近决策」或「汇总本 release 的决策」，Agent 读取 memory/decisions.md 或 ADR 文件后生成摘要；审阅后写入 `docs/process/decisions-summary.md` 或作为 Release Notes 的一节。
- **CI**：在 release workflow 中，在打 tag 或合并 release 分支后运行脚本或调用本逻辑，读取 decisions 文件，生成摘要并写入仓库或挂到 Release 描述。
- 可与 **role-architect**、**architecture-principles** 串联：决策写入时保持 ADR 结构（背景/约束/方案/结论），便于本 Skill 解析与汇总。
