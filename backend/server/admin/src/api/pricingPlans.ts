/**
 * 价格方案管理 API（对应 openapi.yaml 的 admin-content 分组）。
 * `features` 为按语言的逐行功能清单（顶层非翻译字段）。
 */
import { http } from "@/utils/http";
import type {
  Envelope,
  PricingPlan,
  PricingPlanPageData,
  PricingPlanWrite
} from "./contract";

export type ListPricingPlansParams = {
  page?: number;
  page_size?: number;
};

export type PricingPlanPayload = PricingPlanWrite;

export const listPricingPlansApi = (params: ListPricingPlansParams = {}) => {
  return http.request<Envelope<PricingPlanPageData>>("get", "/pricing-plans", {
    params
  });
};

export const createPricingPlanApi = (data: PricingPlanPayload) => {
  return http.request<Envelope<PricingPlan>>("post", "/pricing-plans", {
    data
  });
};

export const updatePricingPlanApi = (id: string, data: PricingPlanPayload) => {
  return http.request<Envelope<PricingPlan>>("patch", `/pricing-plans/${id}`, {
    data
  });
};

export const deletePricingPlanApi = (id: string) => {
  return http.request<Envelope<Record<string, never>>>(
    "delete",
    `/pricing-plans/${id}`
  );
};
