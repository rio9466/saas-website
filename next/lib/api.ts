import { ApiError, RETRYABLE_UNAUTHORIZED_CODES } from "./api-error";

/**
 * Contract-aware API client (contract §7.2).
 *
 * - Browser requests go to the same-origin `/api/**` proxy; SSR requests go
 *   straight to `API_INTERNAL_BASE`.
 * - The envelope is unwrapped, and `code !== 0` throws an `ApiError`.
 * - A 401 with a retryable code triggers exactly one refresh + replay; the
 *   refresh call itself is never retried.
 *
 * The access token lives in memory only (never localStorage/sessionStorage),
 * so it resets on reload and is only ever set by the browser-side refresh.
 */

const API_BASE = "/api";
const DEFAULT_INTERNAL_BASE = "http://127.0.0.1:8080";

export type ApiQueryValue = string | number | boolean | null | undefined;

export interface ApiRequestOptions {
  method?: "GET" | "POST" | "PUT" | "PATCH" | "DELETE";
  query?: Record<string, ApiQueryValue>;
  body?: unknown;
  headers?: Record<string, string>;
  /** Current UI locale, sent as Accept-Language (content locale uses ?locale=). */
  locale?: string;
  signal?: AbortSignal;
  /** Internal marker so a 401 is only retried once. */
  retried?: boolean;
  /** Public auth endpoints (login/refresh) must not refresh + replay on 401. */
  skipAuthRetry?: boolean;
}

interface ApiEnvelope<T> {
  code: number;
  message: string;
  data: T;
  request_id: string;
}

function isServer(): boolean {
  return typeof window === "undefined";
}

let accessToken: string | null = null;

/** Store the in-memory access token. Never call with a persisted value. */
export function setAccessToken(token: string | null): void {
  accessToken = token;
}

export function getAccessToken(): string | null {
  return accessToken;
}

/** Drop the in-memory session (token only; the refresh cookie is HttpOnly). */
export function clearAuthState(): void {
  accessToken = null;
}

function createRequestId(): string {
  if (typeof globalThis.crypto?.randomUUID === "function") {
    return globalThis.crypto.randomUUID();
  }
  return `${Date.now().toString(36)}-${Math.random().toString(36).slice(2, 10)}`;
}

function buildQuery(query?: Record<string, ApiQueryValue>): string {
  if (!query) {
    return "";
  }
  const params = new URLSearchParams();
  for (const [key, value] of Object.entries(query)) {
    if (value === null || value === undefined) {
      continue;
    }
    params.set(key, String(value));
  }
  return params.toString();
}

function resolveUrl(
  path: string,
  query?: Record<string, ApiQueryValue>,
): string {
  const relative = path.startsWith("/") ? path : `/${path}`;
  const base = isServer()
    ? `${(process.env.API_INTERNAL_BASE || DEFAULT_INTERNAL_BASE).replace(/\/+$/, "")}${API_BASE}`
    : API_BASE;
  const search = buildQuery(query);
  return `${base}${relative}${search ? `?${search}` : ""}`;
}

async function execute<T>(
  path: string,
  options: ApiRequestOptions,
): Promise<T> {
  const headers: Record<string, string> = {
    Accept: "application/json",
    "X-Request-ID": createRequestId(),
    ...options.headers,
  };
  if (accessToken) {
    headers.Authorization = `Bearer ${accessToken}`;
  }
  if (options.body !== undefined) {
    headers["Content-Type"] = "application/json";
  }
  if (options.locale) {
    headers["Accept-Language"] = options.locale;
  }

  let response: Response;
  try {
    response = await fetch(resolveUrl(path, options.query), {
      method: options.method ?? "GET",
      headers,
      body: options.body === undefined ? undefined : JSON.stringify(options.body),
      // Send/stash the HttpOnly refresh cookie in the browser only.
      credentials: isServer() ? "omit" : "include",
      signal: options.signal,
    });
  } catch (error) {
    throw new ApiError(
      0,
      -1,
      error instanceof Error ? error.message : "Network request failed",
    );
  }

  let envelope: ApiEnvelope<T> | null = null;
  try {
    envelope = (await response.json()) as ApiEnvelope<T>;
  } catch {
    envelope = null;
  }

  if (!response.ok || !envelope || envelope.code !== 0) {
    throw new ApiError(
      response.status,
      envelope?.code ?? -1,
      envelope?.message ?? response.statusText,
      envelope?.request_id ?? response.headers.get("x-request-id") ?? "",
    );
  }

  return envelope.data;
}

let refreshRequest: Promise<string> | null = null;

/**
 * Exchanges the HttpOnly refresh cookie for a new access token. Refresh is a
 * browser-only concern: SSR has no access to the cookie.
 */
export async function refreshAccessToken(): Promise<string> {
  if (isServer()) {
    throw new ApiError(401, 20001, "Refresh is not available during SSR");
  }
  if (refreshRequest) {
    return refreshRequest;
  }

  refreshRequest = (async () => {
    const response = await fetch(`${API_BASE}/v1/auth/refresh`, {
      method: "POST",
      credentials: "include",
      headers: { "X-Request-ID": createRequestId() },
    });
    const envelope = (await response.json()) as ApiEnvelope<{
      access_token: string;
    }>;
    if (!response.ok || envelope.code !== 0 || !envelope.data?.access_token) {
      throw new ApiError(
        response.status,
        envelope?.code ?? -1,
        envelope?.message ?? "Refresh failed",
        envelope?.request_id ?? "",
      );
    }
    accessToken = envelope.data.access_token;
    return accessToken;
  })();

  try {
    return await refreshRequest;
  } finally {
    refreshRequest = null;
  }
}

export async function apiFetch<T>(
  path: string,
  options: ApiRequestOptions = {},
): Promise<T> {
  try {
    return await execute<T>(path, options);
  } catch (error) {
    const shouldRetry =
      error instanceof ApiError &&
      error.status === 401 &&
      RETRYABLE_UNAUTHORIZED_CODES.has(error.code) &&
      !isServer() &&
      !options.retried &&
      !options.skipAuthRetry;

    if (!shouldRetry) {
      throw error;
    }

    try {
      await refreshAccessToken();
    } catch {
      clearAuthState();
      throw error;
    }
    return apiFetch<T>(path, { ...options, retried: true });
  }
}

export const api = {
  get: <T>(path: string, options?: Omit<ApiRequestOptions, "method" | "body">) =>
    apiFetch<T>(path, { ...options, method: "GET" }),
  post: <T>(
    path: string,
    body?: unknown,
    options?: Omit<ApiRequestOptions, "method" | "body">,
  ) => apiFetch<T>(path, { ...options, method: "POST", body }),
  put: <T>(
    path: string,
    body?: unknown,
    options?: Omit<ApiRequestOptions, "method" | "body">,
  ) => apiFetch<T>(path, { ...options, method: "PUT", body }),
  patch: <T>(
    path: string,
    body?: unknown,
    options?: Omit<ApiRequestOptions, "method" | "body">,
  ) => apiFetch<T>(path, { ...options, method: "PATCH", body }),
  del: <T>(path: string, options?: Omit<ApiRequestOptions, "method" | "body">) =>
    apiFetch<T>(path, { ...options, method: "DELETE" }),
};
