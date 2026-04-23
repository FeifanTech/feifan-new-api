---
name: skill-learner-developer
description: 从市场学习 skills、开发新 skill、优化既有 skill 的 meta-skill。当用户说「学习 skills」「从市场学 skill」「开发/写一个 skill」「优化 skill」「技能市场」时使用。
---

# Skill 学习与开发 Agent

## 角色与目标

以**Skill 学习与开发**角色工作：从技能市场发现与学习优质 skills，按 SKILL.md 规范开发新 skill，并优化既有 skill 的清晰度、可复用性与可触发性。

---

## 一、从市场学习 Skills

### 1.1 推荐市场与来源

| 来源 | 用途 | 链接/说明 |
|------|------|-----------|
| **SkillsMP** | 海量技能发现、按 SKILL.md 标准、可安装 | https://skillsmp.com — 语义/关键词搜索，分类浏览，支持 Claude/Codex/ChatGPT |
| **Agent Skills Guide** | 人工精选、少而精、学写法 | https://www.agentskills.guide — 编辑推荐，按领域分类 |
| **Anthropic/skills** | 官方示例与 SKILL.md 规范 | https://github.com/anthropics/skills — 文档生成等示例，技能规范说明 |
| **awesome-cursorrules** | Cursor 规则写法、上下文与风格 | https://github.com/PatrickJS/awesome-cursorrules — 规则/配置可借鉴到 SKILL.md |
| **Cursor Directory** | 官方社区、规则与 MCP | https://cursor.directory |

### 1.2 学习动作顺序

1. **定目标**：明确要学的领域（如 DevOps、文档生成、Code Review、产品评审）。
2. **选来源**：要「量」用 SkillsMP 搜索；要「质」用 Agent Skills Guide 或 Anthropic/skills。
3. **拆解**：打开 1～3 个优质 SKILL.md，提取：frontmatter（name/description）、结构（原则→检查清单→产出物）、触发条件（何时被调用）、与项目现有 skills 的差异。
4. **落地**：将可复用结构或表述迁移到本仓库 `.cursor/skills/` 下，保持与现有 skill 命名与目录一致。

### 1.3 安装外部 Skill 到本仓库（可选）

- 从 SkillsMP 或 GitHub 拿到 skill 仓库/路径后，可将对应 **SKILL.md（及同目录脚本、模板）** 复制到 `.cursor/skills/<skill-name>/`。
- 安装后检查：name/description 是否与现有 skills 冲突；是否需要按本仓库规范增删「工作结束后的默认动作」等段落。

---

## 二、开发新 Skill

### 2.0 Skill 开发工作流（TDD，建议用于核心 Skill）

**原则**：若未观察过 AI 在没有该 Skill 时的失败行为，就不知道 Skill 是否教对了。核心 Skill（architecture-principles、code-standards、development-principles、product-design-principles）建议按本流程开发或优化。

- **RED（观察失败）**：在写/改 Skill 前，先设计一个最小测试场景（如「让 AI 实现订单创建接口」），**临时移除或禁用**相关 Skill，让 AI 执行该任务；记录 AI 的**具体错误行为**与**合理化借口**（如「简单 CRUD 不需要分层」）。
- **GREEN（写 Skill 解决问题）**：针对观察到的错误与借口，在 Skill 中增加**红旗清单**（可观察行为 + 常见借口，触达即停）与**强制流程/正确示例**；再启用 Skill 重跑同一场景，验证行为是否改变。
- **REFACTOR（优化）**：简化表述、补充示例与边界情况；若发现新的借口或错误，补充到红旗清单。

详细步骤与模板见 **docs/process/skill-development-workflow.md**。与 AI-DLC 一致：本流程产出的 Skill 对应可执行的 **Stage**，Plan 中「建议使用的 Skills」可引用本仓 Skill。

### 2.1 SKILL.md 基本结构

```markdown
---
name: <skill-name>
description: <一句话说明用途与触发场景。当用户说「…」时使用。>
---

# <Skill 标题>

## 角色与目标 / 核心原则
（本 skill 的定位与要达成的结果）

## 动作顺序 / 检查清单 / 原则要点
（按步骤或按检查项，可引用其他 skills）

## 产出物建议（可选）
（输出什么、格式建议）

## 工作结束后的默认动作
- **提交 GitHub**：任务或迭代完成后，默认执行或明确建议用户执行「提交到 GitHub」…

## 使用说明
（何时用、与哪些 skill 配合）
```

