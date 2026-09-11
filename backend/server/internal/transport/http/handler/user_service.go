package handler

import (
	"context"
	"time"

	"github.com/rio9466/easy-admin/server/internal/domain/adminauth"
	userdomain "github.com/rio9466/easy-admin/server/internal/domain/user"
	"github.com/rio9466/easy-admin/server/internal/transport/http/middleware"
)

// PublicSettingsData is the safe public whitelist payload.
type PublicSettingsData struct {
	PlatformName              string
	PublicFrontendURL         string
	PublicAPIURL              string
	RegistrationEnabled       bool
	UsernameLoginEnabled      bool
	EmailLoginEnabled         bool
	EmailVerificationRequired bool
	DefaultAvatarURL          string
}

// UserLoginResult carries tokens for the transport layer. RefreshToken is only
// set in an HttpOnly cookie and never appears in JSON.
type UserLoginResult struct {
	AccessToken      string
	RefreshToken     string
	RefreshExpiresAt time.Time
	ExpiresIn        int64
}

// UserRegisterInput is the self-registration payload.
type UserRegisterInput struct {
	Username string
	Email    string
	Password string
}

// UserClientService is the transport-facing public/user auth surface.
type UserClientService interface {
	middleware.UserAuthzLoader

	GetPublicSettings(ctx context.Context) (*PublicSettingsData, error)
	Register(ctx context.Context, actor Actor, in UserRegisterInput) (*userdomain.User, error)
	VerifyEmail(ctx context.Context, actor Actor, email, token string) error
	ResendVerification(ctx context.Context, actor Actor, email string) error
	Login(ctx context.Context, actor Actor, identifier, password, sourceIP string) (*UserLoginResult, *userdomain.User, error)
	Refresh(ctx context.Context, actor Actor, refreshToken string) (*UserLoginResult, *userdomain.User, error)
	Logout(ctx context.Context, actor Actor) error
	Me(ctx context.Context, actor Actor, userID int64) (*userdomain.User, error)
}

// UserListInput filters the business-user page.
type UserListInput struct {
	Query   string
	Status  string
	LevelID *int64
}

// CreateUserInput for administrator-created business users.
type CreateUserInput struct {
	Username string
	Email    string
	Password string
	Nickname *string
}

// UpdateUserInput updates email/nickname/avatar/remark only.
type UpdateUserInput struct {
	Email     *string
	Nickname  *string
	AvatarURL *string
	Remark    *string
}

// AdjustPointsInput is the atomic ledger adjustment request.
type AdjustPointsInput struct {
	PointsDelta      userdomain.Decimal4
	ConsumptionDelta userdomain.Decimal4
	Reason           string
	IdempotencyKey   string
}

// AssignLevelInput sets manual or automatic level mode.
type AssignLevelInput struct {
	UserID  int64
	LevelID int64
	Mode    string
}

// CreateLevelInput creates a user level.
type CreateLevelInput struct {
	Code            string
	Name            string
	IconURL         string
	ThresholdPoints userdomain.Decimal4
	SortOrder       int
	Enabled         bool
}

// UpdateLevelInput selectively updates a user level.
type UpdateLevelInput struct {
	Name            *string
	IconURL         *string
	ThresholdPoints *userdomain.Decimal4
	SortOrder       *int
	Enabled         *bool
}

// UpdateSystemSettingsInput is the validated settings write.
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

// UserAdminService is the transport-facing business-user management surface.
type UserAdminService interface {
	ListUsers(ctx context.Context, actor Actor, page, pageSize int, filter UserListInput) (adminauth.Page[userdomain.User], error)
	GetUser(ctx context.Context, actor Actor, id int64) (*userdomain.User, error)
	CreateUser(ctx context.Context, actor Actor, in CreateUserInput) (*userdomain.User, error)
	UpdateUser(ctx context.Context, actor Actor, id int64, in UpdateUserInput) (*userdomain.User, error)
	EnableUser(ctx context.Context, actor Actor, id int64) (*userdomain.User, error)
	DisableUser(ctx context.Context, actor Actor, id int64) (*userdomain.User, error)
	ResetUserPassword(ctx context.Context, actor Actor, id int64, newPassword string) error
	AdjustPoints(ctx context.Context, actor Actor, userID int64, in AdjustPointsInput) (*userdomain.PointTransaction, error)
	ListPointTransactions(ctx context.Context, actor Actor, userID int64, page, pageSize int) (adminauth.Page[userdomain.PointTransaction], error)
	AssignUserLevel(ctx context.Context, actor Actor, in AssignLevelInput) (*userdomain.User, error)

	ListUserLevels(ctx context.Context, actor Actor) ([]userdomain.UserLevel, error)
	GetUserLevel(ctx context.Context, actor Actor, id int64) (*userdomain.UserLevel, error)
	CreateUserLevel(ctx context.Context, actor Actor, in CreateLevelInput) (*userdomain.UserLevel, error)
	UpdateUserLevel(ctx context.Context, actor Actor, id int64, in UpdateLevelInput) (*userdomain.UserLevel, error)

	GetSystemSettings(ctx context.Context, actor Actor) (*userdomain.SystemSettings, error)
	UpdateSystemSettings(ctx context.Context, actor Actor, in UpdateSystemSettingsInput) (*userdomain.SystemSettings, error)
}
