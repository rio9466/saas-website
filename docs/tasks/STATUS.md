# 任务状态台账

由 `master-relay`（编排）维护。**执行者不要直接改本文件**；认领与进度通过任务分支的提交信息
（`chore(<ID>): claim task`、`feat(<ID>): ...`）与完成回执反馈，由编排侧更新这里。

状态取值：

- `todo` — 已分配（分支/worktree 已建），待认领
- `in-progress` — 已认领，进行中
- `in-review` — 执行完成，待编排审阅
- `done` — 已合入 `master-relay`
- `blocked` — 受阻（在「证据 / 说明」里写原因）

| ID    | 任务                                             | 分支                     | Base           | 依赖                    | 状态 | 证据 / 说明            |
| ----- | ------------------------------------------------ | ------------------------ | -------------- | ----------------------- | ---- | ---------------------- |
| BE-01 | 内容基础模型 + 公开内容接口 + 管理端内容 CRUD     | `be-content-foundation`  | `master-relay` | —                       | done | 已合入 38e2e9c          |
| BE-02 | 联系表单与收件箱                                  | `be-contact-inbox`       | `master-relay` | BE-01                   | done | 已合入 2b20723（分支已删） |
| BE-03 | 用户控制台 API                                    | `be-user-console-api`    | `master-relay` | —                       | done | 已合入 5a0f461（分支已删） |
| BE-04 | 管理端内容 UI                                     | `be-admin-content-ui`    | `master-relay` | BE-01、BE-02            | done | 已合入 f1b57cb+2e6448d（分支已删） |
| FE-01 | 前端基础：i18n + 布局 + API 客户端 + SEO          | `fe-foundation`          | `master-relay` | —                       | done | 已合入 f83c373          |
| FE-02 | 公开页面（首页/功能/价格/关于/文档）              | `fe-public-pages`        | `master-relay` | BE-01、BE-02、FE-01     | done | 已合入 fa9a792（分支已删） |
| FE-03 | 登录 / 注册 / 用户中心                            | `fe-auth-account`        | `master-relay` | BE-03、FE-01、FE-02     | done | 已合入 c251ffd（含 /api 代理 502 修复） |
| HOTFIX-01 | 运行时错误（Nuxt 500 + admin .env 崩溃）    | `hotfix-runtime-errors`  | `master-relay` | —                       | done | 已合入 5e5f809          |
| FIX-FE-UX | 前端体验：语言切换/记忆、表单红框、页脚居中 | `fix-fe-ux` | `master-relay` | — | todo | worktree 已建，待认领 |

## 更新规则

- 编排侧：建任务分支后置 `todo`；收到认领回执置 `in-progress`；收到完成回执置 `in-review`；
  合入 `master-relay` 后置 `done` 并写上 merge commit。
- 同一任务 ID 只允许一个分支在做；状态已是 `in-progress` 的任务不得重复开工。
- 任务分支在任务 `done` 之后才删除。
