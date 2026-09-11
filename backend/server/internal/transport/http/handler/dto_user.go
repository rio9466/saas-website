package handler

import (
	"strconv"
	"time"

	userdomain "github.com/rio9466/easy-admin/server/internal/domain/user"
)

// --- business-user request DTOs ---

type userRegisterRequest struct {
	Username string `json:"username"`
	Email    string `json:"email"`
	Password string `json:"password"`
}

type userVerifyEmailRequest struct {
	Email string `json:"email"`
	Token string `json:"token"`
}

type userResendVerificationRequest struct {
	Email string `json:"email"`
}

type userLoginRequest struct {
	Identifier string `json:"identifier"`
	Password   string `json:"password"`
}

type createBusinessUserRequest struct {
	Username string  `json:"username"`
	Email    string  `json:"email"`
	Password string  `json:"password"`
	Nickname *string `json:"nickname"`
}

type updateBusinessUserRequest struct {
	Email     *string `json:"email"`
	Nickname  *string `json:"nickname"`
	AvatarURL *string `json:"avatar_url"`
	Remark    *string `json:"remark"`
}

type resetBusinessUserPasswordRequest struct {
	NewPassword string `json:"new_password"`
}

type adjustPointsRequest struct {
	PointsDelta      string `json:"points_delta"`
	ConsumptionDelta string `json:"consumption_delta"`
	Reason           string `json:"reason"`
	IdempotencyKey   string `json:"idempotency_key"`
}

type assignBusinessUserLevelRequest struct {
	LevelMode string `json:"level_mode"`
	LevelID   string `json:"level_id"`
}

type createUserLevelRequest struct {
	Code            string `json:"code"`
	Name            string `json:"name"`
	IconURL         string `json:"icon_url"`
	ThresholdPoints string `json:"threshold_points"`
	SortOrder       int    `json:"sort_order"`
	Enabled         bool   `json:"enabled"`
}

type updateUserLevelRequest struct {
	Name            *string `json:"name"`
	IconURL         *string `json:"icon_url"`
	ThresholdPoints *string `json:"threshold_points"`
	SortOrder       *int    `json:"sort_order"`
	Enabled         *bool   `json:"enabled"`
}

type updateSystemSettingsRequest struct {
	PlatformName              string  `json:"platform_name"`
	PublicFrontendURL         string  `json:"public_frontend_url"`
	PublicAPIURL              string  `json:"public_api_url"`
	RegistrationEnabled       bool    `json:"registration_enabled"`
	UsernameLoginEnabled      bool    `json:"username_login_enabled"`
	EmailLoginEnabled         bool    `json:"email_login_enabled"`
	EmailVerificationRequired bool    `json:"email_verification_required"`
	DefaultLevelID            string  `json:"default_level_id"`
	DefaultAvatarURL          string  `json:"default_avatar_url"`
	RegistrationPoints        string  `json:"registration_points"`
	SMTPEnabled               bool    `json:"smtp_enabled"`
	SMTPHost                  string  `json:"smtp_host"`
	SMTPPort                  int     `json:"smtp_port"`
	SMTPUsername              string  `json:"smtp_username"`
	SMTPPassword              *string `json:"smtp_password"`
	SMTPFromEmail             string  `json:"smtp_from_email"`
	SMTPFromName              string  `json:"smtp_from_name"`
	SMTPTLSMode               string  `json:"smtp_tls_mode"`
	Version                   int     `json:"version"`
}

// --- business-user response DTOs ---

type publicSettingsData struct {
	PlatformName              string `json:"platform_name"`
	PublicFrontendURL         string `json:"public_frontend_url"`
	PublicAPIURL              string `json:"public_api_url"`
	RegistrationEnabled       bool   `json:"registration_enabled"`
	UsernameLoginEnabled      bool   `json:"username_login_enabled"`
	EmailLoginEnabled         bool   `json:"email_login_enabled"`
	EmailVerificationRequired bool   `json:"email_verification_required"`
	DefaultAvatarURL          string `json:"default_avatar_url"`
}

type userLoginData struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	ExpiresIn   int64  `json:"expires_in"`
}

type userMeData struct {
	ID                string         `json:"id"`
	Username          string         `json:"username"`
	Email             string         `json:"email"`
	Nickname          string         `json:"nickname"`
	AvatarURL         string         `json:"avatar_url"`
	Status            string         `json:"status"`
	EmailVerifiedAt   string         `json:"email_verified_at"`
	LastLoginAt       string         `json:"last_login_at"`
	PointsBalance     string         `json:"points_balance"`
	ConsumptionPoints string         `json:"consumption_points"`
	Level             userLevelBrief `json:"level"`
	CreatedAt         string         `json:"created_at"`
	UpdatedAt         string         `json:"updated_at"`
}

