/**
 * 管理员账号管理 API（对应 openapi.yaml 的 admin-users 分组）。
 */
import { http } from "@/utils/http";
import type {
  Envelope,
  Administrator,
  AdministratorPageData
} from "./contract";

export type ListAdministratorsParams = {
  page?: number;
  page_size?: number;
  q?: string;
};

export type CreateAdministratorParams = {
  username: string;
  password: string;
  display_name?: string;
  role_codes: string[];
};

export type UpdateAdministratorParams = {
  display_name?: string;
};

export type ResetPasswordParams = {
  new_password: string;
};

export type AssignRolesParams = {
  role_codes: string[];
};

export const listAdministratorsApi = (params: ListAdministratorsParams) => {
  return http.request<Envelope<AdministratorPageData>>(
    "get",
    "/administrators",
    {
      params
    }
  );
};

export const getAdministratorApi = (id: string) => {
  return http.request<Envelope<Administrator>>("get", `/administrators/${id}`);
};

export const createAdministratorApi = (data: CreateAdministratorParams) => {
  return http.request<Envelope<Administrator>>("post", "/administrators", {
    data
  });
};

export const updateAdministratorApi = (
  id: string,
  data: UpdateAdministratorParams
) => {
  return http.request<Envelope<Administrator>>(
    "patch",
    `/administrators/${id}`,
    { data }
  );
};

export const enableAdministratorApi = (id: string) => {
  return http.request<Envelope<Administrator>>(
    "post",
    `/administrators/${id}/enable`
  );
};

export const disableAdministratorApi = (id: string) => {
  return http.request<Envelope<Administrator>>(
    "post",
    `/administrators/${id}/disable`
  );
};

export const resetAdministratorPasswordApi = (
  id: string,
  data: ResetPasswordParams
) => {
  return http.request<Envelope<Record<string, never>>>(
    "post",
    `/administrators/${id}/reset-password`,
    { data }
  );
};

export const assignAdministratorRolesApi = (
  id: string,
  data: AssignRolesParams
) => {
  return http.request<Envelope<Administrator>>(
    "put",
    `/administrators/${id}/roles`,
    { data }
  );
};
