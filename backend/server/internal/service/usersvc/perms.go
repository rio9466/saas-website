package usersvc

import "github.com/rio9466/easy-admin/server/internal/domain/adminauth"

// Backend permission codes for the business-user management surface. These
// codes are seeded by migration 000007 and checked server-side on every
// admin-facing operation; hiding UI controls is never authorization.
const (
	permissionCustomerRead         = adminauth.PermCustomerRead
	permissionCustomerCreate       = adminauth.PermCustomerCreate
	permissionCustomerUpdate       = adminauth.PermCustomerUpdate
	permissionCustomerDisable      = adminauth.PermCustomerDisable
	permissionCustomerResetPwd     = adminauth.PermCustomerResetPwd
	permissionCustomerPoints       = adminauth.PermCustomerPoints
	permissionCustomerLevelAssign  = adminauth.PermCustomerLevelAssign
	permissionUserLevelRead        = adminauth.PermUserLevelRead
	permissionUserLevelManage      = adminauth.PermUserLevelManage
	permissionSystemSettingsRead   = adminauth.PermSystemSettingsRead
	permissionSystemSettingsManage = adminauth.PermSystemSettingsManage
)
