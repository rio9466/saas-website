# 前端 ↔ 后端 API 契约（官网 + 用户中心）

> Owner: `master-relay`（主线，跨切面）。`frontend/` 与 `backend/` 分支不得编辑本文件；
> 需要变更契约时，先在 `master-relay` 更新本文件，再拆任务。
> 后端权威契约：`backend/server/docs/openapi.yaml` 必须与本文件保持一致。
> Status: **v1 基线** — 由 `docs/prd/saas-website-prd.md` 拆解而来。

本文件是前端接入后端的唯一约定。所有新增后端接口必须遵守 §1–§3；前端必须按 §7 规定实现。

---

## 1. 通用约定

### 1.1 响应信封（所有接口，无例外）

后端由 `backend/server/internal/transport/http/response` 统一封装：

```json
{
  "code": 0,
  "message": "success",
  "data": {},
  "request_id": "c0ffee..."
}
```

- `code == 0` 表示成功，`message == "success"`。
- 失败时 `data` 为 `{}`，`message` 是**面向开发者的英文短句，不是本地化文案**；前端按 `code` 映射到 i18n 文案，**禁止把 `message` 直接显示给用户**。
- HTTP 状态码承载语义（200/201/400/401/403/404/409/429/500/503），`code` 是稳定的业务码。
- `request_id` 必须原样保留用于排障；前端错误上报携带该值。
- 响应头包含 `X-Request-ID`（与 body 中一致）。

### 1.2 数据类型约定

| 类型 | 约定 | 示例 |
| ---- | ---- | ---- |
| 持久化 ID | 十进制字符串（避免 JS 精度丢失） | `"1024"` |
| 时间 | UTC RFC3339 字符串；无值用 `""`（null 时间字段序列化为空串） | `"2026-09-11T08:00:00Z"` |
| 积分 | 定点四位小数字符串，**不参与浮点运算** | `"123.4567"` |
| 金额 | 十进制字符串（两位小数），附 `currency` | `"29.00"` |
| 布尔 | JSON boolean | `true` |
| 分页 | `data = { items, total, page, page_size }` | 默认 `page=1`、`page_size=20`、上限 100 |
| 可选字段 | 对象存在但值可为 `""` / `null`；列表不存在时为 `[]` | — |

### 1.3 请求通用头

| Header | 说明 |
| ------ | ---- |
| `Content-Type: application/json` | 所有写请求 |
| `Authorization: Bearer <access_token>` | 需登录接口 |
| `X-Request-ID` | 可选，前端生成时须满足 `^[A-Za-z0-9._:-]{1,128}$` |
| `Accept-Language` | 前端可传，但后端本地化内容以 `?locale=` 为准 |

---

## 2. 认证与会话

- **Access token**：`Authorization: Bearer`，有效期 15 分钟，audience `easy-admin-user`；payload 只含 subject / session id / token type。前端**只存内存**，禁止写入 `localStorage`/`sessionStorage`。
- **Refresh token**：`HttpOnly` Cookie `ea_user_refresh`，`Path=/api/v1/auth`，`SameSite=Strict`，生产环境 `Secure`。前端 JS 不可读。
- **登录/刷新**：`POST /api/v1/auth/login`、`/api/v1/auth/refresh` 返回新 access token，并轮转 refresh cookie。
- **401 处理**：收到 HTTP 401（`code` 20001/20002/20003/40006）时，前端**只重试一次**：调用 refresh → 成功后重放原请求；失败则清空登录态并跳登录页。
- **会话绝对上限**：30 天，刷新不会延长。
- **登录/刷新/登出** 会校验请求 `Origin`/`Referer` 是否在 `auth.trusted_origins` 中。见 §2.1。

### 2.1 部署与代理（强制）

后端**没有 CORS 中间件**，且 refresh cookie 为 `SameSite=Strict`。因此：

> **浏览器必须只访问前端同源地址下的 `/api/**`，由 Nuxt/Nitro 代理转发到 Go 服务；禁止浏览器直接跨域调用后端。**

