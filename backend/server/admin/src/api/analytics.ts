/**
 * 访问统计 API（对应契约 §6.3 的 analytics overview）。
 * 类型暂未随 OpenAPI 生成，手写在 contract.ts。
 */
import { http } from "@/utils/http";
import type { AnalyticsOverview, AnalyticsRange, Envelope } from "./contract";

export const getAnalyticsOverviewApi = (range: AnalyticsRange) => {
  return http.request<Envelope<AnalyticsOverview>>(
    "get",
    "/analytics/overview",
    { params: { range } }
  );
};