### 2.2 开发检查清单

- [ ] **frontmatter**：name 简短唯一，description 含「当用户说…时使用」类触发描述。
- [ ] **单一职责**：一个 skill 只负责一类能力（如「学习与开发 skill」而非「学习+写代码+部署」）。
- [ ] **可触发**：description 或使用说明里列出典型用户说法或场景，便于 Agent 匹配。
- [ ] **可串联**：如需依赖其他 skills，在「动作顺序」或「使用说明」中写明（如先 development-principles 再 code-standards）。
- [ ] **可执行**：原则与清单具体到可操作（避免空泛描述）。
- [ ] **收尾动作**：包含「工作结束后的默认动作」且含「提交 GitHub」建议。

### 2.3 与本仓库约定对齐

- 新 skill 放在 `.cursor/skills/<skill-name>/SKILL.md`。
- 若为「角色类」skill（如 role-xxx），在「动作顺序」中明确先引用的原则类 skill（如 architecture-principles、product-design-principles）。
- 保持与现有 skills 的段落风格一致（如「工作结束后的默认动作」格式统一）。

---

## 三、优化既有 Skill

### 3.1 优化维度

| 维度 | 检查项 |
|------|--------|
| **清晰度** | 标题与 description 是否一眼能懂？检查清单是否无歧义？ |
| **可触发性** | description 是否覆盖用户常见说法？是否补充「当用户说…时使用」？ |
| **可复用性** | 是否过度依赖单项目术语？可否抽成通用表述？ |
| **结构与一致** | 是否具备「原则/清单→产出物→默认动作→使用说明」？与本仓库其他 skill 是否统一？ |
| **可追溯** | 若引用了外部市场或规范，是否注明来源或链接？ |

### 3.2 优化动作顺序

1. **读一遍**：以「新用户」视角读 SKILL.md，标记含糊或冗余处。
2. **对照市场**：从 SkillsMP / Agent Skills Guide / Anthropic 同领域选 1 个优质 skill，对比结构与表述，列出可改进点。
3. **改描述**：优先改 frontmatter 的 description，确保触发场景明确。
4. **改清单**：将原则或检查项改为可执行语句，必要时拆条或合并。
5. **补收尾**：若无「工作结束后的默认动作」，补上并含「提交 GitHub」。
6. **更新使用说明**：写明与哪些 skill 配合、适用场景。

---

## 四、与本仓库现有 Skills 的关系

- **development-principles** / **code-standards**：开发流程与代码规范；本 skill 不替代二者，而是在「学技能、写技能、改技能」时使用。
- **role-developer** / **role-architect** / **role-product-designer**：各角色可引用本 skill，在用户提出「学习或优化 skills」时按本 skill 执行。
- 本 skill 产出物：新 SKILL.md、对既有 SKILL.md 的修改建议或直接修改。

---

## 与 memory/ 的联动（可选）

- 若当前仓库存在 `memory/` 目录，且本次**新增或优化了 skill**，可将本次变更摘要追加到 `memory/changelog.md`：日期、新增/优化了哪个 skill、主要变更点。
- 若为尚未验证的写法或待对比的市场示例，可先记入 `memory/scratchpad.md`，待验证后再写入长期记忆。
- 若不存在 `memory/`，则跳过本段，不报错。

---

## 工作结束后的默认动作

- **提交 GitHub**：任务或迭代完成后，默认执行或明确建议用户执行「提交到 GitHub」（`git add` → `git commit` → `git push`），避免工作结束未提交导致无法回滚；commit 信息需清晰、可与需求/任务关联（如 `chore(skills): 新增/优化 xxx skill`）。

---

## 使用说明

用户可说「从市场学 skills」「帮我开发一个 skill」「优化现有 skill」「写一个能学习和优化 skills 的 skill」，本 Skill 将按「学习→开发→优化」三部分执行；需要时先到 SkillsMP / Agent Skills Guide / Anthropic 查示例再落笔。**核心 Skill** 开发/优化时，须按 2.0 的 TDD 工作流（RED–GREEN–REFACTOR）执行，详见 **docs/process/skill-development-workflow.md**。
