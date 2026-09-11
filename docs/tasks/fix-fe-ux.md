# Task: FIX-FE-UX 前端体验修复（语言切换/记忆、表单红框、页脚居中）

## Goal

修复前端测试中发现的三个体验问题。只改 `frontend/**`。

## Issue 1：语言切换器

**文件**：`frontend/app/components/LocaleSwitcher.vue`、`frontend/nuxt.config.ts`（i18n 配置）

**现象**：
1. 下拉列表当前语言项带一个对勾图标，要求**移除该图标**。
2. **切到中文后切不回英文**（再次点击不生效）。
3. **没有记忆**：用户选过某种语言后，下次进入站点应默认就是该语言。

**背景**：当前 `nuxt.config.ts` 的 i18n 配置为 `detectBrowserLanguage: false`（无持久化）；
`strategy: 'prefix_except_default'`（默认语言无前缀，其它语言 `/<code>/...`）。

**要求**：
- 移除当前语言项的对勾图标（如按钮上的语言图标也不需要，请与用户确认后再删）。
- 修复双向切换：任意语言之间都能切换并生效。
- 记住用户选择：写入持久化（cookie 或 localStorage），下次访问站点根路径（默认语言路径）时默认使用该语言；直接访问 `/zh-CN/...` 等带前缀 URL 仍以 URL 为准。
- 不得引入重定向循环；语言集合仍来自后端 `settings.locales`（不硬编码到切换器逻辑）。

## Issue 2：登录/注册表单输入框变红

**文件**：`frontend/app/pages/login.vue`、`frontend/app/pages/register.vue`

**现象**：账号密码完全正确时，输入框边框仍会**变红一下**，随后才登录/注册成功。

**背景**：页面用 `UFormField :error + required` 手写 `validate()`，提交前设置 `errors.*`。

**要求**：
- 复现并**定位红框来源**（可能来自 `required`、浏览器 `:invalid`、错误状态未先清空、或 autofill 未触发 `v-model`）。
- 校验通过且请求成功时，输入框**不应**出现红色错误边框。
- 校验失败时仍要正确显示红框与错误文案（回归）。

## Issue 3：页脚版权文字未居中

**文件**：`frontend/app/components/AppFooter.vue`

**现象**：底部版权文字显示在最左边，应该**居中**。

**背景**：`UFooter` 的 `#bottom` 用了 `flex ... sm:justify-between`；当 ICP 为空时只剩版权，靠左。

**要求**：
- 版权文字居中显示。
- 有 ICP 备案号时布局合理（例如版权居中、ICP 另起或同样居中），不要挤在两端。

## Out of scope

- 后端、管理后台、契约、根文档。
- 其它页面功能改动。

## Files / areas

- `frontend/app/components/LocaleSwitcher.vue`
- `frontend/nuxt.config.ts`（仅 i18n 语言持久化相关配置）
- `frontend/app/pages/login.vue`、`frontend/app/pages/register.vue`
- `frontend/app/components/AppFooter.vue`
- `frontend/app/composables/useSiteSettings.ts`（如需要）
- `frontend/i18n/locales/*.json`（如新增文案）

## Acceptance criteria

- [ ] 语言切换器无对勾图标；任意语言可双向切换生效。
- [ ] 用户选过的语言在下次进入站点根路径时默认生效；带前缀 URL 仍按 URL。
- [ ] 登录/注册在凭据正确并成功时无红色错误边框；凭据错误时仍显示红框与文案。
- [ ] 页脚版权文字居中（含无 ICP 与有 ICP 两种情况）。
- [ ] `pnpm lint && pnpm typecheck && pnpm build` 全绿。

## How to verify

```bash
cd frontend
pnpm lint && pnpm typecheck && pnpm build
# 起后端后 pnpm dev：
#   切换语言 → 双向切换；刷新/重开根路径 → 语言被记住
#   /login 与 /register：正确凭据无红框；错误凭据有红框
#   页脚：版权居中
```

## Branch / base

- branch: `fix-fe-ux`
- base: `master-relay`
