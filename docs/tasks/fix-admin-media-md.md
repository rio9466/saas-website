# Task: ADMIN-UX-02 媒体选择器宽度修复 + Markdown 编辑器固定高度

## Goal

修复管理后台内容编辑的两个体验问题。只改 `backend/server/admin/**`。

## 问题 1：Markdown 编辑器没有固定高度

**文件**：`src/views/content/components/MarkdownEditor.vue`

**现象**：Vditor 未设置高度，随正文内容不断撑高（长文档会把弹窗拉得很长）。

**要求**：
- 给 Vditor 设置**固定高度**（建议 360px，做成组件 `height` prop，默认 360），内容超出时在编辑器内部**上下滚动**。
- 多语言 Tab 下每个语言各自一个编辑器，均保持固定高度。
- 亮/暗主题、`cache.enable=false`、销毁逻辑保持不变。

## 问题 2：媒体选择器宽度不足、文字挤在一起

**文件**：`src/views/content/components/MediaPicker.vue`

**现象（用户截图）**：所有使用媒体选择的弹窗（站点信息、首页、功能、价格、页面等）里，
输入框被挤压、占位文字「可直接填写 URL，或从媒…」被截断，「媒体库」「上传」两个按钮挤在一起重叠。

**根因**：`el-input` 的 `#append` 插槽里放了**两个** `el-button`；该插槽只适合单元素/按钮组，
多按钮会破坏输入框宽度计算，导致挤压与重叠。

**要求**：
- 重构布局：输入框占满可用宽度，两个按钮独立摆放不重叠（例如 `display:flex` 一行：
  `<el-input class="flex-1">` + 「媒体库」按钮 + 「上传」按钮；或用 `el-button-group` 放进单个 append 容器）。
- 组件在其所有使用场景（含 `LocaleTranslationTabs` 的 `field.type === 'media'`、系统配置的站点信息）都占满容器宽度，无截断/重叠。
- 媒体库弹窗（720px）与分页、上传、预览逻辑保持不变。

## Out of scope

- 后端 Go、openapi、frontend、根文档、契约。
- 其它非内容编辑页面。

## Files / areas

- `backend/server/admin/src/views/content/components/MarkdownEditor.vue`
- `backend/server/admin/src/views/content/components/MediaPicker.vue`
- （如需要的少量样式调整）`backend/server/admin/src/views/content/components/LocaleTranslationTabs.vue`

## 运行环境

- 后端已在跑 `http://127.0.0.1:8100`；管理后台自测请用**本地不提交**的 vite 配置（如端口 8859、代理到 8100），
  端口 3100/8100/8849 已被占用，不要动它们，也不要提交运行态文件。

## Acceptance criteria

- [ ] Markdown 编辑器固定高度，内容超出在编辑器内部上下滚动；多语言 Tab 下每个语言均如此。
- [ ] 媒体选择器在所有弹窗中宽度占满、无文字截断、按钮不重叠。
- [ ] `pnpm install --frozen-lockfile && pnpm lint && pnpm typecheck && pnpm test && pnpm build` 全绿。
- [ ] 未改动 `server/` Go 代码与 `server/docs/openapi.yaml`，未手工编辑 `types/api.generated.ts`。

## How to verify

```bash
cd backend/server/admin
pnpm install
pnpm lint && pnpm typecheck && pnpm test && pnpm build
# 起 dev（本地配置指向 8100），手工核对：
#   首页区块/页面/文档等弹窗：正文 Vditor 高度固定、超出滚动
#   主图/功能配图/站点信息等媒体字段：输入框与「媒体库」「上传」按钮宽度正常、无重叠
```

## Branch / base

- branch: `fix-admin-media-md`
- base: `master-relay`
