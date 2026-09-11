package primary

import (
	"context"
	"errors"
	"fmt"
	"time"

	userdomain "github.com/rio9466/easy-admin/server/internal/domain/user"
	"gorm.io/gorm"
)

// ErrSettingsVersionConflict reports that the optimistic-lock version changed
// between read and write.
var ErrSettingsVersionConflict = errors.New("system settings version conflict")

// GetSystemSettings loads the typed singleton row.
func (r *UserRepository) GetSystemSettings(ctx context.Context) (*userdomain.SystemSettings, error) {
	var model systemSettingsModel
	if err := r.session(ctx).Where("id = 1").First(&model).Error; err != nil {
		return nil, wrapDBError(err)
	}
	return settingsFromModel(model), nil
}

// UpdateSystemSettings applies a full settings write guarded by the
// optimistic-lock version. expectedVersion must equal the row's current
// version or the update is rejected (no silent last-write-wins). smtpPassword
// is the encrypted ciphertext to store, or nil to retain the existing secret.
func (r *UserRepository) UpdateSystemSettings(ctx context.Context, in *userdomain.SystemSettings, expectedVersion int, smtpPasswordEncrypted *string) error {
	if in == nil {
		return fmt.Errorf("system settings are nil")
	}
	if in.UpdatedBy <= 0 {
		return fmt.Errorf("updater id is required")
	}

	updates := map[string]any{
		"platform_name":               in.PlatformName,
		"public_frontend_url":         in.PublicFrontendURL,
		"public_api_url":              in.PublicAPIURL,
		"registration_enabled":        in.RegistrationEnabled,
		"username_login_enabled":      in.UsernameLoginEnabled,
		"email_login_enabled":         in.EmailLoginEnabled,
		"email_verification_required": in.EmailVerificationRequired,
		"default_level_id":            in.DefaultLevelID,
		"default_avatar_url":          in.DefaultAvatarURL,
		"registration_points":         decimalToNumeric(in.RegistrationPoints),
		"smtp_enabled":                in.SMTPEnabled,
		"smtp_host":                   in.SMTPHost,
		"smtp_port":                   in.SMTPPort,
		"smtp_username":               in.SMTPUsername,
		"smtp_from_email":             in.SMTPFromEmail,
		"smtp_from_name":              in.SMTPFromName,
		"smtp_tls_mode":               in.SMTPTLSMode,
		"version":                     gorm.Expr("version + 1"),
		"updated_by":                  in.UpdatedBy,
		"updated_at":                  time.Now().UTC(),
	}
	if smtpPasswordEncrypted != nil {
		updates["smtp_password_encrypted"] = *smtpPasswordEncrypted
	}

	res := r.session(ctx).
		Model(&systemSettingsModel{}).
		Where("id = 1 AND version = ?", expectedVersion).
		Updates(updates)
	if res.Error != nil {
		return wrapDBError(res.Error)
	}
	if res.RowsAffected == 0 {
		// Either the singleton vanished or the version moved on.
		var count int64
		if err := r.session(ctx).Model(&systemSettingsModel{}).Where("id = 1").Count(&count).Error; err != nil {
			return wrapDBError(err)
		}
		if count == 0 {
			return ErrNotFound
		}
		return ErrSettingsVersionConflict
	}
	return nil
}

// GetSystemSettingsCiphertext returns only the stored SMTP password ciphertext
// so the service layer can decrypt it for sending email. The ciphertext must
// never cross the transport boundary.
func (r *UserRepository) GetSystemSettingsCiphertext(ctx context.Context) (string, error) {
	var ciphertext string
	err := r.session(ctx).Raw(`SELECT smtp_password_encrypted FROM system_settings WHERE id = 1`).Scan(&ciphertext).Error
	if err != nil {
		return "", wrapDBError(err)
	}
	return ciphertext, nil
}

func settingsFromModel(model systemSettingsModel) *userdomain.SystemSettings {
	regPoints, err := numericToDecimal(model.RegistrationPoints)
	if err != nil {
		regPoints = userdomain.Zero()
	}
	return &userdomain.SystemSettings{
		PlatformName:              model.PlatformName,
		PublicFrontendURL:         model.PublicFrontendURL,
		PublicAPIURL:              model.PublicAPIURL,
		RegistrationEnabled:       model.RegistrationEnabled,
		UsernameLoginEnabled:      model.UsernameLoginEnabled,
		EmailLoginEnabled:         model.EmailLoginEnabled,
		EmailVerificationRequired: model.EmailVerificationRequired,
		DefaultLevelID:            model.DefaultLevelID,
		DefaultAvatarURL:          model.DefaultAvatarURL,
		RegistrationPoints:        regPoints,
		SMTPEnabled:               model.SMTPEnabled,
		SMTPHost:                  model.SMTPHost,
		SMTPPort:                  model.SMTPPort,
		SMTPUsername:              model.SMTPUsername,
		SMTPPasswordConfigured:    model.SMTPPasswordEncrypted != "",
		SMTPFromEmail:             model.SMTPFromEmail,
		SMTPFromName:              model.SMTPFromName,
		SMTPTLSMode:               model.SMTPTLSMode,
		Version:                   model.Version,
		UpdatedBy:                 model.UpdatedBy,
		UpdatedAt:                 model.UpdatedAt,
	}
}
