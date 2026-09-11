/**
 * 业务用户（前台用户）管理 API（对应 openapi.yaml 的 admin-business-users 分组）。
 * 与"管理员管理"（administrators）完全分离；用户名/注册 IP/积分/等级/状态/密码
 * 均有专用操作，不允许通过通用 update 静默修改。
 */
import { http } from "@/utils/http";
import type {
  Envelope,
  BusinessUser,
  BusinessUserPageData,
  PointTransactionPageData,
  PointTransaction
} from "./contract";

export type ListUsersParams = {
  page?: number;
  page_size?: number;
  q?: string;
  status?: "pending_verification" | "active" | "disabled";
  level_id?: string;
};

export type CreateUserParams = {
  username: string;
  email: string;
  password: string;
  nickname?: string;
};

export type UpdateUserParams = {
  email?: string;
  nickname?: string;
  avatar_url?: string;
  remark?: string;
};

export type ResetUserPasswordParams = {
  new_password: string;
};

export type AdjustPointsParams = {
  points_delta: string;
  consumption_delta: string;
  reason: string;
  idempotency_key: string;
};

export type AssignLevelParams = {
  level_mode: "auto" | "manual";
  level_id?: string;
};

export const listUsersApi = (params: ListUsersParams) => {
  return http.request<Envelope<BusinessUserPageData>>("get", "/users", {
    params
  });
};

export const getUserApi = (id: string) => {
  return http.request<Envelope<BusinessUser>>("get", `/users/${id}`);
};

export const createUserApi = (data: CreateUserParams) => {
  return http.request<Envelope<BusinessUser>>("post", "/users", { data });
};

export const updateUserApi = (id: string, data: UpdateUserParams) => {
  return http.request<Envelope<BusinessUser>>("patch", `/users/${id}`, {
    data
  });
};

export const enableUserApi = (id: string) => {
  return http.request<Envelope<BusinessUser>>("post", `/users/${id}/enable`);
};

export const disableUserApi = (id: string) => {
  return http.request<Envelope<BusinessUser>>("post", `/users/${id}/disable`);
};

export const resetUserPasswordApi = (
  id: string,
  data: ResetUserPasswordParams
) => {
  return http.request<Envelope<Record<string, never>>>(
    "post",
    `/users/${id}/reset-password`,
    { data }
  );
};

export const adjustUserPointsApi = (id: string, data: AdjustPointsParams) => {
  return http.request<Envelope<PointTransaction>>(
    "post",
    `/users/${id}/points-adjust`,
    { data }
  );
};

export const listUserPointTransactionsApi = (
  id: string,
  params: { page?: number; page_size?: number }
) => {
  return http.request<Envelope<PointTransactionPageData>>(
    "get",
    `/users/${id}/point-transactions`,
    { params }
  );
};

export const assignUserLevelApi = (id: string, data: AssignLevelParams) => {
  return http.request<Envelope<BusinessUser>>("post", `/users/${id}/level`, {
    data
  });
};
