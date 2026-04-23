---
name: release-notes-from-commits
description: 根据 commit 或 PR 列表生成 Release Notes 草稿（新增/修复/破坏性变更等）。当用户说「生成 Release Notes」「release notes」「从 commit 生成发布说明」时应用。
---

# Release Notes 生成 (Release-Notes-From-Commits)

根据本 release 的 commit 或 PR 列表，生成结构化的 Release Notes 草稿，便于人工审阅后采纳或写入 GitHub Release。

## 触发条件

- 用户提到：生成 Release Notes、release notes、从 commit 生成发布说明、发版说明。
- 用户意图：在发版前或打 tag 后，根据代码变更自动生成发布说明草稿。

## 输入

- **commit 列表**：如 `git log --oneline v1.0..HEAD` 或用户粘贴的 commit 列表；或 **PR 列表**（标题 + 描述）。
- **可选**：上一版本 tag、本版本 tag、与 issue/任务关联（commit message 中的 issue 编号或 memory/tasks.md 的 Done 项）。

## 产出结构

生成 Markdown 格式的 Release Notes，建议包含：

1. **版本与日期**：如 `## v1.1.0 (2025-02-xx)`
2. **新增 (Added)**：新功能、新接口、新配置项。
3. **修复 (Fixed)**：Bug 修复、问题修复。
4. **变更 (Changed)**：行为变更、重构、依赖升级（非破坏性）。
5. **破坏性变更 (Breaking)**：不兼容的 API 或配置变更，需单独列出并说明迁移方式。
6. **已知问题 / 后续计划（可选）**：若用户提供或可从 memory/changelog 推断。

## 分类规则建议

- 根据 **commit message 前缀或关键词** 初步分类：`feat`/`add` → Added；`fix`/`bugfix` → Fixed；`refactor`/`chore`/`docs` → Changed；`BREAKING`/`breaking` → Breaking。
- 若 commit 信息不足，可仅列出条目，由人工归类；或结合 PR 标题/描述做摘要。

## 与 memory/ 的联动（可选）

- 若当前仓库存在 `memory/` 目录，可将本次生成的 Release Notes 摘要追加到 `memory/changelog.md`（版本号 + 发布日期 + 简要条目）。
- 若不存在 `memory/`，则跳过本段，不报错。

## 使用说明

- **手动**：用户提供「两个 tag 之间的 commit 列表」或粘贴 `git log` 输出，由本 Skill 生成草稿；审阅后写入 `docs/process/release-notes/` 或 GitHub Release。
- **CI**：在 release workflow 中获取 commit 列表（如 `git log prev_tag..HEAD --pretty=format:"%h %s"`），将输出作为输入传给本 Skill 或调用封装脚本；生成后写入仓库或创建 GitHub Release 草稿。
- 可与 **development-principles** 串联：commit 信息建议与需求/任务关联，便于 Release Notes 可追溯。
