import { defineStore } from "pinia";
import {
  type userType,
  store,
  router,
  resetRouter,
  routerArrays
} from "../utils";
import {
  type LoginParams,
  type ChangePasswordParams,
  loginApi,
  refreshTokenApi,
  logoutApi,
  getProfileApi,
  changePasswordApi,
  updateMyProfileApi
} from "@/api/user";
import type { AdministratorProfile } from "@/api/contract";
import { useMultiTagsStoreHook } from "./multiTags";
import { setAccessToken, getAccessToken, clearAccessToken } from "@/utils/auth";
import { registerRefreshHandler, setAuthExpiredHandler } from "@/utils/http";

/**
 * 认证状态（easy-admin）
 *
 * 安全约束：
 * - access token 与角色/权限等认证信息只保存在内存，刷新页面即丢失；
 *   冷启动由路由守卫调用 restoreSession()（refresh + /me）恢复会话。
 * - 绝不把 token 或认证信息写入 localStorage / sessionStorage / IndexedDB /
 *   Pinia 持久化 / 脚本可读 cookie。
 */
export const useUserStore = defineStore("pure-user", {
  state: (): userType => ({
    // 头像（空则不展示）
    avatar: "",
    // 用户名
    username: "",
    // 昵称（display_name）
    nickname: "",
    // 页面级别权限（角色代码，来自 /me）
    roles: [],
    // 按钮级别权限（有效权限代码，来自 /me）
    permissions: [],
    // 当前管理员 id
    adminId: "",
    // 是否已加载 /me 资料
    profileLoaded: false
  }),
  actions: {
    /** 从 /me 响应填充资料 */
    SET_PROFILE(profile: AdministratorProfile) {
      this.adminId = profile.id;
      this.username = profile.username;
      this.nickname = profile.display_name || profile.username;
      this.roles = profile.role_codes ?? [];
      this.permissions = profile.permission_codes ?? [];
      this.profileLoaded = true;
    },
    /** 仅内存认证状态，不落任何 storage */
    SET_ACCESS_TOKEN(token: string) {
      setAccessToken(token);
    },
    /** 清空内存认证状态 */
    resetAuthState() {
      this.username = "";
      this.nickname = "";
      this.roles = [];
      this.permissions = [];
      this.adminId = "";
      this.profileLoaded = false;
      clearAccessToken();
    },
    /** 登入：登录接口返回 access token（仅内存），再拉取 /me 资料 */
    async loginByUsername(data: LoginParams) {
      const res = await loginApi(data);
      if (!res?.data?.access_token) {
        throw new Error("登录响应缺少 access token");
      }
      this.SET_ACCESS_TOKEN(res.data.access_token);
      await this.getProfile();
    },
    /** 拉取当前管理员资料（/me） */
    async getProfile() {
      const res = await getProfileApi();
      if (!res?.data) {
        throw new Error("/me 响应缺少资料");
      }
      this.SET_PROFILE(res.data);
    },
    /** 冷启动恢复会话：refresh（HttpOnly Cookie）→ /me */
    async restoreSession(): Promise<boolean> {
      if (getAccessToken()) return true;
      try {
        await this.handRefreshToken();
        await this.getProfile();
        return true;
      } catch {
        this.resetAuthState();
        return false;
      }
    },
    /** 刷新 access token（单飞逻辑在 http 层） */
    async handRefreshToken() {
      const res = await refreshTokenApi();
      if (!res?.data?.access_token) {
        throw new Error("刷新响应缺少 access token");
      }
      this.SET_ACCESS_TOKEN(res.data.access_token);
      return res.data.access_token;
    },
    /** 退出登录：调用后端吊销会话，再清理前端状态 */
    async logOut() {
      try {
        await logoutApi();
      } catch {
        // 后端吊销失败时仍清理前端状态，避免出现"看起来已登录"的僵尸会话
      } finally {
        this.resetAuthState();
        useMultiTagsStoreHook().handleTags("equal", [...routerArrays]);
        resetRouter();
        router.push("/login");
      }
    },
    /** 修改当前密码（成功后后端吊销会话，需重新登录） */
    async changePassword(data: ChangePasswordParams) {
      await changePasswordApi(data);
    },
    /** 修改自己的显示名称（无需 admin.user.update）；成功后刷新内存资料并立即同步导航栏 */
    async updateMyProfile(displayName: string) {
      const res = await updateMyProfileApi({ display_name: displayName });
      if (!res?.data) {
        throw new Error("更新响应缺少资料");
      }
      this.SET_PROFILE(res.data);
      return res.data;
    },
    /** 修改昵称（本地展示字段） */
    SET_NICKNAME_VAL(value: string) {
      this.nickname = value;
    }
  }
});

export function useUserStoreHook() {
  return useUserStore(store);
}

/**
 * 注册 http 层的刷新与认证过期处理。
 * - 刷新：调用 /auth/refresh（HttpOnly Cookie 自动携带），成功后 access token
 *   仅写入内存。
 * - 认证过期：清理内存认证状态并跳转登录页。
 */
registerRefreshHandler(() => {
  return useUserStoreHook().handRefreshToken();
});

setAuthExpiredHandler(() => {
  const userStore = useUserStoreHook();
  userStore.resetAuthState();
  useMultiTagsStoreHook().handleTags("equal", [...routerArrays]);
  resetRouter();
  router.push("/login");
});
