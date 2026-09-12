"use client";

import {
  createContext,
  useCallback,
  useContext,
  useMemo,
  useRef,
  useState,
} from "react";
import {
  api,
  clearAuthState,
  getAccessToken,
  refreshAccessToken,
  setAccessToken,
} from "./api";

export interface AuthUserLevel {
  id: string;
  code: string;
  name: string;
  mode: "auto" | "manual";
}

/** User profile returned by `/me` and register (contract §5.2). */
export interface AuthUser {
  id: string;
  username: string;
  email: string;
  nickname: string;
  avatar_url: string;
  status: "pending_verification" | "active" | "disabled";
  email_verified_at: string;
  last_login_at: string;
  points_balance: string;
  consumption_points: string;
  level: AuthUserLevel | null;
  created_at: string;
  updated_at: string;
}

export interface LoginPayload {
  identifier: string;
  password: string;
}

export interface RegisterPayload {
  username: string;
  email: string;
  password: string;
}

export interface UpdateProfilePayload {
  nickname?: string;
  avatar_url?: string;
}

export interface PointTransaction {
  id: string;
  points_delta: string;
  consumption_delta: string;
  balance_after: string;
  consumption_after: string;
  reason: string;
  actor_id: string | null;
  idempotency_key: string;
  created_at: string;
}

export interface PointTransactionPage {
  items: PointTransaction[];
  total: number;
  page: number;
  page_size: number;
}

interface TokenResponse {
  access_token: string;
  token_type: string;
  expires_in: number;
}

interface AuthContextValue {
  user: AuthUser | null;
  isAuthenticated: boolean;
  login: (payload: LoginPayload) => Promise<AuthUser | null>;
  register: (payload: RegisterPayload) => Promise<AuthUser>;
  verifyEmail: (email: string, token: string) => Promise<void>;
  resendVerification: (email: string) => Promise<void>;
  forgotPassword: (email: string) => Promise<void>;
  resetPassword: (
    email: string,
    token: string,
    newPassword: string,
  ) => Promise<void>;
  logout: () => Promise<void>;
  updateProfile: (payload: UpdateProfilePayload) => Promise<AuthUser>;
  changePassword: (currentPassword: string, newPassword: string) => Promise<void>;
  restoreSession: () => Promise<boolean>;
}

const AuthContext = createContext<AuthContextValue | null>(null);

/**
 * Client-side session store. The access token itself lives in `lib/api.ts`
 * memory only (never localStorage/sessionStorage/cookies); this provider only
 * mirrors the resolved `/me` profile for rendering.
 *
 * Session restore is deliberately browser-only: the refresh cookie is scoped to
 * `/api/v1/auth`, so SSR page requests never carry it (contract §2).
 */
export function AuthProvider({ children }: { children: React.ReactNode }) {
  const [user, setUser] = useState<AuthUser | null>(null);
  const restoreRef = useRef<Promise<boolean> | null>(null);

  const loadMe = useCallback(async (): Promise<AuthUser | null> => {
    if (!getAccessToken()) {
      setUser(null);
      return null;
    }
    try {
      const me = await api.get<AuthUser>("/v1/me");
      setUser(me);
      return me;
    } catch (error) {
      setUser(null);
      throw error;
    }
  }, []);

  const login = useCallback(
    async (payload: LoginPayload): Promise<AuthUser | null> => {
      const result = await api.post<TokenResponse>("/v1/auth/login", payload, {
        skipAuthRetry: true,
      });
      setAccessToken(result.access_token);
      return loadMe();
    },
    [loadMe],
  );

  const register = useCallback(
    (payload: RegisterPayload): Promise<AuthUser> =>
      api.post<AuthUser>("/v1/auth/register", payload, { skipAuthRetry: true }),
    [],
  );

  const verifyEmail = useCallback(
    async (email: string, token: string): Promise<void> => {
      await api.post(
        "/v1/auth/verify-email",
        { email, token },
        { skipAuthRetry: true },
      );
    },
    [],
  );

  const resendVerification = useCallback(
    async (email: string): Promise<void> => {
      await api.post(
        "/v1/auth/resend-verification",
        { email },
        { skipAuthRetry: true },
      );
    },
    [],
  );

  const forgotPassword = useCallback(
    async (email: string): Promise<void> => {
      await api.post(
        "/v1/auth/forgot-password",
        { email },
        { skipAuthRetry: true },
      );
    },
    [],
  );

  const resetPassword = useCallback(
    async (
      email: string,
      token: string,
      newPassword: string,
    ): Promise<void> => {
      await api.post(
        "/v1/auth/reset-password",
        { email, token, new_password: newPassword },
        { skipAuthRetry: true },
      );
    },
    [],
  );

  const logout = useCallback(async (): Promise<void> => {
    try {
      await api.post("/v1/auth/logout");
    } catch {
      // Local state is dropped regardless of the network outcome.
    } finally {
      clearAuthState();
      setUser(null);
    }
  }, []);

  const updateProfile = useCallback(
    async (payload: UpdateProfilePayload): Promise<AuthUser> => {
      const updated = await api.patch<AuthUser>("/v1/me", payload);
      setUser(updated);
      return updated;
    },
    [],
  );

  const changePassword = useCallback(
    async (currentPassword: string, newPassword: string): Promise<void> => {
      // Contract §5.4: success revokes every session, so drop all local state.
      await api.post("/v1/me/password", {
        current_password: currentPassword,
        new_password: newPassword,
      });
      clearAuthState();
      setUser(null);
    },
    [],
  );

  const restoreSession = useCallback(async (): Promise<boolean> => {
    if (getAccessToken()) {
      try {
        await loadMe();
        return true;
      } catch {
        clearAuthState();
        setUser(null);
        return false;
      }
    }
    if (!restoreRef.current) {
      restoreRef.current = (async () => {
        try {
          await refreshAccessToken();
          await loadMe();
          return true;
        } catch {
          clearAuthState();
          setUser(null);
          return false;
        } finally {
          restoreRef.current = null;
        }
      })();
    }
    return restoreRef.current;
  }, [loadMe]);

  const value = useMemo<AuthContextValue>(
    () => ({
      user,
      isAuthenticated: user !== null,
      login,
      register,
      verifyEmail,
      resendVerification,
      forgotPassword,
      resetPassword,
      logout,
      updateProfile,
      changePassword,
      restoreSession,
    }),
    [
      user,
      login,
      register,
      verifyEmail,
      resendVerification,
      forgotPassword,
      resetPassword,
      logout,
      updateProfile,
      changePassword,
      restoreSession,
    ],
  );

  return <AuthContext.Provider value={value}>{children}</AuthContext.Provider>;
}

export function useAuth(): AuthContextValue {
  const context = useContext(AuthContext);
  if (!context) {
    throw new Error("useAuth must be used within an AuthProvider");
  }
  return context;
}
