/**
 * 认证状态管理（easy-admin）
 *
 * 安全约束：
 * - access token 只保存在内存中，任何 storage（localStorage / sessionStorage /
 *   IndexedDB / Pinia 持久化 / 脚本可读 cookie）都不允许保存 token。
 * - refresh token 只存在于后端下发的 HttpOnly Cookie 中，脚本不可读，
 *   由浏览器在 same-origin 请求中自动携带。
 * - 页面刷新后内存中的 access token 丢失，由路由守卫先调用 refresh 再加载
 *   /me 恢复会话（见 store/modules/user.ts 的 restoreSession）。
 *
 * 注意：按钮级权限判断在 src/utils/perms.ts（hasPerms），本模块只管理 token。
 */
let accessToken = "";

/** 保存 access token（仅内存） */
export function setAccessToken(token: string) {
  accessToken = token;
}

/** 获取 access token（仅内存） */
export function getAccessToken(): string {
  return accessToken;
}

/** 清空 access token（仅内存） */
export function clearAccessToken() {
  accessToken = "";
}

/** 格式化 token（jwt 格式） */
export const formatToken = (token: string): string => {
  return "Bearer " + token;
};

/** 当前请求路径是否为认证白名单（不需要 access token，且失败时不触发刷新） */
export const authWhiteList = ["/auth/login", "/auth/refresh"];

/** 是否在认证白名单内（用于 axios 请求拦截器） */
export function isAuthWhiteList(url: string): boolean {
  return authWhiteList.some(item => url.endsWith(item));
}
