/**
 * 系统设置 API（对应 openapi.yaml 的 admin-system-settings 分组）。
 * 登录/注册开关、邮件验证、SMTP（密码仅可写，读取只返回 password_configured）。
 * 提交必须携带当前 version（乐观锁），版本过期返回 40013。
 */
import { http } from "@/utils/http";
import type { Envelope, SystemSettings } from "./contract";

export type UpdateSystemSettingsParams = {
  platform_name: string;
  public_frontend_url: string;
  public_api_url: string;
  registration_enabled: boolean;
  username_login_enabled: boolean;
  email_login_enabled: boolean;
  email_verification_required: boolean;
  default_level_id: string;
  default_avatar_url?: string;
  registration_points: string;
  smtp_enabled: boolean;
  smtp_host: string;
  smtp_port: number;
  smtp_username: string;
  smtp_password?: string;
  smtp_from_email: string;
  smtp_from_name: string;
  smtp_tls_mode: "none" | "starttls" | "ssl";
  version: number;
};

export const getSystemSettingsApi = () => {
  return http.request<Envelope<SystemSettings>>("get", "/system-settings");
};

export const updateSystemSettingsApi = (data: UpdateSystemSettingsParams) => {
  return http.request<Envelope<SystemSettings>>("put", "/system-settings", {
    data
  });
};
