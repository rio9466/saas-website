# easy-admin 管理后台（server/admin）

基于 [pure-admin-thin](https://github.com/rio9466/pure-admin-thin) v6.2.0
（MIT，来源与改动记录见 [SOURCE-ADAPTATION.md](./SOURCE-ADAPTATION.md)）精简接入的
管理端，对接 `server/docs/openapi.yaml` 契约，UI/交互语言沿用
[vue-pure-admin](https://github.com/pure-admin/vue-pure-admin) v6 系实现。

## 环境要求

- Node `^20.19.0 || >=22.13.0`（`.nvmrc` 为 v22.20.0）
- pnpm >= 9（使用 `pnpm-lock.yaml` 锁定依赖）

## 常用命令

```sh
pnpm install --frozen-lockfile   # 按锁文件安装
pnpm dev                         # 开发（Vite :8848，/api 代理到后端 :8080）
pnpm generate:api                # 从 server/docs/openapi.yaml 生成类型（勿手改生成文件）
pnpm lint
pnpm typecheck
pnpm test                        # vitest 单测
pnpm build                       # 产物在 dist/
```

## 认证与会话（安全模型）

- access token 只保存在内存；refresh token 只存在于后端 HttpOnly Cookie，
  脚本不可读，禁止写入任何浏览器存储。
- 冷启动由路由守卫先调 refresh 再加载 /me 恢复会话。
- 401 后单飞刷新、原请求只重试一次、刷新失败清理认证状态并跳转登录。
- 菜单/路由为本地定义，按 /me 返回的角色与权限过滤；后端始终是授权权威。

## 目录

```text
src/api/          真实 API 客户端（契约适配层见 src/api/contract.ts）
src/router/       本地路由 + 角色/权限元数据 + 守卫
src/store/        user（内存认证状态）等 Pinia store
src/views/        dashboard / system(管理员、角色、审计) / profile / login / error
types/            api.generated.ts（自动生成）与全局类型
tests/            vitest：刷新单飞/重试上限、路由权限过滤、登录校验
```
