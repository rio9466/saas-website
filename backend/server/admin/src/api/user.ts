/**
 * 认证与当前管理员 API（真实后端，对应 server/docs/openapi.yaml 的 admin-auth 分组）。
 */
import { http } from "@/utils/http";
import type { Envelope, LoginData, AdministratorProfile } from "./contract";

export type LoginParams = {
  username: string;
  password: string;
};

export type ChangePasswordParams = {
  current_password: string;
  new_password: string;
};

/** 登录：返回 access token（仅内存保存），refresh token 由 HttpOnly Cookie 下发 */
export const loginApi = (data: LoginParams) => {
  return http.request<Envelope<LoginData>>("post", "/auth/login", { data });
};

/** 刷新 access token：凭 HttpOnly Cookie 旋转会话 */
export const refreshTokenApi = () => {
  return http.request<Envelope<LoginData>>("post", "/auth/refresh");
};

/** 退出登录：吊销当前会话 */
export const logoutApi = () => {
  return http.request<Envelope<Record<string, never>>>("post", "/auth/logout");
};

/** 当前管理员资料（角色与有效权限代码） */
export const getProfileApi = () => {
  return http.request<Envelope<AdministratorProfile>>("get", "/me");
};

/** 修改自己的显示名称（无需 admin.user.update，返回完整 /me 资料） */
export const updateMyProfileApi = (data: { display_name: string }) => {
  return http.request<Envelope<AdministratorProfile>>("patch", "/me", {
    data
  });
};

/** 修改当前密码（成功后会话被吊销，需要重新登录） */
export const changePasswordApi = (data: ChangePasswordParams) => {
  return http.request<Envelope<Record<string, never>>>("post", "/me/password", {
    data
  });
};
