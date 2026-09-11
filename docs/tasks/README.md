# 任务索引（官网 + 用户中心）

由 `master-relay` 根据 [`docs/prd/saas-website-prd.md`](../prd/saas-website-prd.md) 与
[`docs/api/frontend-api-contract.md`](../api/frontend-api-contract.md) 生成。

任务状态台账见 [`STATUS.md`](STATUS.md)，由 `master-relay` 维护；执行者不要直接改本目录。

## 契约优先

所有任务以 `docs/api/frontend-api-contract.md`（下称「契约」）为准。执行者**不得**自行更改
请求/响应形状；发现契约缺口时先停下，向 `master-relay` 报告并由其更新契约。

## 执行顺序

```
BE-01 content-foundation ──▶ BE-02 contact-inbox
        │                          │
        ├──▶ BE-04 admin-content-ui ◀┘
        │
        └──▶ (公开接口就绪)

BE-03 user-console-api  ──▶ FE-03 auth-account
                                 
FE-01 foundation ──▶ FE-02 public-pages ──▶ FE-03 auth-account
```

| ID | 文档 | 分支 | Base | 依赖 |
| -- | ---- | ---- | ---- | ---- |
| BE-01 | [`be-01-content-foundation.md`](be-01-content-foundation.md) | `be-content-foundation` | `master-relay` | — |
| BE-02 | [`be-02-contact-inbox.md`](be-02-contact-inbox.md) | `be-contact-inbox` | `master-relay` | BE-01 |
| BE-03 | [`be-03-user-console-api.md`](be-03-user-console-api.md) | `be-user-console-api` | `master-relay` | — |
| BE-04 | [`be-04-admin-content-ui.md`](be-04-admin-content-ui.md) | `be-admin-content-ui` | `master-relay` | BE-01、BE-02 |
| FE-01 | [`fe-01-foundation-i18n.md`](fe-01-foundation-i18n.md) | `fe-foundation` | `master-relay` | — |
| FE-02 | [`fe-02-public-pages.md`](fe-02-public-pages.md) | `fe-public-pages` | `master-relay` | BE-01、BE-02、FE-01 |
| FE-03 | [`fe-03-auth-account.md`](fe-03-auth-account.md) | `fe-auth-account` | `master-relay` | BE-03、FE-01、FE-02 |

## 通用验收要求（所有任务）

- 只改本任务「Files / areas」列出的目录；跨区域改动必须先报告。
- 后端：`cd backend/server && make check`（fmt/vet/test/build）通过；接口变更同步 `docs/openapi.yaml`。
- 管理后台：`cd backend/server/admin && pnpm build` 通过。
- 前端：`cd frontend && pnpm build && pnpm lint && pnpm typecheck` 通过。
- 交付说明中给出「执行的命令 + 结果」，不接受“应该可以”。

## 未决问题

`docs/prd/saas-website-prd.md` §13 的开放问题（支付、语言集合、邮箱改绑、图片存储、验证码等）
未决策前，按该表的「默认假设」执行；不要扩大范围。