- 开发：`frontend/nuxt.config.ts` 配置代理 `/api/**` → `runtimeConfig.apiProxyTarget`（默认 `http://127.0.0.1:8080`）。
- 生产：Nitro 代理或边缘/nginx 均可，只要 `/api/**` 与页面同源。
- 代理必须转发 `Origin`/`Referer`（前端 origin）与 `Set-Cookie`/`Cookie`。
- 后端配置 `auth.trusted_origins` 必须包含前端站点 origin，例如 `https://www.example.com`。
- SSR 请求可直接访问后端内网地址（`runtimeConfig.apiInternalBase`），此时无跨域问题；cookie 转发仅发生在浏览器侧。

---

## 3. 错误码表

以下为当前后端已注册的业务码（`backend/server/internal/domain/apperr/apperr.go`）。新增接口只能复用或在此表登记新码。

| code | HTTP | 含义 | 前端处理建议 |
| ---- | ---- | ---- | ------------ |
| 0 | 200/201 | 成功 | — |
| 10001 | 400 | 参数校验失败 | 表单字段级错误提示 |
| 20001 | 401 | 未认证 | 触发刷新；失败跳登录 |
| 20002 | 401 | 会话无效 | 清登录态，跳登录 |
| 20003 | 401 | refresh 重放 | 清登录态，跳登录 |
| 30001 | 403 | 无权限 | 显示 403 页/提示 |
| 40001 | 409 | 冲突 | 通用冲突提示 |
| 40002 | 404 | 不存在 | 显示 404 页 |
| 40005 | 400 | 当前密码错误 | 定位到当前密码字段 |
| 40006 | 401 | 账号被禁用 | 提示已禁用 |
| 40010 | 403 | 注册已关闭 | 隐藏/禁用注册入口 |
| 40011 | 403 | 邮箱未验证 | 引导去验证页 |
| 40012 | 400 | 登录方式被禁用 | 提示并刷新公开设置 |
| 40013 | 409 | 设置版本冲突 | 重新拉取后重试 |
| 40016 | 400 | 验证令牌无效/过期/已用 | 引导重新发送 |
| 42901 | 429 | 触发限流 | 文案提示稍后再试 |
| 50001 | 503 | 依赖不可用 | 通用服务不可用提示 |
| 50002 | 503 | 审计不可用 | 通用服务不可用提示 |
| 50003 | 503 | 邮件服务不可用 | 提示邮件发送失败 |

前端映射表必须覆盖以上全部码，并提供兜底文案。

---

## 4. 公开内容接口（无需登录）

所有内容接口接受 `?locale=<code>`；缺省用默认语言；未知/已禁用语言回退默认语言，且响应体中 `locale` 字段返回实际生效语言。响应均带：

```
Cache-Control: public, max-age=60, stale-while-revalidate=300
```

### 4.1 `GET /api/v1/public/settings`（扩展现有接口）

在原字段基础上**新增**以下字段（原字段保持不变，向后兼容）：

```json
{
  "code": 0, "message": "success",
  "data": {
    "platform_name": "Acme Cloud",
    "public_frontend_url": "https://www.example.com",
    "public_api_url": "https://www.example.com/api",
    "registration_enabled": true,
    "username_login_enabled": true,
    "email_login_enabled": true,
    "email_verification_required": true,
    "default_avatar_url": "",

    "site_name": "Acme Cloud",
    "logo_url": "/media/logo.svg",
    "logo_dark_url": "/media/logo-dark.svg",
    "favicon_url": "/media/favicon.ico",
    "tagline": "多语言软件官网",
    "footer_text": "© 2026 Acme Cloud",
    "icp_record": "",
    "contact_email": "hello@example.com",
    "contact_phone": "+86 000 0000 0000",
    "contact_address": "示例地址",
    "social_links": [
      { "platform": "github", "url": "https://github.com/example" }
    ],
    "seo_default_title": "Acme Cloud",
    "seo_default_description": "多语言软件官网",
    "seo_default_og_image_url": "/media/og.png",
    "default_locale": "en",
    "locales": [
      { "code": "en", "label": "English" },
      { "code": "zh-CN", "label": "简体中文" }
    ]
  },
  "request_id": "..."
}
```

- `locales` 只包含已启用语言；`default_locale` 必须在其中。
- `tagline`/`footer_text`/`seo_default_*` 已按 `?locale` 解析，前端直接渲染。

