package handler

import (
	"context"
	"time"

	"github.com/rio9466/easy-admin/server/internal/domain/adminauth"
	"github.com/rio9466/easy-admin/server/internal/transport/http/middleware"
)

// Actor carries request metadata for audited service operations.
type Actor struct {
	ID          int64
	Username    string
	Email       string
	Nickname    string
	DisplayName string
	RoleCodes   []string
	SessionID   string
	RequestID   string
	SourceIP    string
	UserAgent   string
}

// LoginResult is returned by Login/Refresh. RefreshToken must never appear in JSON.
type LoginResult struct {
	AccessToken      string
	RefreshToken     string
	ExpiresIn        int64
	RefreshExpiresAt time.Time
}

// MeResult is the authenticated administrator profile.
type MeResult struct {
	ID              int64
	Username        string
	DisplayName     string
	Enabled         bool
	RoleCodes       []string
	PermissionCodes []string
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

// CreateAdministratorInput is the service input for creating an administrator.
type CreateAdministratorInput struct {
	Username    string
	Password    string
	DisplayName string
	RoleCodes   []string
	Enabled     bool
}

// UpdateAdministratorInput updates mutable administrator profile fields.
type UpdateAdministratorInput struct {
	DisplayName *string
}

// CreateRoleInput creates a custom role.
type CreateRoleInput struct {
	Code        string
	Name        string
	Description string
}

// UpdateRoleInput updates a role.
type UpdateRoleInput struct {
	Name        *string
	Description *string
	Enabled     *bool
}

// AuditListInput filters paginated audit events.
type AuditListInput struct {
	ActorID      *int64
	Action       string
	ResourceType string
	Outcome      string
	RequestID    string
	From         *time.Time
	To           *time.Time
	Page         int
	PageSize     int
}

// AdminAuthService is the transport-facing admin auth/RBAC/audit use-case surface.
// The concrete implementation lives in internal/service/adminauth.
type AdminAuthService interface {
	middleware.AuthzLoader

	Login(ctx context.Context, username, password string, actor Actor) (*LoginResult, error)
	Refresh(ctx context.Context, refreshToken string, actor Actor) (*LoginResult, error)
	Logout(ctx context.Context, actor Actor) error
	Me(ctx context.Context, actor Actor) (*MeResult, error)
	ChangePassword(ctx context.Context, actor Actor, currentPassword, newPassword string) error
	UpdateMyProfile(ctx context.Context, actor Actor, displayName string) (*MeResult, error)

	ListAdministrators(ctx context.Context, actor Actor, page, pageSize int, query string) (adminauth.Page[adminauth.Administrator], error)
	GetAdministrator(ctx context.Context, actor Actor, id int64) (*adminauth.Administrator, error)
	CreateAdministrator(ctx context.Context, actor Actor, in CreateAdministratorInput) (*adminauth.Administrator, error)
	UpdateAdministrator(ctx context.Context, actor Actor, id int64, in UpdateAdministratorInput) (*adminauth.Administrator, error)
	EnableAdministrator(ctx context.Context, actor Actor, id int64) (*adminauth.Administrator, error)
	DisableAdministrator(ctx context.Context, actor Actor, id int64) (*adminauth.Administrator, error)
	ResetAdministratorPassword(ctx context.Context, actor Actor, id int64, newPassword string) error
	AssignAdministratorRoles(ctx context.Context, actor Actor, id int64, roleCodes []string) (*adminauth.Administrator, error)

	ListRoles(ctx context.Context, actor Actor) ([]adminauth.Role, error)
	GetRole(ctx context.Context, actor Actor, id int64) (*adminauth.Role, error)
	CreateRole(ctx context.Context, actor Actor, in CreateRoleInput) (*adminauth.Role, error)
	UpdateRole(ctx context.Context, actor Actor, id int64, in UpdateRoleInput) (*adminauth.Role, error)
	ReplaceRolePermissions(ctx context.Context, actor Actor, id int64, permissionCodes []string) (*adminauth.Role, error)
	ListPermissions(ctx context.Context, actor Actor) ([]adminauth.Permission, error)

	ListAuditEvents(ctx context.Context, actor Actor, in AuditListInput) (adminauth.Page[adminauth.AuditEvent], error)
	GetAuditEvent(ctx context.Context, actor Actor, id int64) (*adminauth.AuditEvent, error)
}

// AuthCookieSettings controls the HttpOnly refresh cookie.
type AuthCookieSettings struct {
	Name   string
	Path   string
	Secure bool
}

const (
	defaultRefreshCookieName = "ea_admin_refresh"
	defaultRefreshCookiePath = "/api/v1/admin/auth"
)

func (s AuthCookieSettings) normalized() AuthCookieSettings {
	if s.Name == "" {
		s.Name = defaultRefreshCookieName
	}
	if s.Path == "" {
		s.Path = defaultRefreshCookiePath
	}
	return s
}
