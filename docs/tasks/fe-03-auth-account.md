# Task: FE-03 登录注册与用户控制台

## Goal

实现官网的账号闭环与用户控制台：注册、邮箱验证、登录/登出、忘记/重置密码、
用户资料、修改密码、积分余额与流水查询，全部对接 BE-03 接口。

## Dependencies

- FE-01（布局、i18n、`useApi`、`useAuth`）、BE-03（用户接口）、BE-02（无直接依赖）。
- 契约：`docs/api/frontend-api-contract.md` §2、§3、§5、§7。

## Scope

### 1. 认证页面

- `app/pages/register.vue`：用户名 + 邮箱 + 密码 + 确认密码；成功后：
  - `status === "pending_verification"` → 展示「已发送验证邮件」并提供重发按钮；
  - `status === "active"` → 跳转登录页并预填。
  - `40010` 注册关闭时隐藏入口并给出提示；`10001` 做字段级校验提示。
- `app/pages/verify-email.vue`：读取 query `email`、`token`（契约 §5.7），进入后自动提交
  `POST /api/v1/auth/verify-email`；成功显示确认并引导登录；`40016` 显示失效并提供重发。
- `app/pages/login.vue`：`identifier` + 密码；按 `username_login_enabled`/`email_login_enabled`
  提示可用登录方式；处理 `40011`（未验证，引导重发）、`40006`（禁用）、`42901`（限流）。
  支持 `?redirect=` 回跳（仅允许站内相对路径，防开放重定向）。
- `app/pages/forgot-password.vue`：邮箱 → 调用 forgot；无论结果都显示「若邮箱存在已发送」。
- `app/pages/reset-password.vue`：读取 query `email`、`token` + 新密码；成功后跳登录。

### 2. 登录态

- `useAuth`（FE-01）扩展：登录成功后写入内存 access token 并调用 `loadMe()`。
- **会话恢复必须在客户端进行**：refresh cookie 的 `Path=/api/v1/auth`，页面 SSR 请求
  （如 `/account`）不会携带该 cookie。因此账号相关页面首屏渲染加载态，水合后调用
  `POST /api/v1/auth/refresh` 恢复登录态，失败再跳登录。
- `app/middleware/auth.ts`：客户端路由中间件保护 `/account/**`，未登录跳
  `/login?redirect=<当前路径>`。

### 3. 用户控制台（`app/pages/account/`）

- 布局：侧边/顶部导航（概览、积分、修改密码、资料[P1]），移动端可用。
- `index.vue` 概览：昵称、头像、邮箱、注册时间、等级、积分余额；
  数据来自 `GET /api/v1/me`。
- `points.vue` 积分：余额、累计消耗、当前等级，流水表格（时间倒序、分页）；
  数据来自 `GET /api/v1/me/point-transactions`。积分按四位小数字符串展示，
  **禁止浮点转换**；正负变动有视觉区分。
- `security.vue` 修改密码：当前密码 + 新密码 + 确认；成功后清空登录态、
  提示「密码已修改，请重新登录」并跳转登录页（契约 §5.4）；`40005` 定位当前密码字段。
- `profile.vue`（P1）：昵称、头像 URL 编辑，`PATCH /api/v1/me`。
- 登出按钮：调用 `POST /api/v1/auth/logout` 后清空本地状态并回首页/登录页。

### 4. 文案与状态

- en/zh-CN 语言包补齐全部认证/控制台文案，key 集合一致。
- 所有错误按契约 §3 的 `code` 映射，未知码兜底；错误提示不得直接显示后端 `message`。
- 表单提交中禁用按钮，防重复提交。

## Out of scope

- 邮箱改绑（PRD Q5）。
- 第三方登录、积分兑换/充值、支付。
- 后端改动（接口/契约不符时报告 blocker）。

## Files / areas

- `frontend/app/pages/register.vue`、`verify-email.vue`、`login.vue`、
  `forgot-password.vue`、`reset-password.vue`
- `frontend/app/pages/account/**`（index/points/security/profile）
- `frontend/app/layouts/`（账号布局，如需要）
- `frontend/app/middleware/auth.ts`
- `frontend/app/composables/useAuth.ts`（扩展）、`useApi.ts`（按需）
- `frontend/i18n/locales/*.json`

## Acceptance criteria

- [ ] 注册 → 验证邮件链接可直接打开并完成验证（跨设备可用，query 含 email+token）。
- [ ] 注册（无需验证时）→ 登录 → `/account` 可访问；未登录访问 `/account` 跳登录并回跳。
- [ ] 刷新页面后登录态可恢复（客户端 refresh），无需重新登录。
- [ ] 修改密码成功后所有会话失效，跳登录；用旧密码登录失败。
- [ ] 忘记/重置密码全流程可用，令牌重放提示失效。
- [ ] 积分页正确展示余额与流水分页，数值与后端四位小数字符串一致。
- [ ] `40005`/`40006`/`40010`/`40011`/`40016`/`42901` 均有正确双语文案。
- [ ] access token 不落任何浏览器持久存储；Network 无跨域请求。
- [ ] `pnpm lint && pnpm typecheck && pnpm build` 全绿。

## How to verify

```bash
cd frontend
pnpm lint && pnpm typecheck && pnpm build
pnpm dev
# 手工核对（后端已启动）：
#   /register → 邮件验证（或直接验证流程）
#   /login → /account → /account/points → /account/security
#   修改密码后重新登录；登出后 /account 跳登录
```

预期：全流程可用；刷新保持登录；改密后旧会话失效。

## Branch / base

- branch: `fe-auth-account`
- base: `master-relay`
