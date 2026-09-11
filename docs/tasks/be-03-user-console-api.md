# Task: BE-03 用户控制台接口（改密 / 资料 / 积分流水 / 找回密码）

## Goal

补齐业务用户自助能力：修改密码、更新基础资料、查询本人积分流水，以及忘记/重置密码。
不新增数据表，复用现有用户、会话、审计与邮件基础设施。

## Dependencies

- 无（可并行）。契约：`docs/api/frontend-api-contract.md` §5。

## Scope

### 1. `PATCH /api/v1/me`（契约 §5.3）

- 仅接受 `nickname`、`avatar_url`（可选，至少一个）。
- 拒绝 username/email/积分/等级/状态字段（DTO 层面不绑定这些字段，避免越权）。
- 返回完整用户资料（现有 `toUserMeData`）。
- 写审计记录；`avatar_url` 校验协议白名单（http/https/相对路径）。

### 2. `POST /api/v1/me/password`（契约 §5.4）

- 请求 `{current_password, new_password}`；新密码 8–72 字节（UTF-8 字节数，沿用现有规则）。
- 校验当前密码：错误返回 `40005`（与管理员改密一致）。
- 成功语义：**吊销该用户全部会话**（复用 `auth_epoch` + Redis 会话索引），并清除 refresh cookie
  （`Set-Cookie` 置空、`Max-Age=-1`）。
- 安全事件：pending-first 审计（审计不可用则不改密，返回 `50002`）。

### 3. `GET /api/v1/me/point-transactions`（契约 §5.5）

- 分页参数 `page`、`page_size`（默认 1/20，上限 100），时间倒序。
- 仅返回当前用户自己的流水；复用现有 `pointTransactionData` 序列化（四位小数字符串）。
- 认证复用 `AuthenticateUser` 中间件。

### 4. 找回密码（P1，契约 §5.6）

- `POST /api/v1/auth/forgot-password`：请求 `{email}`；**无论邮箱是否存在都返回 200 `{}`**，
  不得泄露账号存在性；存在则生成一次性令牌并通过 SMTP 发送重置链接。
- `POST /api/v1/auth/reset-password`：请求 `{email, token, new_password}`；令牌单次有效、加密存储
  （沿用 `easy-admin:user-verify:<id>` 的 SHA-256 + Lua 比较删除模式，新增命名空间
  `easy-admin:user-reset:<id>`）；失败返回 `40016`；成功后吊销全部会话。
- 限流：forgot 3 次/小时/IP + 按邮箱哈希；reset 10 次/小时/IP。Redis 故障按已限流处理。
- 令牌 TTL：默认 1 小时（可通过配置项覆盖；无则内建常量并在 OpenAPI 说明）。
- 邮件不可用（`50003`）时 forgot 接口的响应不得区分邮箱存在性——统一返回 200，仅记录日志。
- **邮件链接**：按契约 §5.7，重置邮件链接必须为
  `{public_frontend_url}/reset-password?email=<urlencoded>&token=<token>`。
  同时修正既有注册验证邮件链接（`verificationLink`）：目前只带 `token`，
  而 `POST /api/v1/auth/verify-email` 需要 `email`+`token`；
  必须在 link query 中加入 `email`，使链接可跨设备打开。更新受影响的测试。

### 5. 路由与 OpenAPI

- 在 `internal/transport/http/router.go` 的 user 分组下注册新路由。
- `backend/server/docs/openapi.yaml` 补齐路径、请求/响应 schema、错误响应。

## Out of scope

- 邮箱改绑（PRD Q5，本期不做）。
- 前端页面（FE-03）。
- 修改现有 `/me` 返回结构（已满足需求，不要改动）。
- 管理员侧接口。

## Files / areas

- `backend/server/internal/service/usersvc/**`（新增方法）
- `backend/server/internal/transport/http/handler/user_auth_handlers.go`、
  `dto_user.go`、`user_service.go` / `user_adapter.go`
- `backend/server/internal/transport/http/router.go`
- `backend/server/docs/openapi.yaml`
- 相关 `_test.go`（含 http 集成测试）

## Acceptance criteria

- [ ] `PATCH /api/v1/me` 只改昵称/头像，传 email/points 等字段无效果。
- [ ] 改密成功返回 200 `{}` 且响应清除 refresh cookie；旧 access/refresh 全部失效（再请求 `/me` 返回 401）。
- [ ] 改密当前密码错误返回 `40005`；新密码不足 8 字节返回 `10001`。
- [ ] `GET /api/v1/me/point-transactions` 只返回本人数据，分页字段符合契约。
- [ ] forgot-password 对不存在邮箱与存在邮箱返回完全一致的响应与状态码。
- [ ] 验证邮件与重置邮件链接均自包含 `email`+`token`，格式符合契约 §5.7。
- [ ] reset-password 令牌重放返回 `40016`；成功后旧会话失效。
- [ ] 触发限流返回 `42901`。
- [ ] 认证接口响应头/错误码与现有用户接口一致（信封、`request_id`）。
- [ ] OpenAPI 与实现一致；`make check` 全绿。

## How to verify

```bash
cd backend/server
make fmt && make vet && make test && make build
go test ./internal/service/usersvc/... ./internal/transport/http/... -run 'User|Password|Point'
```

预期：新增测试通过；`make check` 全绿。

## Branch / base

- branch: `be-user-console-api`
- base: `master-relay`
