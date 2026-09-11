# Task: ADMIN-UX-01 管理后台内容体验优化（多语言弹窗布局 / Markdown 编辑器 / 站点设置合并）

## Goal

修复并优化管理后台内容管理体验。只改 `backend/server/admin/**`。

## 问题 1：弹窗里的多语言配置 UI 布局错乱

**现象**：凡是弹窗包含「多语言内容」区块（页面、价格方案、首页区块等多语言编辑），
字段被挤成窄列、与上半部分的全宽表单不对齐（见用户截图：标题/正文/SEO 等在窄列里，标签列过宽）。

**定位**：
- `src/views/content/components/LocaleTranslationTabs.vue` 内部渲染了 `el-form-item`；
- 各页面弹窗又把它包在 `<el-form-item label="多语言内容">` 里，等于**嵌套 form-item**，
  再叠加外层 `el-form` 的 `label-width`，导致内层标签/输入宽度错乱。
- 另见 `src/views/content/{pages,pricing,home,features,docs,...}/index.vue` 的弹窗表单结构。

**要求**：
- 让「多语言内容」区块**占满弹窗宽度**，内层字段标签对齐一致、输入框自适应填满剩余宽度。
- 修好所有使用 `LocaleTranslationTabs` 的弹窗，不只某一个。
- 不改多语言数据的读写语义。

## 问题 2：引入上游的 Markdown 编辑器（Vditor）

**背景**：我们的后台基于 `pure-admin-thin` 6.2.0（无 Markdown 编辑器）；上游主项目
`pure-admin/vue-pure-admin`（UI 参照 commit `c0fd7419c58689c7b1055ef92575a9beea7314d3`）
使用 **Vditor**（依赖 `vditor ^3.11.2`），封装在
`src/views/markdown/components/Vditor.vue`。

**要求**：
- 把该 Vditor 封装组件移植进本后台（如 `src/components/MarkdownEditor/` 或 `src/views/content/components/`），
  保留 `v-model`、主题（亮/暗）跟随、`onUnmounted` 销毁等行为；`cache.enable=false`。
- 新增依赖 `vditor`（按上游版本 `^3.11.2`，用 pnpm）。
- 将所有 `body_md` 类字段（页面正文、文档文章正文、功能 `body_md` 等）从纯 `textarea` 换成该编辑器。
- 不改后端；Markdown 仍由后端净化后返回，编辑器只负责编辑。

## 问题 3：站点设置合并到系统配置

**背景**：现在内容菜单下有独立「站点设置」页（`src/views/content/site/index.vue`，
走内容站点设置 API）；系统里另有「系统配置」页（`src/views/system/settings/index.vue`）。

**要求**：
- 把「站点设置」的内容并入**系统配置**页（作为其中一个 section/Tab，例如「站点信息」），
  继续调用现有内容站点设置 API（`src/api/siteSettings.ts`），保留 `version` 乐观锁与 `40013` 冲突提示。
- 从内容菜单/路由中移除独立的「站点设置」入口（`src/router/modules/content.ts`、权限 meta 保持合理）。
- 不改后端接口。

## Out of scope

- 后端 `server/` Go 代码、`server/docs/openapi.yaml`、契约、frontend/、根文档。
- 其它非内容管理页面。

## Files / areas

- `backend/server/admin/src/views/content/**`（弹窗布局、Markdown 字段、移除 site 页）
- `backend/server/admin/src/views/system/settings/index.vue`（并入站点设置 section）
- `backend/server/admin/src/components/**` 或 `src/views/content/components/**`（Vditor 封装）
- `backend/server/admin/src/router/modules/content.ts`（菜单调整）
- `backend/server/admin/package.json`、`pnpm-lock.yaml`（新增 `vditor`）
- `backend/server/admin/types/api.generated.ts`（仅在需要时由 `pnpm generate:api` 生成）

## 运行环境说明

- 管理后台 dev 目前可由一个**本地不提交**的 `vite.config.local.ts`（端口 8849、代理到后端 8100）启动；
  请沿用该方式，不要提交运行态文件；不要占用 3100/8100/8849 上正在运行的其它服务，自测可用别的端口。
- 后端已在运行：http://127.0.0.1:8100。

## Acceptance criteria

- [ ] 所有含多语言编辑的弹窗布局整齐、全宽对齐，无错乱。
- [ ] 页面/文档/功能正文使用 Vditor Markdown 编辑器，可正常编辑并保存；亮/暗主题跟随。
- [ ] 「站点设置」已合并进「系统配置」，内容菜单不再有独立站点设置入口；保存与 `40013` 冲突提示正常。
- [ ] 未修改 `server/` Go 代码与 `server/docs/openapi.yaml`；未手工编辑 `types/api.generated.ts`。
- [ ] `pnpm install --frozen-lockfile && pnpm lint && pnpm typecheck && pnpm test && pnpm build` 全绿。

## How to verify

```bash
cd backend/server/admin
pnpm install
pnpm lint && pnpm typecheck && pnpm test && pnpm build
# 起 dev（本地配置指向 8100），手工核对：
#   页面/价格方案/首页区块弹窗的多语言布局；正文用 Vditor 编辑并保存；
#   系统配置页包含站点信息并保存成功；内容菜单无独立站点设置
```

## Branch / base

- branch: `admin-content-ux`
- base: `master-relay`
