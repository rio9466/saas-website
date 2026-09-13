# 采用 Orca 工作流（Adopting）

> **参考实现（范本）**：<https://github.com/rio9466/saas-website>
> 该仓库就是本工作流的一次完整落地：多 pi 协作、Orca worktree、分支治理、任务派发与验收。
> 新项目可以直接照抄这套文档骨架与流程。

## 1. 这是什么

一套「**多 pi 协作 + Orca 托管 worktree + 分支治理**」的开发流程：

- 用 Orca 为每个分支开一个独立 worktree；一个 worktree = 一个分支 = 一个 agent 主写。
- 三个角色分工（对话 / 编排 / 执行），各自钉在固定分支上，职责不重叠。
- 所有协作围绕**文档**进行：PRD → 契约 → 任务文档 → 提交与验收证据。
- 发布：`master-relay` 合入 `master`，打标签。

## 2. 分支模型

| 分支 | 作用 | 谁能动 |
| ---- | ---- | ------ |
| `master` | 发布基线，只读 | 只有“对话 pi”，且需用户明确同意才合并 |
| `master-relay` | AI 集成分支（由对话 pi 管理） | 对话 pi（任务合回这里） |
| `<task>` | 每个任务一条临时分支 | 执行 pi |

规则：新任务分支**只从 `master-relay` 切**，合回 `master-relay`；不直接提交 `master`。

## 3. 两个角色

原「编排/规划 pi」已并入对话 pi，不再有独立的 relay pi；`master-relay` 仍是集成分支（由对话 pi 管理）。

- **对话 pi**（跑在 `master` 主检出）：对话/决策、**规划（PRD/契约/任务拆分、维护台账）**、
  **项目初始化与老项目接入**、worktree 派发、独立验收、所有 git 合并与发布。
  不写业务代码。持有**本地持久记忆**（见 §6）。用户允许时可做任意 git 操作。
- **执行 pi**（跑在任务分支）：只实现一个任务、自测、回报「命令 + 结果」。
  不改契约/跨领域文档，**永不合并**（既不进 `master-relay` 也不进 `master`）。

> worktree 初始化与 Orca 基础指令，见参考项目的 `ORCA_WORKFLOW.md` §2「Worktree and Orca basics」。

## 4. 必须的文档支撑（复制这一套）

| 文档 | 作用 | 归属 |
| ---- | ---- | ---- |
| `AGENTS.md`（根） | 编码规范 + 文档归属 + 分支约束 + 任务认领 + 角色 | 主线 |
| `ORCA_WORKFLOW.md` | 完整流程：分支模型 / 角色 / 派发 / 认领 / 合并 / 持久记忆 | 主线 |
| `docs/README.md` | 文档分类与归属索引 | 主线 |
| `docs/prd/` | 产品需求（PRD） | 主线 |
| `docs/api/*.md` | 前后端 API 契约（若有前后端） | 主线 |
| `docs/tasks/README.md` + `STATUS.md` | 任务索引与状态台账 | 主线 |
| `docs/tasks/<ID>.md` | 每个任务一份（独立、可执行） | 任务分支 |
| 各区域 `AGENTS.md` | 每个区域的编码规则 | 该区域分支 |
| `CONVERSATION_MEMORY.md` | 对话 pi 的本地记忆（**gitignored，不提交**） | 本地 |

> 关键点：`AGENTS.md` / `ORCA_WORKFLOW.md` 是“强规定”，agent 打开就自动读到；其余按需。

## 5. 项目初始化清单（新项目 / 老项目）

> **第一步：先问用户——这是「新项目」还是「要接入工作流的老项目」？** 再分下面两条路径。

### A. 新项目

1. **建仓库**：`master` 为默认分支；推送到远端。
2. **抄治理文档**：复制 `AGENTS.md`、`ORCA_WORKFLOW.md`、`.gitignore`，以及 `docs/` 骨架
   （`docs/README.md`、`docs/prd/`、`docs/api/`、`docs/tasks/README.md` + `STATUS.md`）。
3. **改占位**（务必替换）：项目名/仓库路径/远端；**区域目录**（见 §7）；端口、数据库、本地配置约定；
   `docs/prd/*`、`docs/api/*` 换成你的内容。
