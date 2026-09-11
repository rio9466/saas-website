import type { AxiosError } from "axios";

/**
 * 后端统一信封错误码 → 中文安全文案。
 * 代码见 server/internal/domain/apperr/apperr.go（ADM-001 禁止修改后端，
 * 前端仅做展示映射；文案刻意不暴露内部细节，也不区分"账号是否存在"）。
 */
const CODE_MESSAGES: Record<number, string> = {
  10001: "请求参数有误，请检查输入",
  20001: "登录已失效，请重新登录",
  20002: "登录已过期，请重新登录",
  20003: "会话已被安全策略注销，请重新登录",
  30001: "您没有权限执行此操作",
  40001: "操作冲突，请刷新后重试",
  40002: "请求的资源不存在",
  40003: "不能修改最后一个启用的超级管理员",
  40004: "不能对当前登录账号执行此操作",
  40005: "密码错误",
  40006: "账号已被禁用",
  40007: "初始化已完成，不能重复操作",
  50001: "系统依赖服务暂不可用，请稍后重试",
  50002: "审计服务暂不可用，本次操作未执行，请稍后重试"
};

/** 无映射码时按 HTTP 状态给出通用中文文案 */
const STATUS_FALLBACK: Record<number, string> = {
  400: "请求参数有误，请检查输入",
  401: "登录已失效，请重新登录",
  403: "您没有权限执行此操作",
  404: "请求的资源不存在",
  409: "操作冲突，请刷新后重试",
  422: "请求参数有误，请检查输入",
  500: "服务器开小差了，请稍后重试",
  502: "网关异常，请稍后重试",
  503: "服务暂不可用，请稍后重试"
};

/** 从响应信封取应用错误码 */
function getEnvelopeCode(error: unknown): number | null {
  const axiosError = error as AxiosError<{
    code?: number;
    message?: string;
  }>;
  const code = axiosError?.response?.data?.code;
  return typeof code === "number" ? code : null;
}

/**
 * 提取用户可读的错误信息（中文安全文案优先）。
 * 1) 优先按后端应用错误码映射中文文案；
 * 2) 无映射码时按 HTTP 状态给通用中文文案（不直接展示英文后端消息）；
 * 3) 网络层错误给出中文提示。
 */
export function getErrorMessage(error: unknown): string {
  const axiosError = error as AxiosError<{
    code?: number;
    message?: string;
  }>;

  const code = getEnvelopeCode(error);
  if (code !== null && CODE_MESSAGES[code]) {
    return CODE_MESSAGES[code];
  }

  const status = axiosError?.response?.status;
  if (status !== undefined && STATUS_FALLBACK[status]) {
    return STATUS_FALLBACK[status];
  }

  if (axiosError?.code === "ECONNABORTED") return "请求超时，请稍后重试";
  if (!axiosError?.response) return "网络异常，请检查网络连接";
  return "请求失败，请稍后重试";
}

/**
 * 取后端应用错误码（供调用方做分支处理，例如登录 20001 的特殊提示）。
 */
export function getErrorCode(error: unknown): number | null {
  return getEnvelopeCode(error);
}