### 4.2 `GET /api/v1/public/navigation?locale=&placement=header|footer`

```json
{ "code": 0, "message": "success", "request_id": "...",
  "data": { "locale": "zh-CN",
    "items": [
      { "id": "1", "label": "功能", "url": "/features", "target": "_self", "sort_order": 1,
        "children": [ { "id": "2", "label": "文档", "url": "/docs", "target": "_self", "sort_order": 1, "children": [] } ] }
    ] } }
```

`placement` 缺省 `header`。只返回 `visible=true` 条目。

### 4.3 `GET /api/v1/public/home?locale=`

```json
{ "code": 0, "message": "success", "request_id": "...",
  "data": { "locale": "en", "sections": [
    { "id": "1", "type": "hero", "sort_order": 1,
      "data": { "title": "...", "subtitle": "...", "primary_cta_label": "Get started", "primary_cta_url": "/register",
                "secondary_cta_label": "Contact us", "secondary_cta_url": "/contact", "image_url": "/media/hero.png" } },
    { "id": "2", "type": "features", "sort_order": 2, "data": { "title": "Features" } },
    { "id": "3", "type": "cta", "sort_order": 3,
      "data": { "title": "...", "body": "...", "cta_label": "Sign up", "cta_url": "/register" } }
  ] } }
```

已知 `type`：`hero`、`features`、`screenshot`、`stats`、`cta`。前端**必须忽略未知 type**（向前兼容）。`features`/`screenshot` 区块从 §4.4 数据取内容。

### 4.4 `GET /api/v1/public/features?locale=`

```json
{ "code": 0, "message": "success", "request_id": "...",
  "data": { "locale": "en", "items": [
    { "id": "1", "icon": "i-lucide-zap", "title": "Fast", "summary": "...", "body_md": "", "image_url": "", "sort_order": 1 }
  ] } }
```

### 4.5 `GET /api/v1/public/pricing?locale=`

```json
{ "code": 0, "message": "success", "request_id": "...",
  "data": { "locale": "en", "currency": "USD", "plans": [
    { "id": "1", "code": "pro", "name": "Pro", "description": "...",
      "monthly_price": "29.00", "yearly_price": "290.00", "currency": "USD",
      "highlighted": true, "cta_label": "Get started", "cta_url": "/register",
      "features": ["10 seats", "Priority support"], "sort_order": 1 }
  ] } }
```

### 4.6 页面

- `GET /api/v1/public/pages?locale=` → `{ "locale", "items": [ { "slug", "title", "updated_at" } ] }`
- `GET /api/v1/public/pages/{slug}?locale=` → `{ "locale", "slug", "title", "body_md", "seo_title", "seo_description", "updated_at" }`；未发布/不存在 → 404 / 40002。

### 4.7 文档

- `GET /api/v1/public/docs?locale=` → `{ "locale", "categories": [ { "id", "slug", "name", "sort_order", "articles": [ { "id", "slug", "title", "sort_order", "updated_at" } ] } ] }`
- `GET /api/v1/public/docs/{slug}?locale=` → `{ "locale", "id", "slug", "title", "body_md", "category": { "id", "slug", "name" }, "prev": null, "next": { "slug", "title" }, "seo_title", "seo_description", "updated_at" }`

### 4.8 `POST /api/v1/public/contact`

请求：

```json
{ "name": "张三", "email": "a@b.com", "company": "ACME", "message": "...", "locale": "zh-CN",
  "consent": true, "website": "" }
```

- `website` 为蜜罐字段，前端渲染但隐藏；非空时后端静默返回成功且不入库。
- `name`、`email`、`message`、`consent=true` 必填。

响应 `201`：

```json
{ "code": 0, "message": "success", "data": { "id": "12", "submitted_at": "2026-09-11T08:00:00Z" }, "request_id": "..." }
```

错误：`10001`（校验）、`42901`（限流）。响应 `Cache-Control: no-store`。

### 4.9 sitemap 数据

前端 sitemap 由 §4.6 页面列表、§4.7 文档列表 + 固定路由生成，不额外提供接口。

### 4.10 `POST /api/v1/public/page-view`（访问上报）

官网每次页面浏览上报一次，供管理端工作台统计 PV 与访问来源。

请求：

