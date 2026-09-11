package user

import "time"

// User status values stored in the users table.
const (
	StatusPendingVerification = "pending_verification"
	StatusActive              = "active"
	StatusDisabled            = "disabled"
)

// Level modes for automatic/manual level selection.
const (
	LevelModeAuto   = "auto"
	LevelModeManual = "manual"
)

// User is a business user account, separate from backend administrators.
type User struct {
	ID                int64
	Username          string
	Email             string
	PasswordHash      string
	AvatarURL         string
	Nickname          string
	RegistrationIP    string
	Status            string
	EmailVerifiedAt   *time.Time
	LastLoginAt       *time.Time
	PointsBalance     Decimal4
	ConsumptionPoints Decimal4
	LevelID           int64
	LevelMode         string
	LevelCode         string
	LevelName         string
	Remark            string
	AuthEpoch         int64
	CreatedAt         time.Time
	UpdatedAt         time.Time
}

// UserLevel is an automatic/manual business-user level with an exact
// cumulative-consumption threshold.
type UserLevel struct {
	ID              int64
	Code            string
	Name            string
	IconURL         string
	ThresholdPoints Decimal4
	SortOrder       int
	Enabled         bool
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

// PointTransaction is one immutable ledger row for a user points change.
type PointTransaction struct {
	ID               int64
	UserID           int64
	PointsDelta      Decimal4
	ConsumptionDelta Decimal4
	BalanceAfter     Decimal4
	ConsumptionAfter Decimal4
	Reason           string
	ActorID          *int64
	IdempotencyKey   string
	CreatedAt        time.Time
}

// SystemSettings is the typed singleton row for platform configuration.
type SystemSettings struct {
	PlatformName              string
	PublicFrontendURL         string
	PublicAPIURL              string
	RegistrationEnabled       bool
	UsernameLoginEnabled      bool
	EmailLoginEnabled         bool
	EmailVerificationRequired bool
	DefaultLevelID            int64
	DefaultAvatarURL          string
	RegistrationPoints        Decimal4
	SMTPEnabled               bool
	SMTPHost                  string
	SMTPPort                  int
	SMTPUsername              string
	SMTPPasswordConfigured    bool
	SMTPFromEmail             string
	SMTPFromName              string
	SMTPTLSMode               string
	Version                   int
	UpdatedBy                 int64
	UpdatedAt                 time.Time
}

// SMTP TLS modes.
const (
	SMTPTLSModeNone     = "none"
	SMTPTLSModeStartTLS = "starttls"
	SMTPTLSModeSSL      = "ssl"
)

// ValidSMTPTLSMode reports whether mode is an accepted TLS mode.
func ValidSMTPTLSMode(mode string) bool {
	switch mode {
	case SMTPTLSModeNone, SMTPTLSModeStartTLS, SMTPTLSModeSSL:
		return true
	default:
		return false
	}
}
