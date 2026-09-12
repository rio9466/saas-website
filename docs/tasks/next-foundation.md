# Task: NEXT-01 基座（Next.js 16 + shadcn/ui：i18n/主题/请求代理/布局）

## Goal

在仓库根新建 `next/`（与 `frontend/`、`backend/` 同级），作为第二套前端模板的基座：
Next.js 16 + React 19 + Tailwind v4 + shadcn/ui，双语 + 暗色主题（**均持久化**），
按契约 §7 实现同源 `/api` 请求与 API 客户端。**独立 pnpm 项目**（不与 `frontend/` 共享 workspace）。

## 技术选型

- Next.js 16（App Router / RSC）+ React 19 + TypeScript + ESLint。
- Tailwind CSS v4 + **shadcn/ui 4.x**（`pnpm dlx shadcn@latest init`）。
- **next-themes**（暗色模式，持久化；SSR 加 `suppressHydrationWarning` 防闪烁）。
- **next-intl**（i18n）：`en`（默认，无前缀）+ `zh-CN`（`/zh-CN/...`），语义对齐现有 `prefix_except_default`；
  **语言选择持久化到 cookie**（`NEXT_LOCALE`），下次进站按 cookie 生效。
- **lucide-react**（shadcn 默认图标）。
- 包管理 pnpm；dev 端口 **3200**。

## 关键实现

1. **脚手架**：在仓库根执行 `pnpm create next-app@latest next ...`（TS + Tailwind + ESLint + App Router + `@/*` 别名），
   然后 `pnpm dlx shadcn@latest init` 并加入基础组件（button/card/dropdown-menu/input/form/label 等）。
2. **请求（契约 §7.1/§7.2）**：
   - 浏览器只访问同源 `/api/**`：用 `app/api/[...path]/route.ts` **运行时**转发到 `API_PROXY_TARGET`
     （读 env，逐请求转发，保留 query/body/Cookie/Set-Cookie；**不要用构建期固定的 next.config rewrites**）。
   - SSR 直连后端内网：`API_INTERNAL_BASE`。
   - `lib/api.ts`：统一封装——解包信封、`code!==0` 抛 `ApiError`、附加 Bearer（内存）、`X-Request-ID`、
     401 单次 refresh 重试（refresh 自身不重试）；token 只存内存，禁止 localStorage/cookie。
3. **i18n**：`next-intl` 中间件 + 路由；**站内链接必须用 next-intl 的 `Link`/`redirect`**（等价 Nuxt 的 `localePath`，
   避免 bare `/x` 丢语言）；`<html lang>`、`hreflang`、canonical。
4. **主题**：`next-themes` + shadcn，切浮尘按钮；持久化（localStorage/cookie），首屏不闪。
5. **布局**：`app/[locale]/layout.tsx`（Header：Logo/站名、导航、语言切换、主题切换、登录/注册/控制台；Footer：导航/联系方式/版权），
   站点信息来自 `GET /api/v1/public/settings`（无数据用兜底）。
6. **首页占位**：`app/[locale]/page.tsx` 简单渲染站点信息即可（正式公开页在 NEXT-02）。
7. **SEO 基座**：`metadata`/`generateMetadata`、`robots.ts`（sitemap 在 NEXT-02）。
8. **`.gitignore`**：`.next/`、`node_modules/`、`.env*`（提供 `.env.example`：`API_PROXY_TARGET`、`API_INTERNAL_BASE`）。
9. **`next/AGENTS.md`**：区域规则（仅 next 任务分支可改；复用契约；pnpm；`pnpm lint && pnpm build` 验收；端口 3200；不含后端代码）。

## Out of scope

- 公开页面/文档/联系（NEXT-02）、认证与用户中心（NEXT-03）、后端改动、`frontend/`、根文档。
- 访问上报埋点（NEXT-02 处理）。

## Files / areas

- 全新 `next/**`（脚手架产物 + 上述实现 + `next/AGENTS.md`）

## Acceptance criteria

- [ ] `next/` 为独立 pnpm 项目；`pnpm install && pnpm lint && pnpm build` 全绿。
- [ ] `pnpm dev -p 3200`（或脚本固定 3200）：`/` 与 `/zh-CN` 正常渲染，语言切换有效且**刷新/重开保持**。
- [ ] 主题切换生效且刷新/重开保持，首屏不闪。
- [ ] 浏览器请求只走同源 `/api`（代理到 `API_PROXY_TARGET`）；`GET /api/v1/public/settings` 经代理返回 200；
      `POST`（带 body）经代理正常（不能 502）。
- [ ] 站内链接不出现裸 `/x`（都经 i18n 链接）；token 不落持久存储。
- [ ] `next/AGENTS.md` 已建立。

## How to verify

```bash
cd next
pnpm install
pnpm lint && pnpm build
API_PROXY_TARGET=http://127.0.0.1:8100 API_INTERNAL_BASE=http://127.0.0.1:8100 pnpm dev -p 3200
# 浏览器：/ 与 /zh-CN；切语言+刷新；切主题+刷新；Network 看 /api 代理
```

## Branch / base

- branch: `next-foundation`
- base: `master-relay`
