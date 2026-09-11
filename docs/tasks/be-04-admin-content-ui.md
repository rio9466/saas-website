# Task: BE-04 管理后台内容管理与联系表单界面

## Goal

在 easy-admin 管理后台（`backend/server/admin/`）新增「内容管理」与「联系表单」界面，
使运营可以维护站点品牌、导航、首页、功能、价格、页面、文档、媒体，并处理联系表单留言。

## Dependencies

- BE-01（内容 CRUD 接口 + OpenAPI）、BE-02（收件箱与媒体接口 + OpenAPI）。
- 契约：`docs/api/frontend-api-contract.md` §6。
- 规则：`backend/server/admin/AGENTS.md`（必须完整遵守）。

## Scope

### 1. 生成类型

- 先运行 `pnpm generate:api`，从 `../docs/openapi.yaml` 重新生成 `types/api.generated.ts`。
- **禁止手工编辑** `types/api.generated.ts`；在 `src/api/contract.ts` 中补充手写类型别名。

### 2. API 模块（`src/api/`）

新增 `siteSettings.ts`、`navigation.ts`、`homeSections.ts`、`features.ts`、
`pricingPlans.ts`、`pages.ts`、`docs.ts`、`media.ts`、`contactSubmissions.ts`，
沿用现有 `src/api/users.ts` 的写法（`http.request<Envelope<T>>` + 类型化的 params/data）。
所有写操作的数据形状遵循契约 §6.1 的 `translations` 映射。

### 3. 视图（`src/views/`）

新增 `src/views/content/` 下：

- `site/`：站点设置表单（品牌/Logo/联系方式/社交链接/默认 SEO/默认语言）——含 `version` 乐观锁，
  提交 `40013` 时提示“内容已被他人修改，请刷新后重试”。
- `navigation/`：header/footer 导航列表，增删改、排序、显隐、父子层级。
- `home/`：首页区块列表（按 type 编辑 payload），排序、显隐。
- `features/`：功能条目 CRUD + 排序 + 发布态。
- `pricing/`：价格方案 CRUD（月/年价、币种、推荐、功能清单逐条编辑）。
- `pages/`：页面 CRUD（slug、发布态、Markdown 正文）。
- `docs/`：文档分类与文章（分类树 + 文章编辑器，slug/排序/发布态）。
- `media/`：媒体库（上传、列表、复制 URL、删除）。

新增 `src/views/contact/index.vue`：联系表单收件箱（状态筛选、分页、详情抽屉、状态流转）。

### 4. 多语言编辑

- 各内容表单按 `supported_locales`（来自公开设置或管理端站点设置）渲染语言 Tab。
- 缺失语言显式标记「未翻译」；保存时只提交填写的语言。
- 复用 Element Plus 组件，不引入第二套 UI 框架。

### 5. 路由与菜单（本地）

新增 `src/router/modules/content.ts`（「内容管理」父级）与收件箱条目，路由 meta 使用
`permissions: ["admin.content.read"]` / `["admin.contact.read"]`，`roles` 参考现有模块。
菜单仍为本地定义，禁止从后端拉取菜单树。

## Out of scope

- 任何 `server/` Go 代码或 `server/docs/openapi.yaml` 的改动（若接口缺失或形状不符，报告 blocker，
  由 BE-01/BE-02 分支修正）。
- 前端官网（`frontend/`）。
- 草稿预览、定时发布、版本历史（PRD P1/P2）。

## Files / areas

- `backend/server/admin/src/api/**`（新增模块 + `contract.ts` 追加类型）
- `backend/server/admin/src/views/content/**`、`src/views/contact/**`（新增）
- `backend/server/admin/src/router/modules/content.ts`（新增，并在路由聚合处注册）、`system.ts` 按需
- `backend/server/admin/types/api.generated.ts`（仅由 `pnpm generate:api` 生成）

## Acceptance criteria

- [ ] `pnpm generate:api` 后类型可编译，无手工修改生成文件。
- [ ] 每个内容资源可完成 列表 → 新建（多语言）→ 编辑 → 删除，且与后端实际响应一致。
- [ ] 站点设置并发冲突提示正确（`40013`）。
- [ ] 收件箱可筛选/分页/查看详情并流转状态。
- [ ] 媒体可上传并在内容表单中引用。
- [ ] 菜单按 `admin.content.read` / `admin.contact.read` 权限显隐；无权限直接访问显示 403。
- [ ] 未登录/refresh 恢复/登出等既有认证行为不被破坏。
- [ ] 浏览器存储中不出现任何 token；未修改 `types/api.generated.ts` 与 `server/`。

## How to verify

```bash
cd backend/server/admin
pnpm install --frozen-lockfile
pnpm generate:api
pnpm lint
pnpm typecheck
pnpm test
pnpm build
```

浏览器手工核对：用有权限管理员登录 → 内容管理各页 CRUD 成功；用无权限账号登录 → 菜单隐藏、直接访问 403。

## Branch / base

- branch: `be-admin-content-ui`
- base: `master-relay`
