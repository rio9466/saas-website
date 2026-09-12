# Task: NEXT-03 认证与用户中心（注册/登录/验证/找回/账户）

## Goal

在 `next/` 上实现账号闭环与用户控制台，对接 BE-03 接口，与 `frontend/`（FE-03）功能对齐。

## 依赖

- NEXT-01（基座）、BE-03（用户接口，已合入）、契约 §2/§3/§5/§7。

## Scope

1. **认证页面**：注册、邮箱验证（query `email`+`token` 自动提交）、登录（`identifier`+密码，处理
   `40011`/`40006`/`42901`，`?redirect=` 仅站内相对路径）、忘记密码、重置密码；文案双语。
2. **登录态**：token 仅存内存；**会话恢复在客户端**（refresh cookie `Path=/api/v1/auth`，SSR 不带）；
   账号页首屏加载态，水合后 `POST /api/v1/auth/refresh` 恢复，失败跳登录。
3. **路由保护**：客户端守卫 `/account/**`，未登录跳 `/[locale]/login?redirect=<当前路径>`（**带语言前缀**）。
4. **用户中心** `/account`：概览、积分（流水表格，四位小数字符串，禁止浮点）、修改密码（成功后清空登录态跳登录）、
   资料编辑（P1）、登出。
5. **持久化**：语言与主题沿用 NEXT-01 的持久化（cookie/localStorage），登录/登出/跳转**不得**导致语言或主题重置
   （所有站内跳转用 i18n 链接/`redirect`，带 locale 前缀）。

## Out of scope

- 邮箱改绑、第三方登录、积分兑换、后端改动、`frontend/`、根文档。

## Files / areas

- `next/app/[locale]/{register,verify-email,login,forgot-password,reset-password}/**`、`next/app/[locale]/account/**`
- `next/lib/auth.ts`（登录态）等

## Acceptance criteria

- [ ] 注册→验证邮件链接（含 email+token）可跨设备打开并完成验证；登录→`/account` 可访问；未登录跳登录并回跳。
- [ ] 刷新页面登录态可恢复；改密后旧会话失效并跳登录；forgot/reset 全流程可用，令牌重放失效。
- [ ] `40005/40006/40010/40011/40016/42901` 均有双语文案；积分四位小数字符串正确展示。
- [ ] access token 不落持久存储；Network 无跨域请求。
- [ ] 登录/登出/跳转后语言与主题保持不重置。
- [ ] `pnpm lint && pnpm build` 全绿。

## How to verify

```bash
cd next
pnpm lint && pnpm build
# 起后端(8100)+dev(3200)：/register → 验证 → /login → /account → /account/points → 改密 → 重新登录；刷新保持登录
```

## Branch / base

- branch: `next-auth`
- base: `master-relay`
