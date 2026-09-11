# Task: FE-01 前端基础：i18n + 布局 + API 客户端 + SEO 基座

## Goal

为官网搭建可复用的前端基座：中英双语路由与文案、全局布局（头部/页脚/语言切换）、
符合契约的 API 客户端与登录态容器、站点品牌信息接入、SEO 基座与错误页。
后续页面任务直接在此基座上开发。

## Dependencies

- BE-01（`GET /api/v1/public/settings` 新字段、`/public/navigation`）；接口未就绪时按契约开发并使用内置兜底。
- 契约：`docs/api/frontend-api-contract.md` §1、§2、§3、§7。

## Scope

### 1. 依赖与配置

- 新增 `@nuxtjs/i18n`（选择与 Nuxt 4 兼容的最新版）并锁定版本。
- `nuxt.config.ts`：
  - i18n：`locales` 静态声明 `en`、`zh-CN`（语言包文件）；`strategy: 'prefix_except_default'`；
    `defaultLocale: 'en'`。
  - `runtimeConfig`：按契约 §7.1 配置 `apiProxyTarget`、`apiInternalBase`、`public.apiBase='/api'`。
  - `routeRules`：`/api/**` 代理到 `apiProxyTarget`（仅开发/生产同源代理用）。
- `app.config.ts`：Nuxt UI 主题色与基础设计 token（品牌主色、圆角、字号）。

> 说明：Nuxt i18n 的路由前缀必须在构建期确定，因此语言集合在前端静态声明；
> 运行期由后端 `settings.locales` 决定**启用**哪些语言与默认语言，未启用的语言在切换器中隐藏。

### 2. API 客户端（契约 §7.2）

- `app/composables/useApi.ts`：
  - SSR 用 `apiInternalBase` 绝对地址，客户端用 `public.apiBase`。
  - 自动附加 `Authorization: Bearer`；生成 `X-Request-ID`。
  - 解包信封，`code !== 0` 抛 `ApiError { status, code, message, requestId }`。
  - 401（20001/20002/20003/40006）单次 refresh 重试；refresh 自身失败不再重试。
- `app/composables/useAuth.ts`：access token 仅存内存（`useState`），提供
  `login/register/logout/refresh/loadMe` 与 `isAuthenticated`；SSR 安全。
- `app/utils/apiError.ts`：按契约 §3 将 `code` 映射到 i18n 文案，未知码兜底。
- 禁止组件内直接调用后端地址或读取 cookie。

### 3. 站点设置与品牌（契约 §4.1、§4.2）

- `app/composables/useSiteSettings.ts`：拉取 `/api/v1/public/settings?locale=`，带请求级缓存。
- `useNavigation(placement)`：拉取 `/api/v1/public/navigation?locale=&placement=`。
- 接口失败时使用内置兜底（站名、基础导航），页面仍可渲染，不得白屏。

### 4. 布局与页面壳

- `app/layouts/default.vue`：头部（Logo/站名、导航、语言切换、登录/注册或控制台入口、移动端抽屉）、
  页脚（导航、联系方式、社交链接、版权、备案）。
- `app/components/`：`AppHeader`、`AppFooter`、`LocaleSwitcher`、`AppLogo`、
  以及加载/空态/错误态基础组件。
- `app/error.vue`：404 与 500 布局，双语文案。
- 语言切换保持当前页面（切换同一路由的 locale 版本）。

### 5. SEO 基座（契约 §7.3）

- `app/composables/useSeo.ts`：按当前页覆盖 `title`/`description`/`og:*`，缺省回退站点默认值。
- 输出 `<html lang>`、canonical、`hreflang`（含 `x-default`）。
- `robots.txt`（public 静态文件）。

### 6. 文案与质量

- 语言包 `i18n/locales/en.json`、`zh-CN.json`；禁止组件内硬编码用户可见文本。
- ESLint 规则或脚本检查语言包 key 一致（en/zh-CN key 集合相同）。
- `pnpm lint`、`pnpm typecheck`、`pnpm build` 全绿。

## Out of scope

- 具体业务页面（首页/功能/价格/文档等）→ FE-02。
- 登录/注册/控制台页面 → FE-03。
- 联系表单 → FE-02。
- 修改后端任何代码。

## Files / areas

- `frontend/nuxt.config.ts`、`frontend/app.config.ts`、`frontend/package.json`
- `frontend/i18n/**`（语言包与配置）
- `frontend/app/composables/**`、`frontend/app/components/**`、`frontend/app/layouts/**`
- `frontend/app/error.vue`、`frontend/public/robots.txt`

## Acceptance criteria

- [ ] `pnpm dev` 下 `/`（英文）与 `/zh-CN`、`/zh-CN/features` 等路由可访问，语言切换停留在当前页。
- [ ] 头部/页脚品牌信息来自 `/api/v1/public/settings`；接口不可用时显示兜底且不报错。
- [ ] 所有请求经由 `useApi`，浏览器 Network 中无跨域请求、无后端直连地址。
- [ ] 401 时单次 refresh 重试生效；refresh 失败清空登录态。
- [ ] access token 不出现在 localStorage/sessionStorage/cookie。
- [ ] `hreflang`/canonical/`<html lang>` 正确输出。
- [ ] `pnpm build`、`pnpm lint`、`pnpm typecheck` 全部通过。

## How to verify

```bash
cd frontend
pnpm install
pnpm lint && pnpm typecheck && pnpm build
pnpm dev   # 手工核对 / 与 /zh-CN 路由、语言切换、页脚、404
```

预期：构建通过；两种语言页面正常；无跨域请求。

## Branch / base

- branch: `fe-foundation`
- base: `master-relay`
