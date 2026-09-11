# Task: BE-01 内容基础模型 + 公开内容接口 + 管理端内容 CRUD

## Goal

为官网提供「可后台运营」的内容基座：新增站点内容数据表、多语言翻译、公开只读内容接口，
以及管理员内容 CRUD 接口与 RBAC 权限。前端据此渲染首页/功能/价格/关于/文档；管理后台据此编辑内容。

## Dependencies

- 无。契约：`docs/api/frontend-api-contract.md` §4、§6。

## Scope

### 1. 数据模型（`backend/server/migrations/primary/`）

新增迁移 `000008_content_platform.up.sql` / `.down.sql`：

- `site_settings` 单例（`id = 1`，`version` 乐观锁）：`site_name`、`logo_url`、`logo_dark_url`、
  `favicon_url`、`contact_email`、`contact_phone`、`contact_address`、`social_links` (JSONB)、
  `seo_default_og_image_url`、`default_locale`、`updated_by`、`updated_at`。
- `site_setting_translations`（`locale`，`tagline`、`footer_text`、`seo_default_title`、
  `seo_default_description`、`icp_record`）。
- `supported_locales`（`code`、`label`、`enabled`、`sort_order`、`is_default`）——至少 `en`、`zh-CN`，
  保证「有且仅有一个默认语言且默认语言必须启用」。
- `navigation_items`（`placement` header/footer、`parent_id`、`url`、`target`、`sort_order`、
  `visible`）+ `navigation_item_translations`（`label`）。
- `home_sections`（`type`、`sort_order`、`published`、`payload` JSONB）+ `home_section_translations`。
- `features`（`icon`、`sort_order`、`published`）+ `feature_translations`（`title`、`summary`、
  `body_md`、`image_url`）。
- `pricing_plans`（`code` unique、`monthly_price`、`yearly_price`、`currency`、`highlighted`、
  `sort_order`、`visible`）+ `pricing_plan_translations`（`name`、`description`、`cta_label`、
  `cta_url`）+ `pricing_plan_features`（`sort_order`、`text`、`locale`）。
- `pages`（`slug` unique、`published`、`sort_order`）+ `page_translations`（`title`、`body_md`、
  `seo_title`、`seo_description`）。
- `doc_categories`（`slug` unique、`sort_order`）+ `doc_category_translations`（`name`）。
- `doc_articles`（`slug` unique、`category_id`、`sort_order`、`published`）+
  `doc_article_translations`（`title`、`body_md`、`seo_title`、`seo_description`）。

约定：金额/积分类 `NUMERIC`；翻译唯一键 `(entity_id, locale)`；所有 `slug` 全局唯一；
外键 `ON DELETE CASCADE` 仅限纯翻译表。

新增迁移 `000009_content_permissions.up.sql` / `.down.sql`：

- 插入权限 `admin.content.read`、`admin.content.manage`、`admin.contact.read`、`admin.contact.manage`；
  授予 `super_admin`（沿用 000007 的写法）。

### 2. 代码分层（沿用现有结构）

- `internal/domain/content/`：实体、校验、locale 回退逻辑。
- `internal/repository/primary/`：SQL 仓储。
- `internal/service/contentsvc/`：公开读、管理端写、审计（pending-first）、版本冲突、发布态过滤。
- `internal/transport/http/handler/` + `internal/transport/http/router.go`：路由注册。

### 3. 公开接口（契约 §4）

- 扩展 `GET /api/v1/public/settings`：追加契约 §4.1 列出的 `site_name`/`logo_url`/.../`locales`
  字段；**原字段与语义不变**。
- 新增 `GET /api/v1/public/navigation`、`/home`、`/features`、`/pricing`、`/pages`、
  `/pages/{slug}`、`/docs`、`/docs/{slug}`。
- 全部支持 `?locale=`，未知/禁用语言回退默认语言，响应携带生效 `locale`。
- 只返回 `published`/`visible` 内容，字段白名单，不透出内部字段（`id` 可透出）。
- `Cache-Control: public, max-age=60, stale-while-revalidate=300`。
- `body_md` 必须服务端净化（Markdown → 安全 HTML 或存储前过滤危险 HTML）；`url` 字段校验协议白名单。

### 4. 管理端接口（契约 §6）

按契约 §6.2 的资源清单实现 CRUD：`site-settings`、`navigation-items`、`home-sections`、
`features`、`pricing-plans`、`pages`、`doc-categories`、`doc-articles`。
统一 `translations` 映射读写；列表分页 `{items,total,page,page_size}`；写操作校验
`admin.content.manage` 并写 pending-first 审计。`site-settings` 用 `version` 乐观锁返回 `40013`。

### 5. OpenAPI

在 `backend/server/docs/openapi.yaml` 补齐以上所有路径、schema、错误响应，保持与实现一致。

## Out of scope

- 联系表单（BE-02）。
- 媒体上传（BE-02，与联系表单一起做；本任务只存 URL）。
- 管理后台界面（BE-04）。
- 草稿预览 token、定时发布、版本历史（PRD P1/P2）。
- 文档全文搜索。

## Files / areas

- `backend/server/migrations/primary/000008*`、`000009*`（新增）
- `backend/server/internal/domain/content/**`、`internal/repository/primary/**`、
  `internal/service/contentsvc/**`、`internal/transport/http/handler/**`、
  `internal/transport/http/router.go`
- `backend/server/docs/openapi.yaml`
- `backend/server/internal/**/*_test.go`（新增测试）

## Acceptance criteria

- [ ] 迁移可重复执行（`schema_migrations` 机制），up/down 成对，`make run` 在全新库成功。
- [ ] 契约 §4 全部路径可用，且返回形状逐字段符合契约。
- [ ] `/api/v1/public/settings` 原字段不变，新增字段齐备。
- [ ] locale 回退：请求 `?locale=xx-XX` 时不报错，`data.locale` 为默认语言。
- [ ] 未发布页面/文档返回 404 / `40002`；未发布导航/功能/价格不出现在公开接口。
- [ ] 管理端每个资源：创建 → 读取（全语言）→ 更新 → 删除 全链路可用。
- [ ] 无 `admin.content.*` 权限的管理员写操作返回 403 / `30001`。
- [ ] `site-settings` 并发写入返回 `40013`。
- [ ] 每个管理写操作产生审计记录。
- [ ] `docs/openapi.yaml` 与路由/响应一致（可用现有 openapi 契约测试校验）。

## How to verify

```bash
cd backend/server
make fmt && make vet && make test && make build
# 启动后（本地库）：
go run ./cmd/server -config configs/config.local.toml
curl -sS 'http://127.0.0.1:8080/api/v1/public/settings?locale=zh-CN' | head
curl -sS 'http://127.0.0.1:8080/api/v1/public/pricing?locale=zh-CN' | head
curl -sS -o /dev/null -w '%{http_code}\n' 'http://127.0.0.1:8080/api/v1/public/pages/nope'
```

预期：`make check` 全绿；接口返回 `code:0` 信封；不存在页面返回 404。

## Branch / base

- branch: `be-content-foundation`
- base: `master-relay`
