# Task: FIX-NEXT-DEV-ORIGIN 允许 127.0.0.1 访问 Next dev 资源

## Problem

用 `http://127.0.0.1:3200` 访问 `next/` dev 时，浏览器报：

```
WebSocket connection to 'ws://127.0.0.1:3200/_next/hmr?id=...' failed
```

dev 日志：

```
⚠ Blocked cross-origin request to Next.js dev resource /_next/hmr from "127.0.0.1".
To allow this host in development, add it to "allowedDevOrigins" in next.config.js and restart:
  allowedDevOrigins: ['127.0.0.1'],
```

根因：Next.js 16 默认拦截跨源 dev 资源（HMR/WS）；用 `localhost` 访问不受影响，用 `127.0.0.1` 被 403 拦。
（已实测：`Origin: http://127.0.0.1:3200` → 403；`Origin: http://localhost:3200` → 放行。）

## Fix

在 `next/next.config.ts` 增加**仅开发用**的白名单：

```ts
const nextConfig: NextConfig = {
  // Dev-only: Next blocks cross-origin dev resources (HMR/WS) by default.
  // Allow accessing the dev server via 127.0.0.1 as well as localhost.
  allowedDevOrigins: ["127.0.0.1", "localhost"],
  // ...existing config
};
```

- `allowedDevOrigins` 只影响开发服务器，不影响生产构建。
- 不改其它配置；不引入构建期 `rewrites`。

## Out of scope

- 其它文件；`frontend/`、`backend/`、根文档。

## Files / areas

- `next/next.config.ts`

## Acceptance criteria

- [ ] `http://127.0.0.1:3200` 访问时，dev 日志不再出现 `Blocked cross-origin request ... /_next/hmr`，浏览器无 HMR WebSocket 报错。
- [ ] `http://localhost:3200` 仍正常。
- [ ] `cd next && pnpm lint && pnpm build` 全绿。

## How to verify

```bash
cd next
pnpm lint && pnpm build
API_PROXY_TARGET=http://127.0.0.1:8100 API_INTERNAL_BASE=http://127.0.0.1:8100 pnpm dev -p 3200
# 用 127.0.0.1:3200 打开：控制台无 WebSocket 报错；dev 日志无 Blocked cross-origin；页面/导航/主题/语言正常
```

## Branch / base

- branch: `fix-next-dev-origin`
- base: `master-relay`
