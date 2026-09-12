# Task: ADMIN-DASHBOARD 工作台改造（4 卡片 + 访问来源列表）

## Goal

把 easy-admin 演示版工作台替换为 4 个卡片 + 一个访问来源列表，移除所有无用的演示组件。

## 4 个卡片

| 卡片 | 数据源 |
| ---- | ------ |
| 访问人数（PV） | `GET /api/v1/admin/analytics/overview?range=` 的 `pv`（契约 §6.3） |
| 访问来源 | 同上 `source_count`（有多少来源跳转到本站） |
| 注册人数 | `GET /api/v1/admin/users?page_size=1` 的 `data.total` |
| 未读留言 | `GET /api/v1/admin/contact-submissions?status=new&page_size=1` 的 `data.total` |

- 顶部提供时间范围切换：**今日 / 7天 / 30天**，仅影响「访问人数」「访问来源」两张卡片（调用 overview 的 `range`）；
  注册人数/未读留言是当前总量，切范围不改变。
- 权限：无 `admin.analytics.read` 时隐藏两张访问统计卡片（注册/未读按各自权限正常显示）。

## 新增：访问来源列表组件

- 展示 `overview.sources`（`[{source,count}]`，已按数量降序），左列来源名（`direct` 显示为「直接访问」），右列次数。
- 放在工作台 4 卡片下方；空数据显示「暂无数据」。

## 移除无用组件

- 删除演示用的折线/柱状/环形图、最新动态表、问候语等：
  `src/views/dashboard/components/{charts,table}/**`、`data.ts`、`greeting.ts`、`utils.ts`（按需），
  以及 `index.vue` 中相应引用；不要留下死代码或未使用 import。

## API 模块

- 新增 `src/api/analytics.ts`：`getAnalyticsOverviewApi(range)`，`Envelope<AnalyticsOverview>`；
  类型加在 `src/api/contract.ts` 手写别名区。
- 复用现有 `listUsersApi`（`src/api/users.ts`）与 `listContactSubmissionsApi`（`src/api/contactSubmissions.ts`）。

## Out of scope

- 后端/前端官网改动（分别由 BE-ANALYTICS、FE-ANALYTICS 负责）。
- 其它页面。

## Files / areas

- `backend/server/admin/src/views/dashboard/**`（重写 `index.vue`、新增来源列表组件、删除演示组件）
- `backend/server/admin/src/api/analytics.ts`（新增）、`src/api/contract.ts`（追加类型）
- `backend/server/admin/tests/**`（如需要）

## 运行环境

- 后端已在跑 `http://127.0.0.1:8100`；管理后台自测用**本地不提交**的 vite 配置（如端口 8859、代理到 8100）；
  不要占用 3100/8100/8849，不要提交运行态文件。

## Acceptance criteria

- [ ] 工作台仅保留：时间范围切换 + 4 卡片 + 访问来源列表；演示组件与死代码全部移除。
- [ ] 访问人数=PV、访问来源=来源数量、注册人数=用户总数、未读留言=新状态留言数，均与后端一致。
- [ ] 今日/7天/30天切换正确影响两张访问统计卡片。
- [ ] 无 `admin.analytics.read` 时隐藏访问统计卡片；未登录/其它权限行为不变。
- [ ] `pnpm install && pnpm lint && pnpm typecheck && pnpm test && pnpm build` 全绿；未手改 `types/api.generated.ts`。

## How to verify

```bash
cd backend/server/admin
pnpm install
pnpm lint && pnpm typecheck && pnpm test && pnpm build
# 起 dev（本地配置指向 8100），登录后查看工作台：4 卡片 + 来源列表 + 范围切换
```

## Branch / base

- branch: `admin-dashboard`
- base: `master-relay`
