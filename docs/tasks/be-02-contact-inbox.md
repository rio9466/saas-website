# Task: BE-02 联系表单收件箱 + 媒体上传

## Goal

实现官网「联系我们」表单的服务端闭环（落库 + 反垃圾 + 通知邮件 + 管理端收件箱），
以及内容运营所需的媒体上传能力。

## Dependencies

- BE-01（复用 `site_settings.contact_email` 与权限/审计基础设施）。
- 契约：`docs/api/frontend-api-contract.md` §4.8、§6.2（媒体与 contact-submissions）。

## Scope

### 1. 数据模型（`backend/server/migrations/primary/000010_contact_and_media.up.sql` / `.down.sql`）

- `contact_submissions`：`name`、`email`、`company`、`message`、`locale`、`status`
  （`new`/`read`/`handled`，默认 `new`）、`source_ip`、`user_agent`、`created_at`、`updated_at`。
- `media_assets`：`url`、`original_name`、`mime`、`size_bytes`、`width`、`height`、`created_by`、`created_at`。

### 2. 公开接口

- `POST /api/v1/public/contact`（契约 §4.8）：
  - 校验 `name`/`email`/`message`/`consent=true`；`email` 格式校验；`message` 长度上限（建议 5000）。
  - 蜜罐字段 `website` 非空 → 静默返回 201 且不入库。
  - 限流：固定窗口 Redis，5 次/小时/IP；超限 `42901`。Redis 故障按「已限流」处理（fail closed）。
  - 落库成功后**尽力**向 `site_settings.contact_email` 发送通知邮件；邮件失败不影响 201，
    记录高优先级日志，不向客户端暴露内部错误。
  - 响应 `Cache-Control: no-store`。

### 3. 管理端接口

- `GET /api/v1/admin/contact-submissions?status=&page=&page_size=`：分页列表，时间倒序。
- `GET /api/v1/admin/contact-submissions/{id}`：详情。
- `PATCH /api/v1/admin/contact-submissions/{id}`：仅允许改 `status`。
- 以上需 `admin.contact.read` / `admin.contact.manage`，写操作走 pending-first 审计。
- `POST /api/v1/admin/media`：`multipart/form-data` 字段 `file`；限制 MIME（png/jpeg/webp/svg/gif）、
  大小（建议 ≤ 5MB）；返回 `{id,url,mime,size,width,height}`。
  `GET /api/v1/admin/media` 分页列表；`DELETE /api/v1/admin/media/{id}`。
  需 `admin.content.manage`。
- 存储：定义 `MediaStore` 接口，默认本地磁盘目录（可配置），便于后续替换对象存储（PRD Q8）。

### 4. OpenAPI

补齐以上路径与 schema，并同步错误响应（`10001`/`42901`）。

## Out of scope

- 管理后台收件箱界面（BE-04）。
- 验证码/第三方反垃圾（PRD Q9，本期用限流 + 蜜罐）。
- 定时发布、内容版本历史。

## Files / areas

- `backend/server/migrations/primary/000010*`（新增）
- `backend/server/internal/domain/**`（contact/media）、`internal/repository/primary/**`、
  `internal/service/**`、`internal/platform/ratelimit/**`（复用）、`internal/platform/mailer/**`（复用）、
  `internal/transport/http/handler/**`、`internal/transport/http/router.go`
- `backend/server/docs/openapi.yaml`
- 相关 `_test.go`

## Acceptance criteria

- [ ] `POST /api/v1/public/contact` 合法请求返回 201 且入库；`consent=false` 返回 `10001`。
- [ ] 蜜罐命中返回 201 且 `contact_submissions` 无新增行。
- [ ] 第 6 次同 IP 请求返回 `42901`。
- [ ] SMTP 不可用时提交仍成功入库，只记录日志。
- [ ] 收件箱列表按状态筛选、分页正确；`PATCH` 只能改状态。
- [ ] 无权限管理员访问收件箱返回 `30001`。
- [ ] 媒体上传类型/大小超限被拒绝（400/`10001`），合法图片返回可访问 URL。
- [ ] 管理写操作产生审计记录；OpenAPI 与实现一致。

## How to verify

```bash
cd backend/server
make fmt && make vet && make test && make build
go run ./cmd/server -config configs/config.local.toml
curl -sS -X POST 'http://127.0.0.1:8080/api/v1/public/contact' \
  -H 'Content-Type: application/json' \
  -d '{"name":"A","email":"a@b.com","message":"hi","consent":true,"locale":"en","website":""}'
# 连续调用第 6 次，预期 code 42901
```

预期：首次 201 入库；重复触发返回 42901；`make check` 全绿。

## Branch / base

- branch: `be-contact-inbox`
- base: `master-relay`
