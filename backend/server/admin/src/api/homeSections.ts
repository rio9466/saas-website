/**
 * 首页区块管理 API（对应 openapi.yaml 的 admin-content 分组）。
 * `type` 决定 payload 语义；`data` 为非翻译基础数据，`translations` 为按语言的覆盖。
 */
import { http } from "@/utils/http";
import type {
  Envelope,
  HomeSection,
  HomeSectionPageData,
  HomeSectionWrite
} from "./contract";

export type ListHomeSectionsParams = {
  page?: number;
  page_size?: number;
};

export type HomeSectionPayload = HomeSectionWrite;

export const listHomeSectionsApi = (params: ListHomeSectionsParams = {}) => {
  return http.request<Envelope<HomeSectionPageData>>("get", "/home-sections", {
    params
  });
};

export const createHomeSectionApi = (data: HomeSectionPayload) => {
  return http.request<Envelope<HomeSection>>("post", "/home-sections", {
    data
  });
};

export const updateHomeSectionApi = (id: string, data: HomeSectionPayload) => {
  return http.request<Envelope<HomeSection>>("patch", `/home-sections/${id}`, {
    data
  });
};

export const deleteHomeSectionApi = (id: string) => {
  return http.request<Envelope<Record<string, never>>>(
    "delete",
    `/home-sections/${id}`
  );
};
