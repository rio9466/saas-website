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

const (
	defaultUserRefreshCookieName = "ea_user_refresh"
	defaultUserRefreshCookiePath = "/api/v1/auth"
)

// UserCookieSettings controls the business-user HttpOnly refresh cookie.
type UserCookieSettings struct {
	Name   string
	Path   string
	Secure bool
}

func (s UserCookieSettings) normalized() UserCookieSettings {
	if s.Name == "" {
		s.Name = defaultUserRefreshCookieName
	}
	if s.Path == "" {
		s.Path = defaultUserRefreshCookiePath
	}
	return s
}

// UserAuthHandlers serves public settings and business-user authentication.
type UserAuthHandlers struct {
	svc            UserClientService
	cookie         UserCookieSettings
	trustedOrigins []string
}

// NewUserAuthHandlers constructs user auth handlers.
func NewUserAuthHandlers(svc UserClientService, cookie UserCookieSettings, trustedOrigins []string) *UserAuthHandlers {
	return &UserAuthHandlers{
		svc:            svc,
		cookie:         cookie.normalized(),
		trustedOrigins: append([]string(nil), trustedOrigins...),
	}
}

// PublicSettings returns the safe settings whitelist.
func (h *UserAuthHandlers) PublicSettings(c *gin.Context) {
	result, err := h.svc.GetPublicSettings(c.Request.Context())
	if err != nil {
		middleware.WriteAppError(c, err)
		return
	}
	data := toPublicSettingsData(result)
	response.OK(c, data)
}

// Register creates a business user account.
func (h *UserAuthHandlers) Register(c *gin.Context) {
	var req userRegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, apperr.CodeValidation, "validation failed")
		return
	}
	req.Username = strings.TrimSpace(req.Username)
	req.Email = strings.TrimSpace(req.Email)
	if req.Username == "" || req.Email == "" || req.Password == "" {
		response.Error(c, http.StatusBadRequest, apperr.CodeValidation, "validation failed")
		return
	}

	user, err := h.svc.Register(c.Request.Context(), anonymousActor(c), UserRegisterInput{
		Username: req.Username,
		Email:    req.Email,
		Password: req.Password,
	})
	if err != nil {
		middleware.WriteAppError(c, err)
		return
	}
	response.JSON(c, http.StatusCreated, toUserMeData(user))
}

// VerifyEmail redeems a one-time verification token.
func (h *UserAuthHandlers) VerifyEmail(c *gin.Context) {
	var req userVerifyEmailRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, apperr.CodeValidation, "validation failed")
		return
	}
	if strings.TrimSpace(req.Email) == "" || strings.TrimSpace(req.Token) == "" {
		response.Error(c, http.StatusBadRequest, apperr.CodeValidation, "validation failed")
		return
	}
	if err := h.svc.VerifyEmail(c.Request.Context(), anonymousActor(c), req.Email, req.Token); err != nil {
		middleware.WriteAppError(c, err)
		return
	}
	response.OK(c, map[string]any{"email_verified": true})
}

// ResendVerification re-mints and re-sends a verification token.
func (h *UserAuthHandlers) ResendVerification(c *gin.Context) {
	var req userResendVerificationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, apperr.CodeValidation, "validation failed")
		return
	}
	if strings.TrimSpace(req.Email) == "" {
		response.Error(c, http.StatusBadRequest, apperr.CodeValidation, "validation failed")
		return
	}
	if err := h.svc.ResendVerification(c.Request.Context(), anonymousActor(c), req.Email); err != nil {
		middleware.WriteAppError(c, err)
		return
	}
	response.OK(c, map[string]any{"resent": true})
}