```json
{ "path": "/features", "referrer": "https://www.baidu.com/s?wd=acme", "locale": "zh-CN" }
```

- `path` 必填，站内路径（以 `/` 开头，长度 ≤ 512）；不合法的上报静默丢弃。
- `referrer` 可空：为空或非 `http(s)` → 来源记为固定字符串 `direct`；否则取 **host**（不含协议/路径，保留子域，如 `www.baidu.com`）。**不做域名识别/归类**。
- `locale` 可选。
- 成功 `200`：`{ "code": 0, "message": "success", "data": {}, "request_id": "..." }`；响应 `Cache-Control: no-store`。
- 限流：每 IP 每小时上限（宽松，防刷）；Redis 故障时 fail-open（直接接受），因为仅统计用。
- 无需登录，不写审计。

---

## 5. 业务用户接口

### 5.1 已有接口（前端直接使用，形状不变）

| 方法 | 路径 | 认证 | 成功 | data |
| ---- | ---- | ---- | ---- | ---- |
| POST | `/api/v1/auth/register` | 否 | 201 | 用户资料（见 §5.2） |
| POST | `/api/v1/auth/verify-email` | 否 | 200 | `{ "email_verified": true }` |
| POST | `/api/v1/auth/resend-verification` | 否 | 200 | `{ "resent": true }` |
| POST | `/api/v1/auth/login` | 否 | 200 | `{ access_token, token_type:"Bearer", expires_in }` + Set-Cookie |
| POST | `/api/v1/auth/refresh` | Cookie | 200 | 同上 |
| POST | `/api/v1/auth/logout` | 是 | 200 | `{}` + 清除 Cookie |
| GET | `/api/v1/me` | 是 | 200 | 用户资料 |

注册：`{ "username", "email", "password" }`，用户名/邮箱唯一，密码 8–72 字节。
登录：`{ "identifier", "password" }`；`identifier` 含 `@` 时按邮箱登录（需后端开启邮箱登录），否则按用户名。

### 5.2 用户资料对象（`/me`、注册返回）

```json
{
  "id": "1", "username": "alice", "email": "a@b.com", "nickname": "Alice", "avatar_url": "",
  "status": "active", "email_verified_at": "2026-09-11T08:00:00Z", "last_login_at": "",
  "points_balance": "123.4567", "consumption_points": "20.0000",
  "level": { "id": "2", "code": "gold", "name": "Gold", "mode": "auto" },
  "created_at": "2026-09-01T08:00:00Z", "updated_at": "2026-09-11T08:00:00Z"
}
```

> 说明：`points_balance`、`consumption_points`、`level`、`created_at` 后端**已返回**，前端控制台直接使用；无需扩展 `/me`。

`status` 取值：`pending_verification` | `active` | `disabled`。
`level.mode` 取值：`auto` | `manual`。

### 5.3 新增：`PATCH /api/v1/me`

请求（均为可选，至少一个）：`{ "nickname": "Alice", "avatar_url": "/media/avatar.png" }`
响应 200：完整用户资料（§5.2）。
说明：**不接受** username/email/积分/等级/状态；邮箱改绑不在本期范围。

### 5.4 新增：`POST /api/v1/me/password`

请求：`{ "current_password": "...", "new_password": "..." }`（新密码 8–72 字节）

响应 200：`{}`，并 `Set-Cookie` 清除 refresh cookie。

语义：**成功即吊销该用户全部会话，要求重新登录**。客户端必须清空内存 access token 并跳转登录页。
错误：`40005` 当前密码错误；`10001` 校验失败。

### 5.5 新增：`GET /api/v1/me/point-transactions?page=&page_size=`

响应 200：`data = { items, total, page, page_size }`，`items` 元素：

```json
{ "id": "9", "points_delta": "10.0000", "consumption_delta": "0.0000",
  "balance_after": "123.4567", "consumption_after": "20.0000",
  "reason": "注册奖励", "actor_id": "3", "idempotency_key": "...", "created_at": "2026-09-11T08:00:00Z" }
```

仅返回本人流水，时间倒序。`actor_id` 可为 `null`。

### 5.6 新增（P1）：找回密码

