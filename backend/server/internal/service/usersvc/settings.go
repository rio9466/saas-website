package usersvc

import (
	"context"
	"fmt"
	"net/url"
	"strings"

	"github.com/rio9466/easy-admin/server/internal/domain/apperr"
	userdomain "github.com/rio9466/easy-admin/server/internal/domain/user"
	"github.com/rio9466/easy-admin/server/internal/platform/mailer"
)

// PublicSettings is the safe whitelist returned to unauthenticated clients.
type PublicSettings struct {
	PlatformName              string
	PublicFrontendURL         string
	PublicAPIURL              string
	RegistrationEnabled       bool
	UsernameLoginEnabled      bool
	EmailLoginEnabled         bool
	EmailVerificationRequired bool
	DefaultAvatarURL          string
}

// GetPublicSettings returns the safe public whitelist.
func (s *Service) GetPublicSettings(ctx context.Context) (*PublicSettings, error) {
	settings, err := s.users.GetSystemSettings(ctx)
	if err != nil {
		return nil, mapError(err)
	}
	return &PublicSettings{
		PlatformName:              settings.PlatformName,
		PublicFrontendURL:         settings.PublicFrontendURL,
		PublicAPIURL:              settings.PublicAPIURL,
		RegistrationEnabled:       settings.RegistrationEnabled,
		UsernameLoginEnabled:      settings.UsernameLoginEnabled,
		EmailLoginEnabled:         settings.EmailLoginEnabled,
		EmailVerificationRequired: settings.EmailVerificationRequired,
		DefaultAvatarURL:          settings.DefaultAvatarURL,
	}, nil
}

// GetSystemSettingsForActor returns the full typed settings for an authorized
// administrator. The SMTP password is represented only as a configured flag;
// ciphertext and plaintext never leave the service boundary.
func (s *Service) GetSystemSettingsForActor(ctx context.Context, actor Actor) (*userdomain.SystemSettings, error) {
	if err := s.requirePermission(ctx, actor, permissionSystemSettingsRead); err != nil {
		return nil, err
	}
	settings, err := s.users.GetSystemSettings(ctx)
	if err != nil {
		return nil, mapError(err)
	}
	return settings, nil
}

// UpdateSystemSettingsInput is the validated write for the typed singleton.
// SMTPPassword carries the plaintext to encrypt when non-empty/nil; a nil or
// empty value retains the stored secret.
type UpdateSystemSettingsInput struct {
	PlatformName              string
	PublicFrontendURL         string
	PublicAPIURL              string
	RegistrationEnabled       bool
	UsernameLoginEnabled      bool
	EmailLoginEnabled         bool
	EmailVerificationRequired bool
	DefaultLevelID            int64
	DefaultAvatarURL          string
	RegistrationPoints        userdomain.Decimal4
	SMTPEnabled               bool
	SMTPHost                  string
	SMTPPort                  int
	SMTPUsername              string
	SMTPPassword              *string
	SMTPFromEmail             string
	SMTPFromName              string
	SMTPTLSMode               string
	Version                   int
}