type userLevelBrief struct {
	ID   string `json:"id"`
	Code string `json:"code"`
	Name string `json:"name"`
	Mode string `json:"mode"`
}

type businessUserData struct {
	ID                string         `json:"id"`
	Username          string         `json:"username"`
	Email             string         `json:"email"`
	Nickname          string         `json:"nickname"`
	AvatarURL         string         `json:"avatar_url"`
	Status            string         `json:"status"`
	RegistrationIP    string         `json:"registration_ip"`
	EmailVerifiedAt   string         `json:"email_verified_at"`
	LastLoginAt       string         `json:"last_login_at"`
	PointsBalance     string         `json:"points_balance"`
	ConsumptionPoints string         `json:"consumption_points"`
	Level             userLevelBrief `json:"level"`
	Remark            string         `json:"remark"`
	CreatedAt         string         `json:"created_at"`
	UpdatedAt         string         `json:"updated_at"`
}

type pointTransactionData struct {
	ID               string  `json:"id"`
	PointsDelta      string  `json:"points_delta"`
	ConsumptionDelta string  `json:"consumption_delta"`
	BalanceAfter     string  `json:"balance_after"`
	ConsumptionAfter string  `json:"consumption_after"`
	Reason           string  `json:"reason"`
	ActorID          *string `json:"actor_id"`
	IdempotencyKey   string  `json:"idempotency_key"`
	CreatedAt        string  `json:"created_at"`
}

type userLevelData struct {
	ID              string `json:"id"`
	Code            string `json:"code"`
	Name            string `json:"name"`
	IconURL         string `json:"icon_url"`
	ThresholdPoints string `json:"threshold_points"`
	SortOrder       int    `json:"sort_order"`
	Enabled         bool   `json:"enabled"`
	CreatedAt       string `json:"created_at"`
	UpdatedAt       string `json:"updated_at"`
}

type systemSettingsData struct {
	PlatformName              string `json:"platform_name"`
	PublicFrontendURL         string `json:"public_frontend_url"`
	PublicAPIURL              string `json:"public_api_url"`
	RegistrationEnabled       bool   `json:"registration_enabled"`
	UsernameLoginEnabled      bool   `json:"username_login_enabled"`
	EmailLoginEnabled         bool   `json:"email_login_enabled"`
	EmailVerificationRequired bool   `json:"email_verification_required"`
	DefaultLevelID            string `json:"default_level_id"`
	DefaultAvatarURL          string `json:"default_avatar_url"`
	RegistrationPoints        string `json:"registration_points"`
	SMTPEnabled               bool   `json:"smtp_enabled"`
	SMTPHost                  string `json:"smtp_host"`
	SMTPPort                  int    `json:"smtp_port"`
	SMTPUsername              string `json:"smtp_username"`
	PasswordConfigured        bool   `json:"password_configured"`
	SMTPFromEmail             string `json:"smtp_from_email"`
	SMTPFromName              string `json:"smtp_from_name"`
	SMTPTLSMode               string `json:"smtp_tls_mode"`
	Version                   int    `json:"version"`
	UpdatedBy                 string `json:"updated_by"`
	UpdatedAt                 string `json:"updated_at"`
}

// --- conversion helpers ---

func parseID(raw string) (int64, error) {
	id, err := strconv.ParseInt(raw, 10, 64)
	if err != nil {
		return 0, err
	}
	return id, nil
}

func toPublicSettingsData(in *PublicSettingsData) publicSettingsData {
	if in == nil {
		return publicSettingsData{}
	}
	return publicSettingsData{
		PlatformName:              in.PlatformName,
		PublicFrontendURL:         in.PublicFrontendURL,
		PublicAPIURL:              in.PublicAPIURL,
		RegistrationEnabled:       in.RegistrationEnabled,
		UsernameLoginEnabled:      in.UsernameLoginEnabled,
		EmailLoginEnabled:         in.EmailLoginEnabled,
		EmailVerificationRequired: in.EmailVerificationRequired,
		DefaultAvatarURL:          in.DefaultAvatarURL,
	}
}

func toUserMeData(u *userdomain.User) userMeData {
	if u == nil {
		return userMeData{Level: userLevelBrief{}}
	}
	return userMeData{
		ID:                idString(u.ID),
		Username:          u.Username,
		Email:             u.Email,
		Nickname:          u.Nickname,
		AvatarURL:         u.AvatarURL,
		Status:            u.Status,
		EmailVerifiedAt:   formatTimePtr(u.EmailVerifiedAt),
		LastLoginAt:       formatTimePtr(u.LastLoginAt),
		PointsBalance:     u.PointsBalance.String(),
		ConsumptionPoints: u.ConsumptionPoints.String(),
		Level: userLevelBrief{
			ID:   idString(u.LevelID),
			Code: u.LevelCode,
			Name: u.LevelName,
			Mode: u.LevelMode,
		},
		CreatedAt: formatTime(u.CreatedAt),
		UpdatedAt: formatTime(u.UpdatedAt),
	}
}

