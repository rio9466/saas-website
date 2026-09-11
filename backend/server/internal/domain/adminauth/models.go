package adminauth

import "time"

const (
	RoleSuperAdmin = "super_admin"
	RoleAdmin      = "admin"
	RoleFinance    = "finance"

	PermDashboardView       = "dashboard.view"
	PermAdminUserRead       = "admin.user.read"
	PermAdminUserCreate     = "admin.user.create"
	PermAdminUserUpdate     = "admin.user.update"
	PermAdminUserDisable    = "admin.user.disable"
	PermAdminUserResetPwd   = "admin.user.reset_password"
	PermAdminUserAssignRole = "admin.user.assign_role"
	PermAdminRoleRead       = "admin.role.read"
	PermAdminRoleManage     = "admin.role.manage"
	PermAuditLogRead        = "audit.log.read"

	// Business-user management permissions (seeded by migration 000007).
	PermCustomerRead         = "admin.customer.read"
	PermCustomerCreate       = "admin.customer.create"
	PermCustomerUpdate       = "admin.customer.update"
	PermCustomerDisable      = "admin.customer.disable"
	PermCustomerResetPwd     = "admin.customer.reset_password"
	PermCustomerPoints       = "admin.customer.points"
	PermCustomerLevelAssign  = "admin.customer.level_assign"
	PermUserLevelRead        = "admin.user_level.read"
	PermUserLevelManage      = "admin.user_level.manage"
	PermSystemSettingsRead   = "admin.system_settings.read"
	PermSystemSettingsManage = "admin.system_settings.manage"

	AuditOutcomePending   = "pending"
	AuditOutcomeSucceeded = "succeeded"
	AuditOutcomeFailed    = "failed"
)

type Administrator struct {
	ID           int64
	Username     string
	PasswordHash string
	DisplayName  string
	Enabled      bool
	AuthEpoch    int64
	CreatedAt    time.Time
	UpdatedAt    time.Time
	RoleCodes    []string
}

type Role struct {
	ID              int64
	Code            string
	Name            string
	Description     string
	BuiltIn         bool
	Enabled         bool
	CreatedAt       time.Time
	UpdatedAt       time.Time
	PermissionCodes []string
}

type Permission struct {
	ID          int64
	Code        string
	Name        string
	Description string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

type AuditEvent struct {
	ID               int64
	ActorID          *int64
	ActorUsername    string
	ActorDisplayName string
	ActorRoleCodes   []string
	Action           string
	ResourceType     string
	ResourceID       string
	Outcome          string
	Details          map[string]any
	RequestID        string
	SourceIP         string
	UserAgent        string
	EventAt          time.Time
	CreatedAt        time.Time
}

type Page[T any] struct {
	Items    []T
	Total    int64
	Page     int
	PageSize int
}
