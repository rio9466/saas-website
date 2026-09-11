import { useAccessToken, useApi, useAuthUser, clearAuthState, refreshAccessToken } from './useApi'

export interface AuthUserLevel {
  id: string
  code: string
  name: string
  mode: 'auto' | 'manual'
}

export interface AuthUser {
  id: string
  username: string
  email: string
  nickname: string
  avatar_url: string
  status: 'pending_verification' | 'active' | 'disabled'
  email_verified_at: string
  last_login_at: string
  points_balance: string
  consumption_points: string
  level: AuthUserLevel | null
  created_at: string
  updated_at: string
}

export interface LoginPayload {
  identifier: string
  password: string
}

export interface RegisterPayload {
  username: string
  email: string
  password: string
}

export interface UpdateProfilePayload {
  nickname?: string
  avatar_url?: string
}

export interface PointTransaction {
  id: string
  points_delta: string
  consumption_delta: string
  balance_after: string
  consumption_after: string
  reason: string
  actor_id: string | null
  idempotency_key: string
  created_at: string
}

export interface PointTransactionPage {
  items: PointTransaction[]
  total: number
  page: number
  page_size: number
}

interface TokenResponse {
  access_token: string
  token_type: string
  expires_in: number
}

let restoreRequest: Promise<boolean> | null = null

export function useAuth() {
  const api = useApi()
  const token = useAccessToken()
  const user = useAuthUser()

  const isAuthenticated = computed(() => Boolean(token.value))

  async function login(payload: LoginPayload): Promise<AuthUser | null> {
    const result = await api.post<TokenResponse>('/v1/auth/login', payload, { skipAuthRetry: true })
    token.value = result.access_token
    return await loadMe()
  }

  async function register(payload: RegisterPayload): Promise<AuthUser> {
    return await api.post<AuthUser>('/v1/auth/register', payload, { skipAuthRetry: true })
  }

  async function verifyEmail(email: string, tokenValue: string): Promise<void> {
    await api.post('/v1/auth/verify-email', { email, token: tokenValue }, { skipAuthRetry: true })
  }

  async function resendVerification(email: string): Promise<void> {
    await api.post('/v1/auth/resend-verification', { email }, { skipAuthRetry: true })
  }

  async function forgotPassword(email: string): Promise<void> {
    await api.post('/v1/auth/forgot-password', { email }, { skipAuthRetry: true })
  }

  async function resetPassword(email: string, tokenValue: string, newPassword: string): Promise<void> {
    await api.post(
      '/v1/auth/reset-password',
      { email, token: tokenValue, new_password: newPassword },
      { skipAuthRetry: true }
    )
  }

  async function loadMe(): Promise<AuthUser | null> {
    if (!token.value) {
      user.value = null
      return null
    }
    try {
      user.value = await api.get<AuthUser>('/v1/me')
      return user.value
    } catch (error) {
      user.value = null
      throw error
    }
  }

  async function updateProfile(payload: UpdateProfilePayload): Promise<AuthUser> {
    const updated = await api.patch<AuthUser>('/v1/me', payload)
    user.value = updated
    return updated
  }

  async function changePassword(currentPassword: string, newPassword: string): Promise<void> {
    // Contract §5.4: success revokes every session, so drop all local state.
    await api.post('/v1/me/password', {
      current_password: currentPassword,
      new_password: newPassword
    })
    clearAuthState()
  }

  /**
   * Restores the session from the HttpOnly refresh cookie. Contract §2: the
   * refresh cookie is scoped to `/api/v1/auth`, so SSR requests to page routes
   * never carry it — restore must happen in the browser after hydration.
   */
  async function restoreSession(): Promise<boolean> {
    if (import.meta.server) {
      return false
    }
    if (token.value) {
      if (!user.value) {
        await loadMe().catch(() => clearAuthState())
      }
      return Boolean(token.value)
    }
    if (!restoreRequest) {
      restoreRequest = (async () => {
        try {
          const next = await refreshAccessToken()
          if (!next) {
            clearAuthState()
            return false
          }
          await loadMe()
          return true
        } catch {
          clearAuthState()
          return false
        } finally {
          restoreRequest = null
        }
      })()
    }
    return restoreRequest
  }

  async function logout(): Promise<void> {
    try {
      await api.post('/v1/auth/logout')
    } finally {
      clearAuthState()
    }
  }

  async function refresh(): Promise<string | null> {
    const nextToken = await refreshAccessToken()
    if (nextToken) {
      await loadMe()
    }
    return nextToken
  }

  return {
    token,
    user,
    isAuthenticated,
    login,
    register,
    verifyEmail,
    resendVerification,
    forgotPassword,
    resetPassword,
    loadMe,
    updateProfile,
    changePassword,
    restoreSession,
    logout,
    refresh
  }
}
