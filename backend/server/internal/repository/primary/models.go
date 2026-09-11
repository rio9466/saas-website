package primary

import "time"

type administratorModel struct {
	ID           int64     `gorm:"column:id;primaryKey"`
	Username     string    `gorm:"column:username"`
	PasswordHash string    `gorm:"column:password_hash"`
	DisplayName  string    `gorm:"column:display_name"`
	Enabled      bool      `gorm:"column:enabled"`
	AuthEpoch    int64     `gorm:"column:auth_epoch"`
	CreatedAt    time.Time `gorm:"column:created_at"`
	UpdatedAt    time.Time `gorm:"column:updated_at"`
}

func (administratorModel) TableName() string { return "administrators" }

type roleModel struct {
	ID          int64     `gorm:"column:id;primaryKey"`
	Code        string    `gorm:"column:code"`
	Name        string    `gorm:"column:name"`
	Description string    `gorm:"column:description"`
	BuiltIn     bool      `gorm:"column:built_in"`
	Enabled     bool      `gorm:"column:enabled"`
	CreatedAt   time.Time `gorm:"column:created_at"`
	UpdatedAt   time.Time `gorm:"column:updated_at"`
}

func (roleModel) TableName() string { return "roles" }

type permissionModel struct {
	ID          int64     `gorm:"column:id;primaryKey"`
	Code        string    `gorm:"column:code"`
	Name        string    `gorm:"column:name"`
	Description string    `gorm:"column:description"`
	CreatedAt   time.Time `gorm:"column:created_at"`
	UpdatedAt   time.Time `gorm:"column:updated_at"`
}

func (permissionModel) TableName() string { return "permissions" }

type administratorRoleModel struct {
	AdministratorID int64     `gorm:"column:administrator_id;primaryKey"`
	RoleID          int64     `gorm:"column:role_id;primaryKey"`
	CreatedAt       time.Time `gorm:"column:created_at"`
}

func (administratorRoleModel) TableName() string { return "administrator_roles" }

type rolePermissionModel struct {
	RoleID       int64     `gorm:"column:role_id;primaryKey"`
	PermissionID int64     `gorm:"column:permission_id;primaryKey"`
	CreatedAt    time.Time `gorm:"column:created_at"`
}

func (rolePermissionModel) TableName() string { return "role_permissions" }