- `POST /api/v1/auth/forgot-password`，请求 `{ "email" }`，响应 200 `{}`（无论邮箱是否存在都返回成功，不泄露账号存在性），限流 `42901`。
- `POST /api/v1/auth/reset-password`，请求 `{ "email", "token", "new_password" }`，响应 200 `{}`；令牌无效 `40016`；成功后吊销全部会话。

### 5.7 邮件链接约定（前端路由必须匹配）

后端邮件中的链接基于 `public_frontend_url` 生成，前端必须提供对应路由并读取 query：

| 用途 | 链接 | 前端路由 |
| ---- | ---- | -------- |
| 邮箱验证 | `{public_frontend_url}/verify-email?email=<urlencoded>&token=<token>` | `/verify-email` |
| 重置密码 | `{public_frontend_url}/reset-password?email=<urlencoded>&token=<token>` | `/reset-password` |

> 现状：`verificationLink` 目前只带 `token`，而 `POST /api/v1/auth/verify-email` 需要 `email`+`token`。
> BE-03 必须把 `email` 加入验证与重置链接的 query，使邮件链接自包含（可在任意设备打开）。

---

## 6. 管理端内容接口（`/api/v1/admin/**`，管理员 Bearer）

### 6.1 约定

- 权限码：`admin.content.read`、`admin.content.manage`、`admin.contact.read`、`admin.contact.manage`、`admin.analytics.read`（命名沿用 `admin.<resource>.<action>`）。
- 所有写操作走现有权限校验 + pending-first 审计。
- 列表分页同 §1.2；`GET` 详情返回**全语言**内容，供后台按语言 Tab 编辑。
- 多语言写入统一使用 `translations` 映射：

```json
{
  "slug": "about",
  "published": true,
  "sort_order": 10,
  "translations": {
    "en": { "title": "About", "body_md": "...", "seo_title": "", "seo_description": "" },
    "zh-CN": { "title": "关于我们", "body_md": "..." }
  }
}
```

读取时返回同一形状；缺失语言不返回该 key。发布态/排序等非翻译字段在顶层。

### 6.2 资源与路径

| 资源 | 路径 | 说明 |
| ---- | ---- | ---- |
| 站点设置 | `GET|PUT /api/v1/admin/site-settings` | 单例；`version` 乐观锁，冲突 `40013` |
| 导航 | `GET|POST /api/v1/admin/navigation-items`，`PATCH|DELETE /{id}` | `placement` header/footer |
| 首页区块 | `GET|POST /api/v1/admin/home-sections`，`PATCH|DELETE /{id}` | `type` + `data` payload |
| 功能 | `GET|POST /api/v1/admin/features`，`PATCH|DELETE /{id}` | |
| 价格方案 | `GET|POST /api/v1/admin/pricing-plans`，`PATCH|DELETE /{id}` | 月/年价、币种、推荐、功能清单 |
| 页面 | `GET|POST /api/v1/admin/pages`，`GET|PATCH|DELETE /{id}` | 顶层 `slug`/`published` |
| 文档分类 | `GET|POST /api/v1/admin/doc-categories`，`PATCH|DELETE /{id}` | |
| 文档文章 | `GET|POST /api/v1/admin/doc-articles`，`GET|PATCH|DELETE /{id}` | 顶层 `category_id` |
| 媒体 | `POST /api/v1/admin/media`（multipart），`GET /api/v1/admin/media`，`DELETE /{id}` | 返回 `{id,url,mime,size,width,height}` |
| 联系表单 | `GET /api/v1/admin/contact-submissions`，`GET|PATCH /{id}` | `status`: `new`/`read`/`handled` |
| 访问统计 | `GET /api/v1/admin/analytics/overview?range=today|7d|30d` | 见 §6.3；权限 `admin.analytics.read` |

### 6.3 `GET /api/v1/admin/analytics/overview`

- `range`：`today`（默认，服务器时区当日）| `7d` | `30d`（含当日的滚动窗口）。
- 权限：`admin.analytics.read`。响应 `Cache-Control: no-store`。

```json
{
  "code": 0, "message": "success",
  "data": {
    "range": "7d",
    "pv": 12345,
    "source_count": 3,
    "sources": [
      { "source": "direct", "count": 8000 },
      { "source": "www.baidu.com", "count": 3000 },
      { "source": "github.com", "count": 1345 }
    ]
  },
  "request_id": "..."
}
```

