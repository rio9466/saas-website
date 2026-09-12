# Task: NEXT-02 公开页面（首页/功能/价格/关于/文档/联系 + sitemap + 访问上报）

## Goal

在 `next/` 基座上实现全部面向访客的公开页面，内容来自后端公开内容接口，双语、SEO、联系表单，
并接入访问上报（契约 §4.10）。与 `frontend/`（FE-02/FE-ANALYTICS）功能对齐，UI 用 shadcn/ui。

## 依赖

- NEXT-01（基座）、BE-01/BE-02（内容与联系接口，已合入）、契约 §4。

## Scope

1. **首页** `app/[locale]/page.tsx`：`GET /api/v1/public/home?locale=`，按 `sections[].type` 渲染
   `hero`/`features`/`screenshot`/`stats`/`cta`；未知 type 忽略不报错；无内容用兜底。
2. **功能页** `/features`、**价格页** `/pricing`（月/年切换，金额按 locale 格式化，`cta_url` 跳转）、
   **关于等** `/about`（复用 `/public/pages` 或 `[slug]`）。
3. **文档中心** `/docs`、`/docs/[slug]`：分类侧栏、正文 Markdown、TOC、上一篇/下一篇、分类高亮；
   未发布/不存在 → 404。
4. **联系页** `/contact`：展示站点联系方式与社交链接；表单 `name/email/company/message/consent` + 隐藏蜜罐 `website`；
   `POST /api/v1/public/contact`（成功/`10001` 字段级提示/`42901` 稍后重试）；提交中禁用防重复。
5. **Markdown 渲染**：白名单净化（与 `frontend` 的 `useMarkdown` 等价），禁止直接注入未净化 HTML。
6. **SEO**：每页 `generateMetadata`（`seo_title`/`seo_description` 回退站点默认值）、OG、hreflang、canonical；
   `app/sitemap.ts`（固定路由 + `/public/pages` + `/public/docs`，按语言输出）+ `robots.ts`。
7. **访问上报**（契约 §4.10）：客户端在路由变化时上报一次 `POST /api/v1/public/page-view`
   （`path`、`referrer`、`locale`），fire-and-forget（`navigator.sendBeacon` 优先，失败静默）；
   跳过 `/account` 等私有区。

## Out of scope

- 登录/注册/账户（NEXT-03）；后端改动；`frontend/`；根文档。

## Files / areas

- `next/app/[locale]/**`（页面）、`next/components/**`、`next/lib/**`、`next/messages/**`（i18n 文案）、
  `next/app/sitemap.ts`、`next/app/robots.ts`

## Acceptance criteria

- [ ] 中英双语下首页/功能/价格/关于/文档/联系均可访问，内容来自后端；未发布不出现；无效 slug → 404。
- [ ] 价格月/年切换、金额格式化、文档侧栏/TOC/上一篇下一篇/分类高亮正确。
- [ ] 联系表单成功/校验失败/限流三种结果双语文案正确；蜜罐隐藏且提交为空。
- [ ] 每页 title/description/OG/hreflang 正确；`sitemap.xml` 可访问且含全部已发布页面与文档。
- [ ] 页面间切换会向后端上报 PV（Network 可见）；失败无可见报错。
- [ ] `pnpm lint && pnpm build` 全绿；无裸 `/x` 站内链接。

## How to verify

```bash
cd next
pnpm lint && pnpm build
# 起后端(8100)+dev(3200)，核对两种语言的各页面、sitemap、联系表单、访问上报
```

## Branch / base

- branch: `next-pages`
- base: `master-relay`
