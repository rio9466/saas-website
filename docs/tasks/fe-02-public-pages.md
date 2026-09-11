# Task: FE-02 官网公开页面（首页/功能/价格/关于/文档/联系）

## Goal

在 FE-01 基座上实现全部面向访客的页面，内容全部来自后端公开内容接口，
支持中英双语、SEO 与联系表单提交。

## Dependencies

- FE-01、BE-01、BE-02（契约 §4）。
- 契约：`docs/api/frontend-api-contract.md` §4、§7。

## Scope

### 1. 首页 `app/pages/index.vue`

- 读取 `GET /api/v1/public/home?locale=`，按 `sections[].type` 渲染已知区块：
  `hero`、`features`（引用 `/public/features`）、`screenshot`、`stats`、`cta`。
- **未知 type 直接忽略**，不得报错（契约 §4.3）。
- 无内容时使用兜底 Hero，不白屏。

### 2. 功能页 `app/pages/features.vue`

- 读取 `/api/v1/public/features?locale=`，渲染图标 + 标题 + 摘要 + 配图；
  有 `body_md` 时渲染 Markdown 区块。

### 3. 价格页 `app/pages/pricing.vue`

- 读取 `/api/v1/public/pricing?locale=`；月付/年付切换为前端行为（同一份数据取不同价格字段）。
- 方案卡：名称、描述、价格（按 locale 格式化 + `currency`）、功能清单、推荐标记、CTA。
- CTA 跳转 `cta_url`；不做支付（PRD 非目标）。

### 4. 通用页面 `app/pages/[slug].vue`（关于/隐私/条款）

- 读取 `/api/v1/public/pages?locale=` 生成路由或按需请求 `/public/pages/{slug}`。
- 404 的 slug 返回 Nuxt 404。
- 建议显式提供 `/about`、`/privacy`、`/terms` 的入口（来自导航或页脚），内容由后台页面承载。

### 5. 文档中心

- `app/pages/docs/index.vue`：读取 `/api/v1/public/docs?locale=`，渲染分类 + 文章侧边栏，默认展示首篇。
- `app/pages/docs/[slug].vue`：读取 `/api/v1/public/docs/{slug}?locale=`，
  渲染标题、Markdown 正文、文章内 TOC、上一篇/下一篇、所属分类面包屑；未发布/不存在 → 404。
- 侧边栏在当前文章所在分类高亮。

### 6. 联系页 `app/pages/contact.vue`

- 展示 `/public/settings` 中的邮箱/电话/地址/社交链接。
- 表单字段：`name`、`email`、`company`（可选）、`message`、隐私同意 `consent`，
  加隐藏蜜罐 `website`（CSS 隐藏，非 `display:none` 以免被识别为垃圾）。
- 提交 `POST /api/v1/public/contact`；成功显示确认信息；`10001` 做字段级提示；`42901` 提示稍后再试。
- 提交中禁用按钮防重复提交。

### 7. Markdown 渲染

- 新增 `app/components/MarkdownContent.vue`：解析 `body_md` 并用白名单净化后渲染；
  不直接 `v-html` 未净化内容；链接 `target`/`rel` 安全处理。

### 8. SEO 与站点文件

- 每页通过 `useSeo()` 使用内容中的 `seo_title`/`seo_description`，缺省回退站点默认值。
- 生成 `sitemap.xml`：固定路由 + `/public/pages` + `/public/docs`，按语言输出 `hreflang`。
  可用 `@nuxtjs/sitemap` 或 Nitro server route 实现（二选一，说明选择理由）。
- 首页与内容页配置 `routeRules` 做预渲染/ISR，沿用后端 `Cache-Control` 语义（契约 §7.4）。

### 9. 语言包

- 新增/补齐以上页面所需文案（en/zh-CN），保持 key 集合一致。

## Out of scope

- 登录/注册/控制台 → FE-03。
- 文档全文搜索、博客、支付。
- 后端任何改动；接口不符时报告 blocker。

## Files / areas

- `frontend/app/pages/**`（index/features/pricing/[slug]/docs/contact）
- `frontend/app/components/**`（MarkdownContent、页面级组件）
- `frontend/app/composables/**`（内容请求 composable，如 `useContent.ts`）
- `frontend/i18n/locales/*.json`
- `frontend/nuxt.config.ts`、`frontend/public/`（sitemap/robots 相关）
- `frontend/package.json`（新增 markdown 依赖）

## Acceptance criteria

- [ ] 中英文下首页/功能/价格/关于/文档/联系均可访问，内容来自后端接口。
- [ ] 后端未发布的内容不出现；不存在的页面/文档返回 404 页。
- [ ] 价格页月/年切换正确显示对应价格，金额按 locale 格式化。
- [ ] 文档侧边栏、TOC、上一篇/下一篇可用；分类高亮正确。
- [ ] 联系表单成功/校验失败/限流三种结果都有正确的双语文案。
- [ ] 联系表单蜜罐字段存在且隐藏，提交后 `website` 为空。
- [ ] 每页 title/description/OG 来自内容或站点默认值；`hreflang` 正确。
- [ ] `sitemap.xml` 可访问且包含全部已发布页面与文档。
- [ ] `pnpm lint && pnpm typecheck && pnpm build` 全绿。

## How to verify

```bash
cd frontend
pnpm lint && pnpm typecheck && pnpm build
pnpm dev
# 手工核对（后端已启动）：
#   /            /zh
#   /features    /pricing    /about    /docs    /docs/<slug>    /contact
#   /sitemap.xml
```

预期：两种语言页面内容与 SEO 正确；无效 slug 返回 404；联系表单提交成功入库。

## Branch / base

- branch: `fe-public-pages`
- base: `master-relay`
