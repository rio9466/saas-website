/**
 * 管理员头像解析（工单：管理员用户菜单头像）。
 *
 * 规则：现有头像非空时优先使用现有头像；为空时使用指定默认头像
 * （GitHub 头像 URL，远程加载失败时由调用方回退为首字母占位）。
 */

/** 管理员无头像时使用的默认头像 URL（不下载到仓库，仅引用）。 */
export const DEFAULT_ADMIN_AVATAR_URL =
  "https://avatars.githubusercontent.com/u/310554857?v=4";

/** 已有头像优先；为空/空白时回退默认头像 URL。 */
export function resolveAdminAvatar(avatar?: string | null): string {
  return typeof avatar === "string" && avatar.trim() !== ""
    ? avatar
    : DEFAULT_ADMIN_AVATAR_URL;
}
