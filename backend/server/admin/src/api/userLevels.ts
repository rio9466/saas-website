/**
 * 用户等级管理 API（对应 openapi.yaml 的 admin-user-levels 分组）。
 * 自动等级按累积消费积分选取最高且 <= 的已启用阈值；手动等级保持权威直到
 * 切回自动。启用阈值的歧义由后端唯一索引拒绝。
 */
import { http } from "@/utils/http";
import type { Envelope, UserLevel, UserLevelListData } from "./contract";

export type CreateUserLevelParams = {
  code: string;
  name: string;
  icon_url?: string;
  threshold_points: string;
  sort_order?: number;
  enabled?: boolean;
};

export type UpdateUserLevelParams = {
  name?: string;
  icon_url?: string;
  threshold_points?: string;
  sort_order?: number;
  enabled?: boolean;
};

export const listUserLevelsApi = () => {
  return http.request<Envelope<UserLevelListData>>("get", "/user-levels");
};

export const getUserLevelApi = (id: string) => {
  return http.request<Envelope<UserLevel>>("get", `/user-levels/${id}`);
};

export const createUserLevelApi = (data: CreateUserLevelParams) => {
  return http.request<Envelope<UserLevel>>("post", "/user-levels", { data });
};

export const updateUserLevelApi = (id: string, data: UpdateUserLevelParams) => {
  return http.request<Envelope<UserLevel>>("patch", `/user-levels/${id}`, {
    data
  });
};