// Login authenticates with identifier + password and sets the user refresh cookie.
func (h *UserAuthHandlers) Login(c *gin.Context) {
	if !h.requireTrustedOrigin(c) {
		return
	}
	var req userLoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, apperr.CodeValidation, "validation failed")
		return
	}
	req.Identifier = strings.TrimSpace(req.Identifier)
	if req.Identifier == "" || req.Password == "" {
		response.Error(c, http.StatusBadRequest, apperr.CodeValidation, "validation failed")
		return
	}

	actor := anonymousActor(c)
	result, _, err := h.svc.Login(c.Request.Context(), actor, req.Identifier, req.Password, c.ClientIP())
	if err != nil {
		middleware.WriteAppError(c, err)
		return
	}
	h.setUserRefreshCookie(c, result)
	response.OK(c, mapUserLoginWire(result))
}

// Refresh rotates the user refresh cookie and returns a new access token.
func (h *UserAuthHandlers) Refresh(c *gin.Context) {
	if !h.requireTrustedOrigin(c) {
		return
	}
	raw, err := c.Cookie(h.cookie.Name)
	if err != nil || strings.TrimSpace(raw) == "" {
		response.Error(c, http.StatusUnauthorized, apperr.CodeUnauthorized, "unauthorized")
		return
	}
	result, _, err := h.svc.Refresh(c.Request.Context(), anonymousActor(c), raw)
	if err != nil {
		h.clearUserRefreshCookie(c)
		middleware.WriteAppError(c, err)
		return
	}
	h.setUserRefreshCookie(c, result)
	response.OK(c, mapUserLoginWire(result))
}

// Logout revokes the current user session and clears the refresh cookie.
func (h *UserAuthHandlers) Logout(c *gin.Context) {
	if !h.requireTrustedOrigin(c) {
		return
	}
	actor := userActorFromContext(c)
	if err := h.svc.Logout(c.Request.Context(), actor); err != nil {
		middleware.WriteAppError(c, err)
		return
	}
	h.clearUserRefreshCookie(c)
	response.OK(c, map[string]any{})
}

// Me returns the authenticated business-user profile.
func (h *UserAuthHandlers) Me(c *gin.Context) {
	actor := userActorFromContext(c)
	user, err := h.svc.Me(c.Request.Context(), actor, 0)
	if err != nil {
		middleware.WriteAppError(c, err)
		return
	}
	response.OK(c, toUserMeData(user))
}

func (h *UserAuthHandlers) requireTrustedOrigin(c *gin.Context) bool {
	if err := platformauth.ValidateTrustedOrigin(c.Request, h.trustedOrigins); err != nil {
		response.Error(c, http.StatusForbidden, apperr.CodeForbidden, "forbidden")
		return false
	}
	return true
}

func (h *UserAuthHandlers) setUserRefreshCookie(c *gin.Context, result *UserLoginResult) {
	cfg := h.cookie.normalized()
	token := ""
	if result != nil {
		token = result.RefreshToken
	}
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

func (h *UserAuthHandlers) clearUserRefreshCookie(c *gin.Context) {
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

func mapUserLoginWire(result *UserLoginResult) userLoginData {
	expiresIn := int64(platformauth.AccessTokenTTL.Seconds())
	token := ""
	if result != nil {
		if result.ExpiresIn > 0 {
			expiresIn = result.ExpiresIn
		}
		token = result.AccessToken
	}
	return userLoginData{
		AccessToken: token,
		TokenType:   "Bearer",
		ExpiresIn:   expiresIn,
	}
}

// userActorFromContext builds the business-user actor for /me and logout.
func userActorFromContext(c *gin.Context) Actor {
	return Actor{
		ID:        middleware.UserIDFromContext(c),
		Username:  middleware.UserUsernameFromContext(c),
		Email:     middleware.UserEmailFromContext(c),
		Nickname:  middleware.UserNicknameFromContext(c),
		SessionID: middleware.UserSessionIDFromContext(c),
		RequestID: middleware.RequestIDFromContext(c),
		SourceIP:  c.ClientIP(),
		UserAgent: c.Request.UserAgent(),
	}
}
