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

interface TokenResponse {
  access_token: string
  token_type: string
  expires_in: number
}

export function useAuth() {
  const api = useApi()
  const token = useAccessToken()
  const user = useAuthUser()

  const isAuthenticated = computed(() => Boolean(token.value))

  async function login(payload: LoginPayload): Promise<AuthUser | null> {
    const result = await api.post<TokenResponse>('/v1/auth/login', payload)
    token.value = result.access_token
    return await loadMe()
  }

  async function register(payload: RegisterPayload): Promise<AuthUser> {
    return await api.post<AuthUser>('/v1/auth/register', payload)
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
    loadMe,
    logout,
    refresh
  }
}