4. **建集成分支**：`git branch master-relay master`（并按需在 Orca 建 worktree）。
5. **定义区域**：确定项目的 area，每个 area 放一个 `AGENTS.md`，在根 `AGENTS.md`/`ORCA_WORKFLOW.md` 登记。
6. **建本地记忆**：仓库根 `CONVERSATION_MEMORY.md` 并加进 `.gitignore`。
7. **开始**：对话 pi 写 PRD → 拆任务文档 → 建任务分支/worktree → 派发执行 pi → 验收合并。

### B. 老项目接入工作流

1. **审计**：确认默认分支、目录结构、技术栈、构建/测试命令、端口与依赖（数据库/缓存等）。
2. **补治理文档**：加入/改造 `AGENTS.md`、`ORCA_WORKFLOW.md`、`docs/` 骨架与各区域 `AGENTS.md`。
3. **建集成分支**：从默认分支建 `master-relay`；任务从 `master-relay` 切、合回 `master-relay`。
4. **打基线标签**：在当前状态打一个 `vX.Y.Z`，作为接入点（此前提交与本次接入解耦）。
5. **建本地记忆**：`CONVERSATION_MEMORY.md`（git-ignored），记录项目事实、坑与约定。
6. **开始**：对话 pi 写 PRD/任务 → 建任务 worktree → 派发执行 pi → 验收合并。

## 6. 对话 pi 的持久记忆

**优先写入 Obsidian**（如果用户机器上装了）：

1. 检测 Obsidian：macOS 下 `/Applications/Obsidian.app` 和/或
   `~/Library/Application Support/obsidian/obsidian.json`（列出 vault，`open:true` 为当前 vault）。
2. 在当前 vault 里找 `CONVERSATION_MEMORY.md`：不存在就**新建**；存在就**合并到对应小节**（不要覆盖其它小节）。
   用 Obsidian **双链 `[[...]]`** 组织成「枢纽 + 子笔记」（如 [[用户偏好]]、[[项目事实]]、[[约定与坑]]、[[轻量决策]]，
   子笔记回链枢纽），这样在**关系图谱**里有连接、可跳转。
3. 之后所有记忆写入都放这个笔记（它是主存储）。

**回退**：没装 Obsidian 时，用仓库根目录的本地、git-ignored `CONVERSATION_MEMORY.md`。

记录内容：

- 用户偏好与习惯（语言、简洁度、是否先给方案、合并是否需授权、命名/端口约定、派发偏好）；
- 项目事实（分支模型、发布基线、端口、数据库、外部依赖）；
- 约定与坑（例如 pnpm 版本、忽略规则、代理/Origin 之类踩过的坑）；
- 轻量决策与理由。

会话开始读取，用户给偏好时即时更新，检查点更新；保持精炼。

## 7. 区域是项目自定义的

> **区域按你的项目定义；下面是本项目曾用的区域示例。**

| 示例区域 | 技术栈 | 说明 |
| -------- | ------ | ---- |
| `frontend/` | Nuxt 4 + Nuxt UI + pnpm | 官网前端（模板一） |
| `next/` | Next.js + React + shadcn/ui | 官网前端（模板二） |
| `backend/` | easy-admin（Go + 管理后台） | 后端与管理后台 |

换成你的目录即可；区域之间**互不改**，跨区域要拆任务或显式例外。

## 8. 可复制文件速查（从参考项目拿）

```
AGENTS.md
ORCA_WORKFLOW.md
.gitignore
docs/README.md
docs/prd/            # 换成你的 PRD
docs/api/            # 换成你的契约（示例：frontend-api-contract.md）
docs/tasks/README.md
docs/tasks/STATUS.md # 清空任务行
<area>/AGENTS.md     # 每个区域一个
```

## 9. 约定建议

- **端口**：不同环境/模板用不同端口，避免开发分支互相占用；本地运行配置（`config.local.toml`、
  `vite.config.local.ts`、`.env*`）一律 git-ignore，只提交 `.env.example`。
- **契约优先**：前后端以一份契约文档为准；执行者不得私改请求/响应形状，发现缺口先报告。
- **验收**：执行者必须给「命令 + 结果」，对话 pi 独立复核后才合并。
- **发布**：`master-relay` → `master`，打 `vX.Y.Z` 标签。
