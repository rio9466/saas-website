# Task: BE-ANALYTICS 访问统计（PV + 访问来源）

## Goal

实现官网访问上报与访问统计聚合，供管理后台工作台使用。契约：`docs/api/frontend-api-contract.md` §4.10、§6.1、§6.3。

## Scope

### 1. 数据模型（`backend/server/migrations/primary/000011_analytics.up.sql` / `.down.sql`）

- `page_views`：`id`、`path`（text，≤512）、`source`（text：`direct` 或 referrer host）、
  `locale`（text，可空）、`created_at`（timestamptz，默认 now）。
- 索引：`created_at`、`(source, created_at)`。

### 2. 权限

- 新增 `admin.analytics.read`（命名沿用 §6.1），授予 `super_admin`（沿用 000009 写法）；
  如放在同一迁移里也可。

### 3. 公开上报 `POST /api/v1/public/page-view`（契约 §4.10）

- 请求 `{ path, referrer?, locale? }`：`path` 必填且以 `/` 开头、≤512，否则静默丢弃（仍返回 200）。
- `source` 解析：`referrer` 为空/非 http(s) → `direct`；否则取 **host**（保留子域，如 `www.baidu.com`），
  **不做域名识别**。
- 成功 `200 {code:0,...,data:{}}`；响应 `Cache-Control: no-store`。
- 限流：每 IP 每小时上限（宽松，防刷），Redis 故障时 **fail-open**（直接接受）。
- 无需登录，不写审计。

### 4. 管理端聚合 `GET /api/v1/admin/analytics/overview?range=` （契约 §6.3）

- `range`：`today`（默认，服务器时区当日）| `7d` | `30d`（含当日的滚动窗口）。
- 权限 `admin.analytics.read`；响应 `Cache-Control: no-store`。
- 返回 `{ range, pv, source_count, sources: [{source,count}] }`；`sources` 按 `count` 降序，最多 50 条。

### 5. 代码分层（沿用现有结构）

- `internal/domain/analytics/`（来源解析、范围解析）
- `internal/repository/primary/`（插入 + 聚合查询）
- `internal/service/analyticssvc/`
- `internal/transport/http/handler/` + `router.go`
- `backend/server/docs/openapi.yaml`

## Out of scope

- 访问来源的域名识别/归类（只取 host 与 `direct`）。
- UV/去重、会话、地区等其它维度。
- 前端埋点（FE-ANALYTICS）与管理端界面（ADMIN-DASHBOARD）。

## Files / areas

- `backend/server/migrations/primary/000011*`（新增）
- `backend/server/internal/domain/analytics/**`、`internal/repository/primary/**`、
  `internal/service/analyticssvc/**`、`internal/transport/http/handler/**`、
  `internal/transport/http/router.go`
- `backend/server/docs/openapi.yaml`
- 相关 `_test.go`

## Acceptance criteria

- [ ] §4.10 上报：合法返回 200；`referrer` 为空/外部 → `direct`/host 正确入库；非法 path 不入库。
- [ ] §6.3 聚合：`pv` = 窗口内条数；`sources` 按数量降序；`source_count` 正确；`range` 三种都可用。
- [ ] 无 `admin.analytics.read` 权限访问聚合接口 → `30001`。
- [ ] 迁移 up/down 成对、可重复执行。
- [ ] `docs/openapi.yaml` 与实现一致；`make check` 全绿。

## How to verify

```bash
cd backend/server
make check
# 起服务后：
curl -sS -X POST http://127.0.0.1:8100/api/v1/public/page-view -H 'Content-Type: application/json' \
  -d '{"path":"/features","referrer":"https://www.baidu.com/s?wd=x"}'
curl -sS 'http://127.0.0.1:8100/api/v1/admin/analytics/overview?range=7d'   # 需管理员 token
```

## Branch / base

- branch: `be-analytics`
- base: `master-relay`
