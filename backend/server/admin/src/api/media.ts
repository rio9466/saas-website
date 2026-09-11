/**
 * 媒体库 API（对应 openapi.yaml 的 admin-content 分组）。
 * 上传为 multipart/form-data（字段 `file`）；返回 id/url/mime/size/width/height。
 */
import { http } from "@/utils/http";
import type { Envelope, MediaAsset, MediaAssetPageData } from "./contract";

export type ListMediaParams = {
  page?: number;
  page_size?: number;
};

export const listMediaApi = (params: ListMediaParams = {}) => {
  return http.request<Envelope<MediaAssetPageData>>("get", "/media", {
    params
  });
};

export const uploadMediaApi = (file: File) => {
  const data = new FormData();
  data.append("file", file);
  // 显式声明 multipart，避免 axios 默认的 application/json 把 FormData 序列化；
  // 浏览器随后会补上带 boundary 的 Content-Type。
  return http.request<Envelope<MediaAsset>>("post", "/media", {
    data,
    headers: { "Content-Type": "multipart/form-data" }
  });
};

export const deleteMediaApi = (id: string) => {
  return http.request<Envelope<Record<string, never>>>(
    "delete",
    `/media/${id}`
  );
};
