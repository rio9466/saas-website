import Axios, {
  type AxiosInstance,
  type AxiosRequestConfig,
  type CustomParamsSerializer
} from "axios";
import type {
  PureHttpError,
  PureHttpResponse,
  PureHttpRequestConfig,
  RequestMethods
} from "./types.d";
import { stringify } from "qs";
import {
  getAccessToken,
  formatToken,
  isAuthWhiteList,
  clearAccessToken
} from "@/utils/auth";
import { RefreshGate, decideAfter401 } from "@/utils/refreshGate";

/**
 * 统一请求客户端（easy-admin）
 *
 * - baseURL 为 `/api/v1/admin`：开发环境经 Vite `/api` 代理到后端，生产环境
 *   与后端同源部署。
 * - access token 仅从内存获取并写入 Authorization 头；refresh token 由浏览器
 *   在 HttpOnly Cookie 中自动携带，脚本不可见。
 * - 401 处理：单飞 refresh（同一时刻只允许一个刷新请求）、原请求只重试一次、
 *   刷新失败立即清理内存认证状态并跳转登录页；登录/刷新接口永不触发刷新流程。
 *
 * createHttpClient 是纯逻辑工厂（token 行为可注入），可用真实 axios + 本地
 * fake server 做拦截器级联测；默认导出 http 为应用共享实例。
 */
const defaultConfig: AxiosRequestConfig = {
  // 统一 API 前缀：开发环境经 Vite /api 代理，生产环境同源 /api/v1
  baseURL: "/api/v1/admin",
  // 请求超时时间
  timeout: 15000,
  // same-origin 代理下 cookie 自动携带；显式声明以强调会话依赖 cookie
  withCredentials: true,
  headers: {
    Accept: "application/json, text/plain, */*",
    "Content-Type": "application/json",
    "X-Requested-With": "XMLHttpRequest"
  },
  // 数组格式参数序列化（https://github.com/axios/axios/issues/5142）
  paramsSerializer: {
    serialize: stringify as unknown as CustomParamsSerializer
  }
};

/** 令牌/认证行为注入点（应用于测试与共享实例） */
export interface HttpTokenApi {
  getToken(): string;
  clearToken(): void;
  refresh(): Promise<string>;
  isWhiteList(url: string): boolean;
  onAuthExpired?: () => void;
}

export interface PureHttpClient {
  request<T>(
    method: RequestMethods,
    url: string,
    param?: AxiosRequestConfig,
    axiosConfig?: PureHttpRequestConfig
  ): Promise<T>;
}

/** 创建带认证/单飞刷新/原请求重试一次的 http 客户端（纯逻辑，可注入 token API） */
export function createHttpClient(
  tokenApi: HttpTokenApi,
  config?: Partial<AxiosRequestConfig>
): PureHttpClient {
  const axiosInstance: AxiosInstance = Axios.create({
    ...defaultConfig,
    ...config
  });
  const refreshGate = new RefreshGate(() => tokenApi.refresh());

  function handleUnauthorized() {
    tokenApi.clearToken();
    tokenApi.onAuthExpired?.();
  }

  axiosInstance.interceptors.request.use(
    (config: PureHttpRequestConfig): any => {
      const url = config.url ?? "";
      // 白名单（登录/刷新）不附加 token，也避免 401 后死循环
      if (tokenApi.isWhiteList(url)) return config;
      const token = tokenApi.getToken();
      if (token) {
        config.headers["Authorization"] = formatToken(token);
      }
      return config;
    },
    error => Promise.reject(error)
  );

  axiosInstance.interceptors.response.use(
    (response: PureHttpResponse) => {
      // 成功响应只解包一次：返回信封 data
      const $config = response.config as PureHttpRequestConfig;
      if (typeof $config.beforeResponseCallback === "function") {
        $config.beforeResponseCallback(response);
      }
      return response.data;
    },
    (error: PureHttpError) => {
      const $error = error;
      $error.isCancelRequest = Axios.isCancel($error);
      const config = $error.config as
        | (PureHttpRequestConfig & { _retry?: boolean })
        | undefined;
      if (!config) return Promise.reject($error);

      const url = config.url ?? "";
      const decision = decideAfter401(
        tokenApi.isWhiteList(url),
        config._retry === true
      );

      // 白名单（登录/刷新）：失败直接返回，由调用方处理
      if (decision === "skip") return Promise.reject($error);

      // 非 401 直接返回
      if ($error.response?.status !== 401) return Promise.reject($error);

      // 已经重试过一次仍然 401：清理认证状态，不再重试
      if (decision === "clear") {
        handleUnauthorized();
        return Promise.reject($error);
      }

      // 单飞刷新后，原请求只重试一次。注意：axiosInstance(config) 会再次经过
      // 同一响应拦截器，其成功分支已把响应解包，因此这里不再二次解包。
      // 使用两分支 then：onRejected 只处理 refresh 自身失败并清理认证；
      // 重试请求的失败（500/403/超时/断网，或再次 401 已在 clear 分支清理）
      // 会原样向上传递，不再次清理认证、不再次刷新。
      return refreshGate.acquire().then(
        token => {
          config._retry = true;
          config.headers["Authorization"] = formatToken(token);
          return axiosInstance(config) as Promise<unknown>;
        },
        refreshError => {
          // 仅 refresh 自身失败时才清理认证状态
          handleUnauthorized();
          return Promise.reject(refreshError ?? $error);
        }
      );
    }
  );

  return {
    request<T>(
      method: RequestMethods,
      url: string,
      param?: AxiosRequestConfig,
      axiosConfig?: PureHttpRequestConfig
    ): Promise<T> {
      const config = {
        method,
        url,
        ...param,
        ...axiosConfig
      } as PureHttpRequestConfig;
      return axiosInstance.request(config) as Promise<T>;
    }
  };
}

/** 应用的刷新实现（由 user store 注册，避免顶层循环依赖） */
let refreshHandler: (() => Promise<string>) | null = null;

export function registerRefreshHandler(fn: () => Promise<string>) {
  refreshHandler = fn;
}

/** 认证过期（刷新失败）回调，由 user store 注册 */
function setAuthExpiredHandler(fn: () => void) {
  sharedTokenApi.onAuthExpired = fn;
}
export { setAuthExpiredHandler };

const sharedTokenApi: HttpTokenApi = {
  getToken: () => getAccessToken(),
  clearToken: () => clearAccessToken(),
  refresh: () => {
    if (refreshHandler) return refreshHandler();
    // 冷启动兜底：动态引入 user store（避免顶层循环依赖）
    return import("@/store/modules/user").then(m =>
      m.useUserStoreHook().handRefreshToken()
    );
  },
  isWhiteList: url => isAuthWhiteList(url)
};

export const http = createHttpClient(sharedTokenApi);
