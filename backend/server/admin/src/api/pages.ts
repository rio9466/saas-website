/**
 * 页面管理 API（对应 openapi.yaml 的 admin-content 分组）。
 * 顶层 slug/published；翻译在 translations 中按语言编辑。
 */
import { http } from "@/utils/http";
import type {
  ContentPage,
  ContentPageListData,
  Envelope,
  PageWrite
} from "./contract";

export type ListPagesParams = {
  page?: number;
  page_size?: number;
};

export type PagePayload = PageWrite;

export const listPagesApi = (params: ListPagesParams = {}) => {
  return http.request<Envelope<ContentPageListData>>("get", "/pages", {
    params
  });
};

export const getPageApi = (id: string) => {
  return http.request<Envelope<ContentPage>>("get", `/pages/${id}`);
};

export const createPageApi = (data: PagePayload) => {
  return http.request<Envelope<ContentPage>>("post", "/pages", { data });
};

export const updatePageApi = (id: string, data: PagePayload) => {
  return http.request<Envelope<ContentPage>>("patch", `/pages/${id}`, {
    data
  });
};

export const deletePageApi = (id: string) => {
  return http.request<Envelope<Record<string, never>>>(
    "delete",
    `/pages/${id}`
  );
};