func toBusinessUserData(u *userdomain.User) businessUserData {
	if u == nil {
		return businessUserData{Level: userLevelBrief{}}
	}
	return businessUserData{
		ID:                idString(u.ID),
		Username:          u.Username,
		Email:             u.Email,
		Nickname:          u.Nickname,
		AvatarURL:         u.AvatarURL,
		Status:            u.Status,
		RegistrationIP:    u.RegistrationIP,
		EmailVerifiedAt:   formatTimePtr(u.EmailVerifiedAt),
		LastLoginAt:       formatTimePtr(u.LastLoginAt),
		PointsBalance:     u.PointsBalance.String(),
		ConsumptionPoints: u.ConsumptionPoints.String(),
		Level: userLevelBrief{
			ID:   idString(u.LevelID),
			Code: u.LevelCode,
			Name: u.LevelName,
			Mode: u.LevelMode,
		},
		Remark:    u.Remark,
		CreatedAt: formatTime(u.CreatedAt),
		UpdatedAt: formatTime(u.UpdatedAt),
	}
}

func toBusinessUserList(items []userdomain.User) []businessUserData {
	out := make([]businessUserData, 0, len(items))
	for i := range items {
		out = append(out, toBusinessUserData(&items[i]))
	}
	return out
}

func toPointTransactionData(pt *userdomain.PointTransaction) pointTransactionData {
	if pt == nil {
		return pointTransactionData{}
	}
	var actorID *string
	if pt.ActorID != nil {
		s := idString(*pt.ActorID)
		actorID = &s
	}
	return pointTransactionData{
		ID:               idString(pt.ID),
		PointsDelta:      pt.PointsDelta.String(),
		ConsumptionDelta: pt.ConsumptionDelta.String(),
		BalanceAfter:     pt.BalanceAfter.String(),
		ConsumptionAfter: pt.ConsumptionAfter.String(),
		Reason:           pt.Reason,
		ActorID:          actorID,
		IdempotencyKey:   pt.IdempotencyKey,
		CreatedAt:        formatTime(pt.CreatedAt),
	}
}

func toPointTransactionList(items []userdomain.PointTransaction) []pointTransactionData {
	out := make([]pointTransactionData, 0, len(items))
	for i := range items {
		out = append(out, toPointTransactionData(&items[i]))
	}
	return out
}

func toUserLevelData(l *userdomain.UserLevel) userLevelData {
	if l == nil {
		return userLevelData{}
	}
	return userLevelData{
		ID:              idString(l.ID),
		Code:            l.Code,
		Name:            l.Name,
		IconURL:         l.IconURL,
		ThresholdPoints: l.ThresholdPoints.String(),
		SortOrder:       l.SortOrder,
		Enabled:         l.Enabled,
		CreatedAt:       formatTime(l.CreatedAt),
		UpdatedAt:       formatTime(l.UpdatedAt),
	}
}

func toUserLevelList(items []userdomain.UserLevel) []userLevelData {
	out := make([]userLevelData, 0, len(items))
	for i := range items {
		out = append(out, toUserLevelData(&items[i]))
	}
	return out
}

func toSystemSettingsData(s *userdomain.SystemSettings) systemSettingsData {
	if s == nil {
		return systemSettingsData{}
	}
	return systemSettingsData{
		PlatformName:              s.PlatformName,
		PublicFrontendURL:         s.PublicFrontendURL,
		PublicAPIURL:              s.PublicAPIURL,
		RegistrationEnabled:       s.RegistrationEnabled,
		UsernameLoginEnabled:      s.UsernameLoginEnabled,
		EmailLoginEnabled:         s.EmailLoginEnabled,
		EmailVerificationRequired: s.EmailVerificationRequired,
		DefaultLevelID:            idString(s.DefaultLevelID),
		DefaultAvatarURL:          s.DefaultAvatarURL,
		RegistrationPoints:        s.RegistrationPoints.String(),
		SMTPEnabled:               s.SMTPEnabled,
		SMTPHost:                  s.SMTPHost,
		SMTPPort:                  s.SMTPPort,
		SMTPUsername:              s.SMTPUsername,
		PasswordConfigured:        s.SMTPPasswordConfigured,
		SMTPFromEmail:             s.SMTPFromEmail,
		SMTPFromName:              s.SMTPFromName,
		SMTPTLSMode:               s.SMTPTLSMode,
		Version:                   s.Version,
		UpdatedBy:                 idString(s.UpdatedBy),
		UpdatedAt:                 formatTime(s.UpdatedAt),
	}
}

func formatTimePtr(t *time.Time) string {
	if t == nil {
		return ""
	}
	return formatTime(*t)
}
