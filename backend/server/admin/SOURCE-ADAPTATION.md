# Source Adaptation Record — easy-admin admin client

This directory (`server/admin/`) is derived from the
[`rio9466/pure-admin-thin`](https://github.com/rio9466/pure-admin-thin)
executable base and follows the UI/interaction language of
[`pure-admin/vue-pure-admin`](https://github.com/pure-admin/vue-pure-admin).

## Imported base

| Item | Value |
| --- | --- |
| Repository | `https://github.com/rio9466/pure-admin-thin.git` |
| Resolved commit | `f0ff132561ab684bb78379239adf29f9a38ac7f1` |
| Release | `6.2.0` (single tag-less release commit `release: update 6.2.0`, 2025-10-30) |
| License | MIT — see `LICENSE` (upstream © pure-admin, retained) |
| Package manager | pnpm (>= 9), lockfile `pnpm-lock.yaml` |
| Node engine | `^20.19.0 || >=22.13.0` (`.nvmrc` = v22.20.0) |
| UI reference | `pure-admin/vue-pure-admin` @ commit `c0fd7419c58689c7b1055ef92575a9beea7314d3`（release `6.2.0`, 2025-10-16, MIT） |

UI reference details: page/component patterns for administrator management, role/
permission management, and the two-panel table/dialog interaction are adapted from
`vue-pure-admin` at the pinned commit above, with per-page records in the
[business page evidence](#business-page-ui-evidence) section below. The thin base
provides the executable shell (layout, router, stores, Element Plus wiring, `@pureadmin/*`
components); the full project provides the compatible v6-era interaction language.

No nested Git metadata was imported. The upstream 7.x line was not used because
its router, state, build, and TypeScript versions differ (product decision in
`docs/admin-rbac-prd.md`).

## Material compatibility changes

| Source (upstream path) | Destination | Change |
| --- | --- | --- |
| `mock/*` | removed | No mock runtime: login/refresh/async-routes mocks deleted; `vite-plugin-fake-server` dependency and plugin removed. |
| `src/api/routes.ts` | removed | Remote/async route API deleted; menus/routes are local and filtered by `/me`. |
| `src/utils/sso.ts` | removed | SSO sample deleted (non-goal). |
| `src/utils/auth.ts` | adapted | Access token kept in memory only; no `js-cookie`/localStorage token persistence; `js-cookie` dependency removed. |
| `src/store/modules/user.ts` | rewritten | Real login/refresh/logout//me/password APIs; roles/permissions from `/me` in memory; cold-start `restoreSession()`; no storage persistence. |
| `src/utils/http/index.ts` | rewritten | `/api/v1/admin` base URL, Vite `/api` proxy in dev, same-origin `/api/v1` in prod; single-flight refresh (`RefreshGate`), original request retried at most once, refresh failure clears auth state and redirects to login; login/refresh whitelisted from the refresh loop. |
| `src/router/index.ts`, `src/router/utils.ts` | adapted | Local routes with `roles`/`permissions` meta; `initRouter()` is local-only (no remote menu request); route guard restores session on cold start and 403s unauthorized direct navigation. |
| `src/layout/components/lay-notice/*` | removed | Notification center deferred (no runtime/requests/UI). |
| `src/views/permission/*`, `src/views/welcome/*` | removed | Demo permission pages and template dashboard replaced by product pages. |
| `public/logo.svg`, `src/assets/login/avatar.svg`, `favicon` | replaced | Neutral easy-admin mark; `src/assets/user.jpg` sample avatar removed. |
| `index.html`, `public/platform-config.json`, `build/info.ts`, `README*` | replaced | Product naming/title; `CachingAsyncRoutes` removed; build banner de-branded. |
| `vite.config.ts` | adapted | `/api` → `http://127.0.0.1:8080` proxy (no permissive CORS). |
| `package.json` | adapted | name `easy-admin-admin`; added `generate:api` (openapi-typescript from `server/docs/openapi.yaml`) and `test` (vitest) scripts; removed mock/token-cookie deps. |

## Business page UI evidence

Upstream: `pure-admin/vue-pure-admin` @ `c0fd7419c58689c7b1055ef92575a9beea7314d3`（release 6.2.0）.

| Destination (server/admin) | Upstream source path (v6.2.0) | Material adaptation |
| --- | --- | --- |
| `src/views/system/administrator/index.vue` + `src/api/administrators.ts` | `src/views/system/user/index.vue`, `src/views/system/user/form/index.vue`, `src/views/system/user/form/role.vue`, `src/views/system/user/utils/{hook.tsx,rule.ts,types.ts}` | 上/下方两个兄弟面板（筛选/操作面板 + 表格/分页面板）布局、表格列+行内弹窗交互参照上游；数据源改为 SRV-003 契约（`/administrators`），角色分配弹窗改为多选角色代码（无组织树），增补启用/禁用二次确认、密码重置确认、自助保护禁用与后端失败状态。 |
| `src/views/system/role/index.vue` + `src/api/roles.ts` | `src/views/system/role/index.vue`, `src/views/system/role/form.vue`, `src/views/system/role/utils/{hook.tsx,rule.ts,types.ts}` | 列表+新建/编辑/分配权限弹窗交互参照上游；权限分配改为本地权限目录（`/permissions`）分组勾选框而非上游 tree/table 权限矩阵，普通管理员只读、`admin.role.manage`（仅 super_admin）才展示管理动作。 |
| `src/views/system/audit/index.vue` + `src/api/audit.ts` | 上游无同构页面（`src/views/monitor/logs/login/index.vue` 仅登录日志） | 交互沿用同一两面板表格语言；数据只读自 `/audit-events`，无变更/导出控件。 |
| `src/views/dashboard/index.vue` | `src/views/welcome/index.vue`（模板欢迎页） | 移除模板文案，改为真实 `/me` 会话信息 + 诚实空态，无伪造指标。 |
| `src/views/profile/index.vue`、`src/views/login/index.vue` | `src/views/login/index.vue`（上游）；个人中心参照同布局组件 | 登录页沿用上游版式但移除演示账号并接入真实接口；个人中心展示 `/me` 资料 + 修改密码（12-72 字节、二次确认、成功后强制重登）。 |

## License

- `LICENSE` — upstream MIT license text retained as required by the upstream
  project. Product UI and runtime contain no template branding, sponsorship,
  docs/GitHub links, or demo identities.
