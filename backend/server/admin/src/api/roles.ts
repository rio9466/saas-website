/**
 * 角色 / 权限 API（对应 openapi.yaml 的 admin-roles 分组）。
 */
import { http } from "@/utils/http";
import type {
  Envelope,
  Role,
  RoleListData,
  Permission,
  PermissionListData
} from "./contract";

export type CreateRoleParams = {
  code: string;
  name: string;
  description?: string;
};

export type UpdateRoleParams = {
  name?: string;
  description?: string;
  enabled?: boolean;
};

export type ReplaceRolePermissionsParams = {
  permission_codes: string[];
};

export const listRolesApi = () => {
  return http.request<Envelope<RoleListData>>("get", "/roles");
};

export const getRoleApi = (id: string) => {
  return http.request<Envelope<Role>>("get", `/roles/${id}`);
};

export const createRoleApi = (data: CreateRoleParams) => {
  return http.request<Envelope<Role>>("post", "/roles", { data });
};

export const updateRoleApi = (id: string, data: UpdateRoleParams) => {
  return http.request<Envelope<Role>>("patch", `/roles/${id}`, { data });
};

export const replaceRolePermissionsApi = (
  id: string,
  data: ReplaceRolePermissionsParams
) => {
  return http.request<Envelope<Role>>("put", `/roles/${id}/permissions`, {
    data
  });
};

export const listPermissionsApi = () => {
  return http.request<Envelope<PermissionListData>>("get", "/permissions");
};

export type { Permission };