- `pv`：该时间窗内的页面浏览次数（§4.10 上报条数）。
- `sources`：按去重后的来源（`direct` 或 referrer host）聚合，按 `count` 降序，最多返回前 50。
- `source_count`：`sources` 数量（用于工作台「访问来源」卡片）。
- 注册人数与未读留言不在本接口，由管理端分别取 `/api/v1/admin/users` 与
  `/api/v1/admin/contact-submissions?status=new` 的 `total`。

> 管理端接口的完整 OpenAPI 由后端任务在 `backend/server/docs/openapi.yaml` 中补齐；
> 管理后台（`backend/server/admin/`）与 Go 服务同属后端分支，二者以 OpenAPI + 本文件为准。

---

## 7. 前端实现规定

### 7.1 运行时配置

```ts
// nuxt.config.ts
runtimeConfig: {
  apiProxyTarget: 'http://127.0.0.1:8080',   // 仅服务端使用
  apiInternalBase: 'http://127.0.0.1:8080',  // SSR 直连（生产改为内网地址）
  public: { apiBase: '/api' }                // 浏览器唯一入口
}
```

- 所有前端请求统一通过 `useApi()` 组合式函数，禁止组件内直接 `$fetch` 后端地址。
- SSR：使用 `apiInternalBase` 绝对地址；客户端：使用 `public.apiBase` 相对地址。

> **本契约的「前端」适用于所有前端模板**：`frontend/`（Nuxt 4 + Nuxt UI）与 `next/`（Next.js + shadcn/ui）。
> 两者都遵守本节规定。`next/` 的等价运行时变量为 `API_PROXY_TARGET`（浏览器同源 `/api/**` 的代理目标，需在
> **运行时**逐请求转发，不要用构建期固定的 rewrite）与 `API_INTERNAL_BASE`（SSR 直连后端内网地址）。

### 7.2 API 客户端规则

- 自动附加 `Authorization: Bearer`（来自内存 store）。
- 统一解包信封：`code !== 0` 抛 `ApiError { status, code, message, requestId }`。
- 401 单次 refresh 重试（§2）；refresh 请求自身 401 不再重试。
- 错误按 `code` 走 i18n 映射（§3），未知码用兜底文案。
- 生成并发送 `X-Request-ID`，便于与服务端日志对齐。

### 7.3 多语言规则

- 语言列表与默认语言来自 `GET /api/v1/public/settings`；前端不得硬编码语言集合。
- 所有内容请求必须携带当前 `locale`；响应中的 `locale` 若与请求不同（发生回退），仍按响应内容渲染并记录。
- 界面固定文案走前端语言包；后端返回的运营内容不进入语言包。
- URL 策略：默认语言无前缀，其他语言 `/<code>/...`；`<html lang>`、`hreflang`、canonical、sitemap 按语言输出。

### 7.4 缓存规则

- 公开内容请求依赖后端 `Cache-Control`；SSR 侧使用 Nuxt `routeRules` 做 ISR/预渲染。
- 登录态相关请求（`/api/v1/me*`、`/api/v1/auth/*`）必须 `no-store`，不得被缓存。
- 语言切换或内容发布后允许短暂陈旧（最长 `max-age`）。

### 7.5 内容安全

- `body_md` 由后端净化后返回；前端仍禁止 `v-html` 渲染未经 `body_md` 标记的内容。
- 不渲染后端返回的任意 HTML；Markdown 渲染使用固定组件 + 白名单。

---

## 8. 变更记录

| 日期 | 变更 | 作者 |
| ---- | ---- | ---- |
| 2026-09-11 | v1 基线：信封/认证/错误码/公开内容/用户/管理端约定与前端规定 | master-relay |
| 2026-09-11 | v1.1：新增 §5.7 邮件链接约定（验证/重置链接携带 email） | master-relay |
| 2026-09-11 | v1.2：新增 §4.10 访问上报与 §6.3 访问统计接口（PV / 访问来源） | master-relay |
| 2026-09-12 | v1.3：明确契约适用于所有前端模板（新增 `next/`），补充其运行时配置 | master-relay |
