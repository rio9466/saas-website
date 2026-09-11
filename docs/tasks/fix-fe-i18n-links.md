# Task: FIX-FE-I18N 内部跳转本地化 + 用 Pinia 持久化语言/主题

## Goal

修复「切换语言后点击站内链接/登录/退出又变回默认语言」的问题，并用 Pinia 持久化统一记录
用户的**语言**与**主题**选择。只改 `frontend/**`。

## 根因（必读）

项目 i18n 策略为 `strategy: 'prefix_except_default'`（默认语言无前缀）。
**Nuxt i18n 不会自动给 `<NuxtLink to="/x">` 加语言前缀**，必须用 `useLocalePath()` / `<NuxtLinkLocale>`。
当前全项目大量使用裸站内路径，全部指向默认语言（en）：

- Header：`UHeader to="/"`、Logo `<NuxtLink to="/">`（用户截图点击点）、导航 `:to="item.url"`（后端下发 `/features` 等）、`/account`、`/login`、`/register`
- Footer 导航 `:to="item.url"`；`DocsLayout to="/docs"`；首页/价格/功能区的 CTA（后端下发 `cta_url`/`primary_cta_url`）
- 程序化跳转：登出 `router.push('/')`、登录 `router.push(safeRedirect(...) || '/account')`、注册/改密/重置成功 `{path:'/login'}`、中间件 `navigateTo({path:'/login'})`、`error.vue clearError({redirect:'/'})`

另外：客户端跳转到 `/` 不触发 `detectBrowserLanguage`（它只在 SSR 根请求判定），所以 cookie 记忆也救不回来。

## 方案

### A. 内部跳转全面本地化（核心）
- 统一 `const localePath = useLocalePath()`：
  - 所有站内 `to="/x"` → `:to="localePath('/x')"`。
  - 程序化：`router.push(localePath('/account'))`、`navigateTo(localePath('/login'))`、`clearError({ redirect: localePath('/') })`。
  - 中间件 `redirect` 用 `to.fullPath`（已含前缀），fallback 用 `localePath('/account')`。
- **后端下发的站内链接**（导航、页脚、CTA）统一本地化：`item.url.startsWith('/') ? localePath(item.url) : item.url`（`http(s)://`、`mailto:`、`tel:` 等原样）。
- 新增可复用 `app/components/AppLink.vue`（内部用 `<NuxtLinkLocale>`）与 `app/composables/useLocalizedUrl.ts`（判断并本地化站内 URL），后续新代码统一走它们。
- Nuxt UI 的 `UButton`/`UHeader` 的 `to` 只接受字符串，传 `localePath(...)` 结果。

### B. 语言持久化（Pinia）
- 依赖：
  ```
  pnpm add pinia @pinia/nuxt pinia-plugin-persistedstate
  pnpm add -D @pinia-plugin-persistedstate/nuxt
  ```
- `nuxt.config.ts`：`modules` 增加 `@pinia/nuxt`、`@pinia-plugin-persistedstate/nuxt`；`pinia: { storesDirs: ['./app/stores/**'] }`。
- `app/stores/preferences.ts`：
  ```ts
  export const usePreferencesStore = defineStore('preferences', {
    state: () => ({ locale: '' as string, colorMode: '' as '' | 'light' | 'dark' | 'system' }),
    persist: { key: 'saas-preferences', storage: piniaPluginPersistedstate.cookies() }
  })
  ```
- `app/plugins/preferences.ts`：启动时读 store → `setLocale(store.locale)`（非空时）、应用主题。
- `LocaleSwitcher.switchTo`：`await setLocale(code)` 后写 `prefs.locale = code`；并保持 Nuxt i18n 的 `detectBrowserLanguage` cookie 一致（该 cookie 负责 SSR 路由正确性）。
- 兜底：客户端进入根路径时若 store.locale 与当前不同，`setLocale(store.locale)`。

### C. 主题持久化（Pinia + color-mode cookie）
- `nuxt.config.ts` 增加：
  ```ts
  colorMode: { preference: 'system', fallback: 'light', storage: 'cookie', storageKey: 'nuxt-color-mode', classSuffix: '' }
  ```
- 新增 `app/components/ThemeToggle.vue`：切换时同时写 `prefs.colorMode` 与 `useColorMode().preference`；在 Header 替换原 `UColorModeButton`（或包一层保持一致）。
- 以 store 为启动时的记录来源，SSR 用 cookie 直接渲染，避免闪烁。

### D. 一致性
- 启动插件按 store 应用 locale + 主题；SSR 由 cookie 直接渲染正确语言与主题。

## Out of scope

- 后端、管理后台、契约、根文档、`docs/tasks/**`（本任务文档除外）。

## Files / areas

- 新增：`frontend/app/stores/preferences.ts`、`frontend/app/plugins/preferences.ts`、
  `frontend/app/components/AppLink.vue`、`frontend/app/components/ThemeToggle.vue`、
  `frontend/app/composables/useLocalizedUrl.ts`
- 修改：`frontend/nuxt.config.ts`、`frontend/package.json`、`frontend/pnpm-lock.yaml`、
  `frontend/app/components/{AppHeader,AppFooter,DocsLayout,LocaleSwitcher,HomeSection,PricingPlanCard,FeatureCard}.vue`、
  `frontend/app/pages/**`、`frontend/app/layouts/account.vue`、`frontend/app/error.vue`、`frontend/app/middleware/auth.ts`

## 运行环境

- 后端已在跑 `http://127.0.0.1:8100`；3100/8849 已被占用，自测请用别的端口并把
  `NUXT_API_PROXY_TARGET`/`NUXT_API_INTERNAL_BASE` 指到 8100，不要动 3100/8849。

## Acceptance criteria

- [ ] `/zh-CN/**` 下点击 Header 标题/Logo、导航项、Footer 导航、首页/价格 CTA、登录/注册/退出/改密/中间件跳转，**始终保持中文**；英文同理。
- [ ] 语言选择刷新、重开浏览器后仍生效（Pinia 持久化 + i18n cookie）。
- [ ] 主题选择刷新、重开浏览器后仍保持，且首屏不闪（cookie 存储）。
- [ ] 站内链接不再有裸 `to="/..."`（外链除外）；后端下发的站内 URL 也被本地化。
- [ ] `pnpm lint && pnpm typecheck && pnpm build` 全绿。

## How to verify

```bash
cd frontend
pnpm install
pnpm lint && pnpm typecheck && pnpm build
# 起后端 + pnpm dev，手工核对：
#   /zh-CN → 点标题/导航/页脚/CTA/登录/退出/改密 → 始终中文
#   /en → 同样操作始终英文
#   切语言后刷新、重开浏览器再进 → 语言保持
#   切主题后刷新、重开浏览器 → 主题保持且不闪
```

## Branch / base

- branch: `fix-fe-i18n-links`
- base: `master-relay`
