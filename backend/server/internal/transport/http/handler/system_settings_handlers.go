package handler

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/rio9466/easy-admin/server/internal/domain/apperr"
	userdomain "github.com/rio9466/easy-admin/server/internal/domain/user"
	"github.com/rio9466/easy-admin/server/internal/transport/http/middleware"
	"github.com/rio9466/easy-admin/server/internal/transport/http/response"
)

// SystemSettingsHandlers serves the typed system-settings singleton.
type SystemSettingsHandlers struct {
	svc UserAdminService
}

// NewSystemSettingsHandlers constructs settings handlers.
func NewSystemSettingsHandlers(svc UserAdminService) *SystemSettingsHandlers {
	return &SystemSettingsHandlers{svc: svc}
}

// Get returns the typed settings. SMTP password is only a configured flag.
func (h *SystemSettingsHandlers) Get(c *gin.Context) {
	settings, err := h.svc.GetSystemSettings(c.Request.Context(), actorFromContext(c))
	if err != nil {
		middleware.WriteAppError(c, err)
		return
	}
	response.OK(c, toSystemSettingsData(settings))
}

// Update writes the singleton with optimistic locking.
func (h *SystemSettingsHandlers) Update(c *gin.Context) {
	var req updateSystemSettingsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, apperr.CodeValidation, "validation failed")
		return
	}

	defaultLevelID, err := parseID(req.DefaultLevelID)
	if err != nil || defaultLevelID <= 0 {
		response.Error(c, http.StatusBadRequest, apperr.CodeValidation, "validation failed")
		return
	}
	registrationPoints, err := userdomain.ParseDecimal4(req.RegistrationPoints)
	if err != nil {
		response.Error(c, http.StatusBadRequest, apperr.CodeValidation, "validation failed")
		return
	}

	settings, err := h.svc.UpdateSystemSettings(c.Request.Context(), actorFromContext(c), UpdateSystemSettingsInput{
		PlatformName:              strings.TrimSpace(req.PlatformName),
		PublicFrontendURL:         strings.TrimSpace(req.PublicFrontendURL),
		PublicAPIURL:              strings.TrimSpace(req.PublicAPIURL),
		RegistrationEnabled:       req.RegistrationEnabled,
		UsernameLoginEnabled:      req.UsernameLoginEnabled,
		EmailLoginEnabled:         req.EmailLoginEnabled,
		EmailVerificationRequired: req.EmailVerificationRequired,
		DefaultLevelID:            defaultLevelID,
		DefaultAvatarURL:          strings.TrimSpace(req.DefaultAvatarURL),
		RegistrationPoints:        registrationPoints,
		SMTPEnabled:               req.SMTPEnabled,
		SMTPHost:                  strings.TrimSpace(req.SMTPHost),
		SMTPPort:                  req.SMTPPort,
		SMTPUsername:              strings.TrimSpace(req.SMTPUsername),
		SMTPPassword:              req.SMTPPassword,
		SMTPFromEmail:             strings.TrimSpace(req.SMTPFromEmail),
		SMTPFromName:              strings.TrimSpace(req.SMTPFromName),
		SMTPTLSMode:               strings.TrimSpace(req.SMTPTLSMode),
		Version:                   req.Version,
	})
	if err != nil {
		middleware.WriteAppError(c, err)
		return
	}
	response.OK(c, toSystemSettingsData(settings))
}
