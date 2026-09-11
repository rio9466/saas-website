package handler

import (
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/rio9466/easy-admin/server/internal/domain/apperr"
	platformauth "github.com/rio9466/easy-admin/server/internal/platform/auth"
	"github.com/rio9466/easy-admin/server/internal/transport/http/middleware"
	"github.com/rio9466/easy-admin/server/internal/transport/http/response"
)

// AuthHandlers serves administrator authentication and profile endpoints.
type AuthHandlers struct {
	svc            AdminAuthService
	cookie         AuthCookieSettings
	trustedOrigins []string
}

// NewAuthHandlers constructs auth handlers.
func NewAuthHandlers(svc AdminAuthService, cookie AuthCookieSettings, trustedOrigins []string) *AuthHandlers {
	return &AuthHandlers{
		svc:            svc,
		cookie:         cookie.normalized(),
		trustedOrigins: append([]string(nil), trustedOrigins...),
	}
}

// Login authenticates credentials and sets the refresh cookie.
func (h *AuthHandlers) Login(c *gin.Context) {
	if !h.requireTrustedOrigin(c) {
		return
	}
	var req loginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, apperr.CodeValidation, "validation failed")
		return
	}
	req.Username = strings.TrimSpace(req.Username)
	if req.Username == "" || req.Password == "" {
		response.Error(c, http.StatusBadRequest, apperr.CodeValidation, "validation failed")
		return
	}

	result, err := h.svc.Login(c.Request.Context(), req.Username, req.Password, anonymousActor(c))
	if err != nil {
		middleware.WriteAppError(c, err)
		return
	}
	h.setRefreshCookie(c, result)
	response.OK(c, accessTokenPayload(result))
}

// Refresh rotates the refresh cookie and returns a new access token.
func (h *AuthHandlers) Refresh(c *gin.Context) {
	if !h.requireTrustedOrigin(c) {
		return
	}
	raw, err := c.Cookie(h.cookie.Name)
	if err != nil || strings.TrimSpace(raw) == "" {
		response.Error(c, http.StatusUnauthorized, apperr.CodeUnauthorized, "unauthorized")
		return
	}

	result, err := h.svc.Refresh(c.Request.Context(), raw, anonymousActor(c))
	if err != nil {
		h.clearRefreshCookie(c)
		middleware.WriteAppError(c, err)
		return
	}
	h.setRefreshCookie(c, result)
	response.OK(c, accessTokenPayload(result))
}

// Logout revokes the current session and clears the refresh cookie.
func (h *AuthHandlers) Logout(c *gin.Context) {
	if !h.requireTrustedOrigin(c) {
		return
	}
	actor := actorFromContext(c)
	if err := h.svc.Logout(c.Request.Context(), actor); err != nil {
		middleware.WriteAppError(c, err)
		return
	}
	h.clearRefreshCookie(c)
	response.OK(c, map[string]any{})
}

// Me returns the current administrator profile and effective permissions.
func (h *AuthHandlers) Me(c *gin.Context) {
	result, err := h.svc.Me(c.Request.Context(), actorFromContext(c))
	if err != nil {
		middleware.WriteAppError(c, err)
		return
	}
	response.OK(c, toMeData(result))
}

// UpdateMe updates the current administrator's display name. No admin.user.update
// permission is required; only the authenticated account's display_name can change.
func (h *AuthHandlers) UpdateMe(c *gin.Context) {
	var req updateMeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, apperr.CodeValidation, "validation failed")
		return
	}
	if req.DisplayName == nil || strings.TrimSpace(*req.DisplayName) == "" {
		response.Error(c, http.StatusBadRequest, apperr.CodeValidation, "validation failed")
		return
	}

	result, err := h.svc.UpdateMyProfile(c.Request.Context(), actorFromContext(c), *req.DisplayName)
	if err != nil {
		middleware.WriteAppError(c, err)
		return
	}
	response.OK(c, toMeData(result))
}

// ChangePassword updates the current password and revokes the session.
func (h *AuthHandlers) ChangePassword(c *gin.Context) {
	var req changePasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, apperr.CodeValidation, "validation failed")
		return
	}
	if req.CurrentPassword == "" || req.NewPassword == "" {
		response.Error(c, http.StatusBadRequest, apperr.CodeValidation, "validation failed")
		return
	}

	if err := h.svc.ChangePassword(c.Request.Context(), actorFromContext(c), req.CurrentPassword, req.NewPassword); err != nil {
		middleware.WriteAppError(c, err)
		return
	}
	h.clearRefreshCookie(c)
	response.OK(c, map[string]any{})
}

func (h *AuthHandlers) requireTrustedOrigin(c *gin.Context) bool {
	if err := platformauth.ValidateTrustedOrigin(c.Request, h.trustedOrigins); err != nil {
		response.Error(c, http.StatusForbidden, apperr.CodeForbidden, "forbidden")
		return false
	}
	return true
}

func (h *AuthHandlers) setRefreshCookie(c *gin.Context, result *LoginResult) {
	cfg := h.cookie.normalized()
	token := ""
	if result != nil {
		token = result.RefreshToken
	}
	// Max-Age reflects the remaining time until the session's absolute refresh
	// expiry; rotated sessions never extend past the original login expiry.
	maxAge := int(platformauth.RefreshTokenTTL.Seconds())
	if result != nil && !result.RefreshExpiresAt.IsZero() {
		if remain := time.Until(result.RefreshExpiresAt.UTC()); remain > 0 {
			if seconds := int(remain.Seconds()); seconds > 0 {
				maxAge = seconds
			}
		}
	}
	http.SetCookie(c.Writer, &http.Cookie{
		Name:     cfg.Name,
		Value:    token,
		Path:     cfg.Path,
		MaxAge:   maxAge,
		HttpOnly: true,
		Secure:   cfg.Secure,
		SameSite: http.SameSiteStrictMode,
	})
}

func (h *AuthHandlers) clearRefreshCookie(c *gin.Context) {
	cfg := h.cookie.normalized()
	http.SetCookie(c.Writer, &http.Cookie{
		Name:     cfg.Name,
		Value:    "",
		Path:     cfg.Path,
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   cfg.Secure,
		SameSite: http.SameSiteStrictMode,
	})
}

func accessTokenPayload(result *LoginResult) accessTokenData {
	expiresIn := int64(platformauth.AccessTokenTTL.Seconds())
	if result != nil && result.ExpiresIn > 0 {
		expiresIn = result.ExpiresIn
	}
	token := ""
	if result != nil {
		token = result.AccessToken
	}
	return accessTokenData{
		AccessToken: token,
		TokenType:   "Bearer",
		ExpiresIn:   expiresIn,
	}
}

func actorFromContext(c *gin.Context) Actor {
	return Actor{
		ID:          middleware.AdminIDFromContext(c),
		Username:    middleware.UsernameFromContext(c),
		DisplayName: middleware.DisplayNameFromContext(c),
		RoleCodes:   middleware.RolesFromContext(c),
		SessionID:   middleware.SessionIDFromContext(c),
		RequestID:   middleware.RequestIDFromContext(c),
		SourceIP:    c.ClientIP(),
		UserAgent:   c.Request.UserAgent(),
	}
}

func anonymousActor(c *gin.Context) Actor {
	return Actor{
		RequestID: middleware.RequestIDFromContext(c),
		SourceIP:  c.ClientIP(),
		UserAgent: c.Request.UserAgent(),
	}
}
