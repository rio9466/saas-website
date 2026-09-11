/**
 * 功能条目管理 API（对应 openapi.yaml 的 admin-content 分组）。
 */
import { http } from "@/utils/http";
import type {
  Envelope,
  Feature,
  FeaturePageData,
  FeatureWrite
} from "./contract";

export type ListFeaturesParams = {
  page?: number;
  page_size?: number;
};

export type FeaturePayload = FeatureWrite;

export const listFeaturesApi = (params: ListFeaturesParams = {}) => {
  return http.request<Envelope<FeaturePageData>>("get", "/features", {
    params
  });
};

export const createFeatureApi = (data: FeaturePayload) => {
  return http.request<Envelope<Feature>>("post", "/features", { data });
};

export const updateFeatureApi = (id: string, data: FeaturePayload) => {
  return http.request<Envelope<Feature>>("patch", `/features/${id}`, { data });
};

export const deleteFeatureApi = (id: string) => {
  return http.request<Envelope<Record<string, never>>>(
    "delete",
    `/features/${id}`
  );
};
