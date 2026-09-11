/**
 * 联系表单收件箱 API（对应 openapi.yaml 的 admin-content 分组）。
 * 列表按时间倒序，可按 status 过滤；仅允许流转 status（new/read/handled）。
 */
import { http } from "@/utils/http";
import type {
  ContactStatus,
  ContactSubmission,
  ContactSubmissionPageData,
  Envelope
} from "./contract";

export type ListContactSubmissionsParams = {
  page?: number;
  page_size?: number;
  status?: ContactStatus;
};

export const listContactSubmissionsApi = (
  params: ListContactSubmissionsParams = {}
) => {
  return http.request<Envelope<ContactSubmissionPageData>>(
    "get",
    "/contact-submissions",
    { params }
  );
};

export const getContactSubmissionApi = (id: string) => {
  return http.request<Envelope<ContactSubmission>>(
    "get",
    `/contact-submissions/${id}`
  );
};

export const updateContactSubmissionStatusApi = (
  id: string,
  status: ContactStatus
) => {
  return http.request<Envelope<ContactSubmission>>(
    "patch",
    `/contact-submissions/${id}`,
    { data: { status } }
  );
};
