/**
 * 站点内容设置 API（对应 openapi.yaml 的 admin-content 分组）。
 * 单例资源，提交必须携带当前 version（乐观锁），冲突返回 40013。
 * 翻译统一使用 translations 映射（按语言 Tab 编辑）。
 */
import { http } from "@/utils/http";
import type {
  Envelope,
  PublicSettings,
  SiteSettings,
  SiteSettingsWrite
} from "./contract";

export type UpdateSiteSettingsParams = SiteSettingsWrite;

export const getSiteSettingsApi = () => {
  return http.request<Envelope<SiteSettings>>("get", "/site-settings");
};

export const updateSiteSettingsApi = (data: UpdateSiteSettingsParams) => {
  return http.request<Envelope<SiteSettings>>("put", "/site-settings", {
    data
  });
};

/**
 * 公开设置（同源 `/api/v1/public/settings`，无需登录）。
 * 管理端唯一可读取「已启用语言（supported_locales）」的来源，用于渲染翻译语言 Tab。
 * baseURL 覆盖为 public，避免拼到 /admin 下；令牌可选，公开接口忽略它。
 */
export const getPublicSettingsApi = () => {
  return http.request<Envelope<PublicSettings>>("get", "/settings", {
    baseURL: "/api/v1/public"
  });
};