// UpdateSystemSettings validates and atomically writes the singleton with the
// optimistic-lock version. m is the administrator identity for updated_by and
// audit details.
func (s *Service) UpdateSystemSettings(ctx context.Context, actor Actor, in UpdateSystemSettingsInput) (*userdomain.SystemSettings, error) {
	if err := s.requirePermission(ctx, actor, permissionSystemSettingsManage); err != nil {
		return nil, err
	}
	if actor.AdminID <= 0 {
		return nil, apperr.New(401, apperr.CodeUnauthorized, "unauthorized", apperr.ErrUnauthorized)
	}
	if in.Version < 1 {
		return nil, validationError("settings version is required")
	}

	name := strings.TrimSpace(in.PlatformName)
	frontendURL := strings.TrimSpace(in.PublicFrontendURL)
	apiURL := strings.TrimSpace(in.PublicAPIURL)
	if name == "" {
		return nil, validationError("platform name is required")
	}
	if err := validatePublicURL("public_frontend_url", frontendURL); err != nil {
		return nil, err
	}
	if err := validatePublicURL("public_api_url", apiURL); err != nil {
		return nil, err
	}
	if !in.UsernameLoginEnabled && !in.EmailLoginEnabled {
		return nil, validationError("at least one login method must stay enabled")
	}
	if in.RegistrationPoints.IsNegative() {
		return nil, validationError("registration points must be zero or greater")
	}
	tlsMode := strings.TrimSpace(in.SMTPTLSMode)
	if tlsMode == "" {
		tlsMode = userdomain.SMTPTLSModeStartTLS
	}

	// Resolve the default level; a disabled level cannot be the default for
	// new user assignments.
	level, err := s.users.GetUserLevelByID(ctx, in.DefaultLevelID)
	if err != nil {
		return nil, mapError(err)
	}
	if !level.Enabled {
		return nil, validationError("default level must be enabled")
	}

	// Load the current row for SMTP secret retention/comparison and version.
	current, err := s.users.GetSystemSettings(ctx)
	if err != nil {
		return nil, mapError(err)
	}

	settings := &userdomain.SystemSettings{
		PlatformName:              name,
		PublicFrontendURL:         frontendURL,
		PublicAPIURL:              apiURL,
		RegistrationEnabled:       in.RegistrationEnabled,
		UsernameLoginEnabled:      in.UsernameLoginEnabled,
		EmailLoginEnabled:         in.EmailLoginEnabled,
		EmailVerificationRequired: in.EmailVerificationRequired,
		DefaultLevelID:            in.DefaultLevelID,
		DefaultAvatarURL:          in.DefaultAvatarURL,
		RegistrationPoints:        in.RegistrationPoints,
		SMTPEnabled:               in.SMTPEnabled,
		SMTPHost:                  strings.TrimSpace(in.SMTPHost),
		SMTPPort:                  in.SMTPPort,
		SMTPUsername:              strings.TrimSpace(in.SMTPUsername),
		SMTPFromEmail:             strings.TrimSpace(in.SMTPFromEmail),
		SMTPFromName:              strings.TrimSpace(in.SMTPFromName),
		SMTPTLSMode:               tlsMode,
		Version:                   in.Version,
		UpdatedBy:                 actor.AdminID,
	}

	// --- SMTP / verification invariants ------------------------------------
	// A new SMTP password, when provided, is encrypted at the application
	// boundary; a blank value retains the stored secret.
	var smtpPasswordEncrypted *string
	if in.SMTPPassword != nil && strings.TrimSpace(*in.SMTPPassword) != "" {
		if s.box == nil {
			return nil, smtpMasterKeyMissing()
		}
		enc, encErr := s.box.Encrypt(*in.SMTPPassword)
		if encErr != nil {
			return nil, smtpMasterKeyMissing()
		}
		smtpPasswordEncrypted = &enc
	}

	if in.SMTPEnabled {
		mailSettings := mailer.Settings{
			Host:      settings.SMTPHost,
			Port:      in.SMTPPort,
			Username:  settings.SMTPUsername,
			FromEmail: settings.SMTPFromEmail,
			FromName:  settings.SMTPFromName,
			TLSMode:   in.SMTPTLSMode,
		}
		if err := mailer.ValidateSettings(mailSettings); err != nil {
			return nil, validationError(fmt.Sprintf("invalid smtp settings: %v", err))
		}

		// Authenticated SMTP needs a decryptable password: the new one just
		// provided, or the previously stored ciphertext when retained.
		passwordOK := false
		if smtpPasswordEncrypted != nil {
			if s.box == nil {
				return nil, smtpMasterKeyMissing()
			}
			if _, decErr := s.box.Decrypt(*smtpPasswordEncrypted); decErr != nil {
				return nil, validationError("smtp password could not be validated")
			}
			passwordOK = true
		} else if current.SMTPPasswordConfigured && s.box != nil {
			stored, storedErr := s.users.GetSystemSettingsCiphertext(ctx)
			if storedErr != nil {
				return nil, mapError(storedErr)
			}
			if _, decErr := s.box.Decrypt(stored); decErr == nil {
				passwordOK = true
			}
		}
		if mailSettings.Username != "" && !passwordOK {
			return nil, validationError("smtp username requires a configured smtp password")
		}
	}

	if in.EmailVerificationRequired {
		if !in.SMTPEnabled {
			return nil, validationError("email verification requires smtp to be enabled")
		}
		switch in.SMTPTLSMode {
		case userdomain.SMTPTLSModeStartTLS, userdomain.SMTPTLSModeSSL:
		default:
			return nil, validationError("email verification requires starttls or ssl smtp encryption")
		}
		if s.box == nil {
			return nil, smtpMasterKeyMissing()
		}
		// Anonymous SMTP has no password by definition; authenticated SMTP was
		// validated above. The decryptable guarantee is the master key + any
		// retained ciphertext round-trip requirement.
		if settings.SMTPUsername != "" {
			stored, storedErr := s.users.GetSystemSettingsCiphertext(ctx)
			if storedErr != nil {
				return nil, mapError(storedErr)
			}
			if smtpPasswordEncrypted != nil {
				stored = *smtpPasswordEncrypted
			}
			if _, decErr := s.box.Decrypt(stored); decErr != nil {
				return nil, validationError("email verification requires a decryptable smtp setup")
			}
		}
	}

	auditDetails := map[string]any{
		"platform_name":               settings.PlatformName,
		"public_frontend_url":         settings.PublicFrontendURL,
		"public_api_url":              settings.PublicAPIURL,
		"registration_enabled":        settings.RegistrationEnabled,
		"username_login_enabled":      settings.UsernameLoginEnabled,
		"email_login_enabled":         settings.EmailLoginEnabled,
		"email_verification_required": settings.EmailVerificationRequired,
		"default_level_id":            idString(settings.DefaultLevelID),
		"registration_points":         settings.RegistrationPoints.String(),
		"smtp_enabled":                settings.SMTPEnabled,
		"smtp_host":                   settings.SMTPHost,
		"smtp_port":                   settings.SMTPPort,
		"smtp_username":               settings.SMTPUsername,
		"smtp_from_email":             settings.SMTPFromEmail,
		"smtp_from_name":              settings.SMTPFromName,
		"smtp_tls_mode":               settings.SMTPTLSMode,
		"password_configured":         smtpPasswordEncrypted != nil || current.SMTPPasswordConfigured,
	}
	if err := s.withPendingAudit(ctx, actor, ActionSystemSettingsUpdate, ResourceSystemSettings, "1", auditDetails, func() error {
		return s.users.UpdateSystemSettings(ctx, settings, in.Version, smtpPasswordEncrypted)
	}); err != nil {
		return nil, err
	}

	// Reload so the response reflects the persisted version and flag state.
	updated, err := s.users.GetSystemSettings(ctx)
	if err != nil {
		return nil, mapError(err)
	}
	return updated, nil
}

func smtpMasterKeyMissing() error {
	return apperr.New(503, apperr.CodeDependencyUnavailable, "dependency unavailable", apperr.ErrDependencyUnavailable)
}

func validatePublicURL(field, raw string) error {
	u, err := url.Parse(raw)
	if err != nil || u.Scheme == "" || u.Host == "" {
		return validationError(field + " must be a valid absolute http/https URL")
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return validationError(field + " must be a valid http/https URL")
	}
	return nil
}
