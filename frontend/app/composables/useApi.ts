import type { AuthUser } from './useAuth'

export type ApiQueryValue = string | number | boolean | null | undefined

export interface ApiRequestOptions {
  method?: 'GET' | 'POST' | 'PUT' | 'PATCH' | 'DELETE'
  query?: Record<string, ApiQueryValue>
  body?: unknown
  headers?: Record<string, string>
  /** Current UI locale, sent as Accept-Language (content locale uses ?locale=). */
  locale?: string
  /** Internal marker so a 401 is only retried once. */
  retried?: boolean
}

interface ApiEnvelope<T> {
  code: number
  message: string
  data: T
  request_id: string
}

/** 401 business codes that trigger a single refresh + replay (contract §2). */
const RETRYABLE_UNAUTHORIZED_CODES = new Set([20001, 20002, 20003, 40006])

export function useAccessToken() {
  // Access token lives in memory only; never persisted to storage or cookies.
  return useState<string | null>('auth.accessToken', () => null)
}

export function useAuthUser() {
  return useState<AuthUser | null>('auth.user', () => null)
}

export function clearAuthState() {
  useAccessToken().value = null
  useAuthUser().value = null
}

function createRequestId(): string {
  if (typeof globalThis.crypto?.randomUUID === 'function') {
    return globalThis.crypto.randomUUID()
  }
  return `${Date.now().toString(36)}-${Math.random().toString(36).slice(2, 10)}`
}

function toApiError(error: unknown): ApiError {
  if (error instanceof ApiError) {
    return error
  }
  const fetchError = error as {
    status?: number
    message?: string
    data?: Partial<ApiEnvelope<unknown>>
    response?: { status?: number, headers?: Headers }
  }
  const status = fetchError.response?.status ?? fetchError.status ?? 0
  const envelope = fetchError.data
  const code = typeof envelope?.code === 'number' ? envelope.code : -1
  const message = typeof envelope?.message === 'string' ? envelope.message : (fetchError.message ?? 'Request failed')
  const requestId = envelope?.request_id ?? fetchError.response?.headers?.get('x-request-id') ?? ''
  return new ApiError(status, code, message, requestId)
}

let refreshRequest: Promise<string | null> | null = null

/** Refreshes the in-memory access token via the HttpOnly refresh cookie. */
export async function refreshAccessToken(): Promise<string | null> {
  // Refresh cookies are only forwarded by the browser; SSR never refreshes.
  if (import.meta.server) {
    return null
  }
  if (refreshRequest) {
    return refreshRequest
  }
  const token = useAccessToken()
  const config = useRuntimeConfig()
  refreshRequest = (async () => {
    try {
      const envelope = await $fetch<ApiEnvelope<{ access_token: string }>>(
        `${config.public.apiBase}/v1/auth/refresh`,
        {
          method: 'POST',
          credentials: 'include',
          headers: { 'X-Request-ID': createRequestId() }
        }
      )
      if (envelope.code !== 0) {
        throw new ApiError(200, envelope.code, envelope.message, envelope.request_id)
      }
      token.value = envelope.data.access_token
      return token.value
    } finally {
      refreshRequest = null
    }
  })()
  return refreshRequest
}

/**
 * Contract-aware API client. Paths are relative to `runtimeConfig.public.apiBase`
 * (for example `/v1/public/settings`), so the browser only ever calls the
 * same-origin `/api/**` proxy while SSR uses the internal backend address.
 */
export function useApi() {
  const token = useAccessToken()
  const config = useRuntimeConfig()
  const apiBase = config.public.apiBase

  function resolveUrl(path: string): string {
    const relative = path.startsWith('/') ? path : `/${path}`
    if (import.meta.server) {
      // `apiInternalBase` is a server-only runtime key; the browser always uses
      // the same-origin `/api/**` proxy below.
      const internalBase = String(config.apiInternalBase || '').replace(/\/$/, '')
      return `${internalBase}${apiBase}${relative}`
    }
    return `${apiBase}${relative}`
  }

  async function execute<T>(path: string, options: ApiRequestOptions): Promise<T> {
    const headers: Record<string, string> = {
      'Accept': 'application/json',
      'X-Request-ID': createRequestId(),
      ...options.headers
    }
    if (token.value) {
      headers.Authorization = `Bearer ${token.value}`
    }
    if (options.body !== undefined) {
      headers['Content-Type'] = 'application/json'
    }
    if (options.locale) {
      headers['Accept-Language'] = options.locale
    }

    let envelope: ApiEnvelope<T>
    try {
      envelope = await $fetch<ApiEnvelope<T>>(resolveUrl(path), {
        method: options.method ?? 'GET',
        query: options.query,
        body: options.body as Record<string, unknown> | undefined,
        headers,
        credentials: import.meta.client ? 'include' : 'omit'
      })
    } catch (error) {
      throw toApiError(error)
    }

    if (envelope.code !== 0) {
      throw new ApiError(200, envelope.code, envelope.message, envelope.request_id)
    }
    return envelope.data
  }

  async function request<T>(path: string, options: ApiRequestOptions = {}): Promise<T> {
    try {
      return await execute<T>(path, options)
    } catch (error) {
      if (
        error instanceof ApiError
        && error.status === 401
        && RETRYABLE_UNAUTHORIZED_CODES.has(error.code)
        && import.meta.client
        && !options.retried
      ) {
        try {
          await refreshAccessToken()
        } catch {
          clearAuthState()
          throw error
        }
        return request<T>(path, { ...options, retried: true })
      }
      throw error
    }
  }

  return {
    request,
    get: <T>(path: string, options?: Omit<ApiRequestOptions, 'method' | 'body'>) =>
      request<T>(path, { ...options, method: 'GET' }),
    post: <T>(path: string, body?: unknown, options?: Omit<ApiRequestOptions, 'method' | 'body'>) =>
      request<T>(path, { ...options, method: 'POST', body }),
    put: <T>(path: string, body?: unknown, options?: Omit<ApiRequestOptions, 'method' | 'body'>) =>
      request<T>(path, { ...options, method: 'PUT', body }),
    patch: <T>(path: string, body?: unknown, options?: Omit<ApiRequestOptions, 'method' | 'body'>) =>
      request<T>(path, { ...options, method: 'PATCH', body }),
    del: <T>(path: string, options?: Omit<ApiRequestOptions, 'method' | 'body'>) =>
      request<T>(path, { ...options, method: 'DELETE' })
  }
}
