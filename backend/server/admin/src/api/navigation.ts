/**
 * 导航管理 API（对应 openapi.yaml 的 admin-content 分组）。
 * header/footer 两处导航，支持排序、显隐与父子层级。
 */
import { http } from "@/utils/http";
import type {
  Envelope,
  NavigationItem,
  NavigationItemPageData,
  NavigationItemWrite
} from "./contract";

export type ListNavigationItemsParams = {
  page?: number;
  page_size?: number;
  placement?: "header" | "footer";
};

export type NavigationItemPayload = NavigationItemWrite;

export const listNavigationItemsApi = (
  params: ListNavigationItemsParams = {}
) => {
  return http.request<Envelope<NavigationItemPageData>>(
    "get",
    "/navigation-items",
    { params }
  );
};

export const createNavigationItemApi = (data: NavigationItemPayload) => {
  return http.request<Envelope<NavigationItem>>("post", "/navigation-items", {
    data
  });
};

export const updateNavigationItemApi = (
  id: string,
  data: NavigationItemPayload
) => {
  return http.request<Envelope<NavigationItem>>(
    "patch",
    `/navigation-items/${id}`,
    { data }
  );
};

export const deleteNavigationItemApi = (id: string) => {
  return http.request<Envelope<Record<string, never>>>(
    "delete",
    `/navigation-items/${id}`
  );
};
