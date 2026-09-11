# Task: FE-ANALYTICS 官网访问上报

## Goal

官网每次页面浏览上报一次到 `POST /api/v1/public/page-view`（契约 §4.10），供访问统计使用。

## Scope

- 新增客户端插件（如 `frontend/app/plugins/page-view.client.ts`）：
  - **仅客户端**运行；每次路由切换（`useRouter().afterEach` 或 `page:finish`）上报一次。
  - 上报内容：`{ path: route.fullPath 去掉 query/hash? 或 to.path, referrer: document.referrer, locale }`。
    建议 `path` 用 `to.path`（不含 query/hash），长度 ≤512。
  - 发送方式 fire-and-forget：优先 `navigator.sendBeacon`，不可用时 `$fetch(..., { keepalive: true })`；
    **不要阻塞导航**，失败静默（不报错、不提示）。
  - 跳过非公开页面（如 `/account` 等可选）——至少不要上报管理后台相关路径（本项目不含管理后台页面）。
- 遵守契约 §7.1：请求经 `/api`（`public.apiBase`）；不过度改造 `useApi`（上报是幂等 fire-and-forget，可直接用 `$fetch`）。

## Out of scope

- 后端实现（BE-ANALYTICS）、管理端工作台（ADMIN-DASHBOARD）。
- UV/会话/去重。

## Files / areas

- `frontend/app/plugins/**`（新增 page-view 插件）
- 必要时 `frontend/nuxt.config.ts`（仅在需要时）

## Acceptance criteria

- [ ] 公开页面之间切换会在客户端各上报一次（Network 可见 `POST /api/v1/public/page-view`）。
- [ ] 上报不影响页面渲染与导航性能；失败不产生可见错误。
- [ ] `pnpm lint && pnpm typecheck && pnpm build` 全绿。
- [ ] 不使用 `localStorage` 等持久存储；不触碰 token。

## How to verify

```bash
cd frontend
pnpm lint && pnpm typecheck && pnpm build
# 起后端(8100)+dev(用其它端口)，在页面间切换，观察 Network 的上报请求；
# 或到管理后台「访问统计」看 PV 是否增长（依赖 BE-ANALYTICS）。
```

## Branch / base

- branch: `fe-analytics`
- base: `master-relay`
