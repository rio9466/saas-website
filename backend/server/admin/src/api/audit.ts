/**
 * 审计日志只读 API（对应 openapi.yaml 的 admin-audit 分组）。
 * 本模块不提供任何变更 / 导出操作。
 */
import { http } from "@/utils/http";
import type {
  Envelope,
  AuditEvent,
  AuditPageData,
  AuditOutcome
} from "./contract";

export type ListAuditEventsParams = {
  page?: number;
  page_size?: number;
  actor_id?: string;
  action?: string;
  resource_type?: string;
  outcome?: AuditOutcome;
  request_id?: string;
  from?: string;
  to?: string;
};

export const listAuditEventsApi = (params: ListAuditEventsParams) => {
  return http.request<Envelope<AuditPageData>>("get", "/audit-events", {
    params
  });
};

export const getAuditEventApi = (id: string) => {
  return http.request<Envelope<AuditEvent>>("get", `/audit-events/${id}`);
};
