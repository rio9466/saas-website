package primary

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/rio9466/easy-admin/server/internal/domain/adminauth"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// AdminRepository persists administrators, roles, and permissions in primary PostgreSQL.
type AdminRepository struct {
	db *gorm.DB
}

// NewAdminRepository constructs an AdminRepository backed by db.
func NewAdminRepository(db *gorm.DB) *AdminRepository {
	return &AdminRepository{db: db}
}

func (r *AdminRepository) session(ctx context.Context) *gorm.DB {
	return r.db.WithContext(ctx)
}

// CountAdministrators returns the total number of administrator rows.
func (r *AdminRepository) CountAdministrators(ctx context.Context) (int64, error) {
	var count int64
	err := r.session(ctx).Model(&administratorModel{}).Count(&count).Error
	return count, wrapDBError(err)
}

// CreateAdministrator inserts an administrator and assigns roles in one transaction.
func (r *AdminRepository) CreateAdministrator(ctx context.Context, admin *adminauth.Administrator, roleIDs []int64) error {
	if admin == nil {
		return fmt.Errorf("administrator is nil")
	}

	now := time.Now().UTC()
	model := administratorModel{
		Username:     strings.TrimSpace(admin.Username),
		PasswordHash: admin.PasswordHash,
		DisplayName:  admin.DisplayName,
		Enabled:      admin.Enabled,
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	err := r.session(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&model).Error; err != nil {
			return err
		}
		if err := replaceAdministratorRolesTx(tx, model.ID, roleIDs, now); err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return wrapDBError(err)
	}

	admin.ID = model.ID
	admin.Username = model.Username
	admin.CreatedAt = model.CreatedAt
	admin.UpdatedAt = model.UpdatedAt
	return nil
}

// GetAdministratorByID loads an administrator and its role codes. PasswordHash is omitted.
func (r *AdminRepository) GetAdministratorByID(ctx context.Context, id int64) (*adminauth.Administrator, error) {
	var model administratorModel
	err := r.session(ctx).Where("id = ?", id).First(&model).Error
	if err != nil {
		return nil, wrapDBError(err)
	}

	codes, err := r.RoleCodesForAdmin(ctx, id)
	if err != nil {
		return nil, err
	}

	return &adminauth.Administrator{
		ID:          model.ID,
		Username:    model.Username,
		DisplayName: model.DisplayName,
		Enabled:     model.Enabled,
		AuthEpoch:   model.AuthEpoch,
		CreatedAt:   model.CreatedAt,
		UpdatedAt:   model.UpdatedAt,
		RoleCodes:   codes,
	}, nil
}

// GetAdministratorByUsername loads an administrator by normalized username, including password hash and roles.
func (r *AdminRepository) GetAdministratorByUsername(ctx context.Context, username string) (*adminauth.Administrator, error) {
	var model administratorModel
	err := r.session(ctx).
		Where("lower(btrim(username)) = lower(btrim(?))", username).
		First(&model).Error
	if err != nil {
		return nil, wrapDBError(err)
	}

	codes, err := r.RoleCodesForAdmin(ctx, model.ID)
	if err != nil {
		return nil, err
	}

	return &adminauth.Administrator{
		ID:           model.ID,
		Username:     model.Username,
		PasswordHash: model.PasswordHash,
		DisplayName:  model.DisplayName,
		Enabled:      model.Enabled,
		AuthEpoch:    model.AuthEpoch,
		CreatedAt:    model.CreatedAt,
		UpdatedAt:    model.UpdatedAt,
		RoleCodes:    codes,
	}, nil
}

// ListAdministrators returns a paginated administrator list filtered by optional query text.
func (r *AdminRepository) ListAdministrators(ctx context.Context, page, pageSize int, query string) (adminauth.Page[adminauth.Administrator], error) {
	page, pageSize = normalizePage(page, pageSize)
	out := adminauth.Page[adminauth.Administrator]{Page: page, PageSize: pageSize, Items: []adminauth.Administrator{}}

	db := r.session(ctx).Model(&administratorModel{})
	if q := strings.TrimSpace(query); q != "" {
		like := "%" + escapeLike(q) + "%"
		db = db.Where("username ILIKE ? ESCAPE '\\' OR display_name ILIKE ? ESCAPE '\\'", like, like)
	}

	if err := db.Count(&out.Total).Error; err != nil {
		return out, wrapDBError(err)
	}

	var models []administratorModel
	err := db.Order("id ASC").
		Offset((page - 1) * pageSize).
		Limit(pageSize).
		Find(&models).Error
	if err != nil {
		return out, wrapDBError(err)
	}

	out.Items = make([]adminauth.Administrator, 0, len(models))
	for _, model := range models {
		codes, err := r.RoleCodesForAdmin(ctx, model.ID)
		if err != nil {
			return out, err
		}
		out.Items = append(out.Items, adminauth.Administrator{
			ID:          model.ID,
			Username:    model.Username,
			DisplayName: model.DisplayName,
			Enabled:     model.Enabled,
			CreatedAt:   model.CreatedAt,
			UpdatedAt:   model.UpdatedAt,
			RoleCodes:   codes,
		})
	}
	return out, nil
}

// UpdateAdministrator updates mutable administrator profile fields.
func (r *AdminRepository) UpdateAdministrator(ctx context.Context, id int64, displayName *string) error {
	if displayName == nil {
		return nil
	}
	res := r.session(ctx).Model(&administratorModel{}).
		Where("id = ?", id).
		Updates(map[string]any{
			"display_name": *displayName,
			"updated_at":   time.Now().UTC(),
		})
	if res.Error != nil {
		return wrapDBError(res.Error)
	}
	if res.RowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}

// SetAdministratorEnabled sets the enabled flag and bumps the auth epoch so all
// previously issued sessions are invalidated. Re-enabling never restores old
// epochs, so sessions issued before a disable can never become valid again.
func (r *AdminRepository) SetAdministratorEnabled(ctx context.Context, id int64, enabled bool) error {
	res := r.session(ctx).Model(&administratorModel{}).
		Where("id = ?", id).
		Updates(map[string]any{
			"enabled":    enabled,
			"auth_epoch": gorm.Expr("auth_epoch + 1"),
			"updated_at": time.Now().UTC(),
		})
	if res.Error != nil {
		return wrapDBError(res.Error)
	}
	if res.RowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}

// SetAdministratorPassword replaces the stored password hash and bumps the auth
// epoch so every previously issued session is immediately invalidated.
func (r *AdminRepository) SetAdministratorPassword(ctx context.Context, id int64, passwordHash string) error {
	res := r.session(ctx).Model(&administratorModel{}).
		Where("id = ?", id).
		Updates(map[string]any{
			"password_hash": passwordHash,
			"auth_epoch":    gorm.Expr("auth_epoch + 1"),
			"updated_at":    time.Now().UTC(),
		})
	if res.Error != nil {
		return wrapDBError(res.Error)
	}
	if res.RowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}

// ReplaceAdministratorRoles replaces all role assignments for an administrator in one transaction.
func (r *AdminRepository) ReplaceAdministratorRoles(ctx context.Context, id int64, roleIDs []int64) error {
	now := time.Now().UTC()
	err := r.session(ctx).Transaction(func(tx *gorm.DB) error {
		var count int64
		if err := tx.Model(&administratorModel{}).Where("id = ?", id).Count(&count).Error; err != nil {
			return err
		}
		if count == 0 {
			return gorm.ErrRecordNotFound
		}
		if err := replaceAdministratorRolesTx(tx, id, roleIDs, now); err != nil {
			return err
		}
		return tx.Model(&administratorModel{}).Where("id = ?", id).Update("updated_at", now).Error
	})
	return wrapDBError(err)
}

const (
	bootstrapAdvisoryLockKey = "easy-admin:bootstrap-admin"
	lastSuperAdvisoryLockKey = "easy-admin:last-super-admin"
)

// BootstrapSuperAdmin atomically creates the first super administrator.
// Uses a PostgreSQL transaction advisory lock so concurrent bootstraps serialize.
func (r *AdminRepository) BootstrapSuperAdmin(ctx context.Context, admin *adminauth.Administrator, roleID int64) error {
	if admin == nil {
		return fmt.Errorf("administrator is nil")
	}
	if roleID <= 0 {
		return fmt.Errorf("super_admin role id is required")
	}

	now := time.Now().UTC()
	model := administratorModel{
		Username:     strings.TrimSpace(admin.Username),
		PasswordHash: admin.PasswordHash,
		DisplayName:  admin.DisplayName,
		Enabled:      true,
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	err := r.session(ctx).Transaction(func(tx *gorm.DB) error {
		if err := advisoryXactLock(tx, bootstrapAdvisoryLockKey); err != nil {
			return err
		}

		var count int64
		if err := tx.Model(&administratorModel{}).Count(&count).Error; err != nil {
			return err
		}
		if count > 0 {
			return ErrBootstrapAlreadyCompleted
		}

		if err := tx.Create(&model).Error; err != nil {
			return err
		}
		return replaceAdministratorRolesTx(tx, model.ID, []int64{roleID}, now)
	})
	if err != nil {
		if errors.Is(err, ErrBootstrapAlreadyCompleted) {
			return err
		}
		return wrapDBError(err)
	}

	admin.ID = model.ID
	admin.Username = model.Username
	admin.Enabled = model.Enabled
	admin.CreatedAt = model.CreatedAt
	admin.UpdatedAt = model.UpdatedAt
	return nil
}

// CountEnabledSuperAdmins counts enabled administrators that hold the super_admin role.
func (r *AdminRepository) CountEnabledSuperAdmins(ctx context.Context) (int64, error) {
	return countEnabledSuperAdminsTx(r.session(ctx))
}

// DisableAdministratorGuardingLastSuper disables an administrator inside one
// transaction protected by an advisory lock and an enabled-super count check.
func (r *AdminRepository) DisableAdministratorGuardingLastSuper(ctx context.Context, id int64) error {
	err := r.session(ctx).Transaction(func(tx *gorm.DB) error {
		if err := advisoryXactLock(tx, lastSuperAdvisoryLockKey); err != nil {
			return err
		}

		var target administratorModel
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("id = ?", id).
			First(&target).Error; err != nil {
			return err
		}

		isSuper, err := administratorHasRoleTx(tx, id, adminauth.RoleSuperAdmin)
		if err != nil {
			return err
		}
		if target.Enabled && isSuper {
			count, err := countEnabledSuperAdminsTx(tx)
			if err != nil {
				return err
			}
			if count <= 1 {
				return ErrLastSuperAdmin
			}
		}

		res := tx.Model(&administratorModel{}).
			Where("id = ?", id).
			Updates(map[string]any{
				"enabled":    false,
				"auth_epoch": gorm.Expr("auth_epoch + 1"),
				"updated_at": time.Now().UTC(),
			})
		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected == 0 {
			return gorm.ErrRecordNotFound
		}
		return nil
	})
	if errors.Is(err, ErrLastSuperAdmin) {
		return err
	}
	return wrapDBError(err)
}

// ReplaceAdministratorRolesGuardingLastSuper replaces role assignments while
// preventing removal of the last enabled super_admin in the same transaction.
func (r *AdminRepository) ReplaceAdministratorRolesGuardingLastSuper(ctx context.Context, id int64, roleIDs []int64) error {
	now := time.Now().UTC()
	err := r.session(ctx).Transaction(func(tx *gorm.DB) error {
		if err := advisoryXactLock(tx, lastSuperAdvisoryLockKey); err != nil {
			return err
		}

		var target administratorModel
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("id = ?", id).
			First(&target).Error; err != nil {
			return err
		}

		currentlySuper, err := administratorHasRoleTx(tx, id, adminauth.RoleSuperAdmin)
		if err != nil {
			return err
		}

		keepsSuper := false
		if len(roleIDs) > 0 {
			var n int64
			if err := tx.Model(&roleModel{}).
				Where("id IN ? AND enabled = TRUE AND lower(code) = ?", roleIDs, adminauth.RoleSuperAdmin).
				Count(&n).Error; err != nil {
				return err
			}
			keepsSuper = n > 0
		}

		if target.Enabled && currentlySuper && !keepsSuper {
			count, err := countEnabledSuperAdminsTx(tx)
			if err != nil {
				return err
			}
			if count <= 1 {
				return ErrLastSuperAdmin
			}
		}

		if err := replaceAdministratorRolesTx(tx, id, roleIDs, now); err != nil {
			return err
		}
		return tx.Model(&administratorModel{}).Where("id = ?", id).Update("updated_at", now).Error
	})
	if errors.Is(err, ErrLastSuperAdmin) {
		return err
	}
	return wrapDBError(err)
}

func advisoryXactLock(tx *gorm.DB, key string) error {
	return tx.Exec(`SELECT pg_advisory_xact_lock(hashtext(?))`, key).Error
}

func countEnabledSuperAdminsTx(db *gorm.DB) (int64, error) {
	var count int64
	err := db.Raw(`
		SELECT COUNT(*) FROM (
			SELECT DISTINCT a.id
			FROM administrators AS a
			JOIN administrator_roles AS ar ON ar.administrator_id = a.id
			JOIN roles AS r ON r.id = ar.role_id
			WHERE a.enabled = TRUE AND r.enabled = TRUE AND lower(r.code) = ?
		) AS enabled_supers
	`, adminauth.RoleSuperAdmin).Scan(&count).Error
	return count, err
}

func administratorHasRoleTx(tx *gorm.DB, id int64, roleCode string) (bool, error) {
	var count int64
	err := tx.Table("administrator_roles AS ar").
		Joins("JOIN roles AS r ON r.id = ar.role_id").
		Where("ar.administrator_id = ? AND r.enabled = TRUE AND lower(r.code) = lower(btrim(?))", id, roleCode).
		Count(&count).Error
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

// AdministratorHasRole reports whether the administrator holds the given enabled role code.
func (r *AdminRepository) AdministratorHasRole(ctx context.Context, id int64, roleCode string) (bool, error) {
	var count int64
	err := r.session(ctx).
		Table("administrator_roles AS ar").
		Joins("JOIN roles AS r ON r.id = ar.role_id").
		Where("ar.administrator_id = ? AND r.enabled = TRUE AND lower(r.code) = lower(btrim(?))", id, roleCode).
		Count(&count).Error
	if err != nil {
		return false, wrapDBError(err)
	}
	return count > 0, nil
}

// ListRoles returns all roles with their permission codes.
func (r *AdminRepository) ListRoles(ctx context.Context) ([]adminauth.Role, error) {
	var models []roleModel
	if err := r.session(ctx).Order("id ASC").Find(&models).Error; err != nil {
		return nil, wrapDBError(err)
	}

	roles := make([]adminauth.Role, 0, len(models))
	for _, model := range models {
		codes, err := r.permissionCodesForRole(ctx, model.ID)
		if err != nil {
			return nil, err
		}
		roles = append(roles, toDomainRole(model, codes))
	}
	return roles, nil
}

// GetRoleByID loads a role and its permission codes.
func (r *AdminRepository) GetRoleByID(ctx context.Context, id int64) (*adminauth.Role, error) {
	var model roleModel
	if err := r.session(ctx).Where("id = ?", id).First(&model).Error; err != nil {
		return nil, wrapDBError(err)
	}
	codes, err := r.permissionCodesForRole(ctx, id)
	if err != nil {
		return nil, err
	}
	role := toDomainRole(model, codes)
	return &role, nil
}

// GetRoleByCode loads a role by normalized code and its permission codes.
func (r *AdminRepository) GetRoleByCode(ctx context.Context, code string) (*adminauth.Role, error) {
	var model roleModel
	if err := r.session(ctx).
		Where("lower(btrim(code)) = lower(btrim(?))", code).
		First(&model).Error; err != nil {
		return nil, wrapDBError(err)
	}
	codes, err := r.permissionCodesForRole(ctx, model.ID)
	if err != nil {
		return nil, err
	}
	role := toDomainRole(model, codes)
	return &role, nil
}

// CreateRole inserts a role row.
func (r *AdminRepository) CreateRole(ctx context.Context, role *adminauth.Role) error {
	if role == nil {
		return fmt.Errorf("role is nil")
	}
	now := time.Now().UTC()
	model := roleModel{
		Code:        strings.TrimSpace(role.Code),
		Name:        role.Name,
		Description: role.Description,
		BuiltIn:     role.BuiltIn,
		Enabled:     role.Enabled,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	if err := r.session(ctx).Create(&model).Error; err != nil {
		return wrapDBError(err)
	}
	role.ID = model.ID
	role.Code = model.Code
	role.CreatedAt = model.CreatedAt
	role.UpdatedAt = model.UpdatedAt
	return nil
}

// UpdateRole updates mutable role fields. Nil pointers leave fields unchanged.
func (r *AdminRepository) UpdateRole(ctx context.Context, id int64, name, description *string, enabled *bool) error {
	updates := map[string]any{"updated_at": time.Now().UTC()}
	if name != nil {
		updates["name"] = *name
	}
	if description != nil {
		updates["description"] = *description
	}
	if enabled != nil {
		updates["enabled"] = *enabled
	}
	if len(updates) == 1 {
		return nil
	}
	res := r.session(ctx).Model(&roleModel{}).Where("id = ?", id).Updates(updates)
	if res.Error != nil {
		return wrapDBError(res.Error)
	}
	if res.RowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}

// ReplaceRolePermissions replaces all permissions assigned to a role in one transaction.
func (r *AdminRepository) ReplaceRolePermissions(ctx context.Context, roleID int64, permissionIDs []int64) error {
	now := time.Now().UTC()
	err := r.session(ctx).Transaction(func(tx *gorm.DB) error {
		var count int64
		if err := tx.Model(&roleModel{}).Where("id = ?", roleID).Count(&count).Error; err != nil {
			return err
		}
		if count == 0 {
			return gorm.ErrRecordNotFound
		}
		if err := tx.Where("role_id = ?", roleID).Delete(&rolePermissionModel{}).Error; err != nil {
			return err
		}
		if len(permissionIDs) == 0 {
			return nil
		}
		rows := make([]rolePermissionModel, 0, len(permissionIDs))
		for _, permissionID := range permissionIDs {
			rows = append(rows, rolePermissionModel{
				RoleID:       roleID,
				PermissionID: permissionID,
				CreatedAt:    now,
			})
		}
		return tx.Create(&rows).Error
	})
	return wrapDBError(err)
}

// ListPermissions returns the full permission catalog.
func (r *AdminRepository) ListPermissions(ctx context.Context) ([]adminauth.Permission, error) {
	var models []permissionModel
	if err := r.session(ctx).Order("id ASC").Find(&models).Error; err != nil {
		return nil, wrapDBError(err)
	}
	perms := make([]adminauth.Permission, 0, len(models))
	for _, model := range models {
		perms = append(perms, adminauth.Permission{
			ID:          model.ID,
			Code:        model.Code,
			Name:        model.Name,
			Description: model.Description,
			CreatedAt:   model.CreatedAt,
			UpdatedAt:   model.UpdatedAt,
		})
	}
	return perms, nil
}

// GetPermissionIDsByCodes resolves permission IDs for the given codes (case-insensitive).
func (r *AdminRepository) GetPermissionIDsByCodes(ctx context.Context, codes []string) ([]int64, error) {
	normalized := normalizeCodes(codes)
	if len(normalized) == 0 {
		return []int64{}, nil
	}
	var ids []int64
	err := r.session(ctx).
		Model(&permissionModel{}).
		Where("lower(btrim(code)) IN ?", normalized).
		Order("id ASC").
		Pluck("id", &ids).Error
	return ids, wrapDBError(err)
}

// GetRoleIDsByCodes resolves enabled role IDs for the given codes (case-insensitive).
// Disabled roles are excluded so they cannot be assigned or keep granting access.
func (r *AdminRepository) GetRoleIDsByCodes(ctx context.Context, codes []string) ([]int64, error) {
	normalized := normalizeCodes(codes)
	if len(normalized) == 0 {
		return []int64{}, nil
	}
	var ids []int64
	err := r.session(ctx).
		Model(&roleModel{}).
		Where("enabled = TRUE AND lower(btrim(code)) IN ?", normalized).
		Order("id ASC").
		Pluck("id", &ids).Error
	return ids, wrapDBError(err)
}

// EffectivePermissionCodes returns the union of permission codes for an administrator's roles.
// If the administrator holds super_admin, every permission code is returned.
func (r *AdminRepository) EffectivePermissionCodes(ctx context.Context, adminID int64) ([]string, error) {
	hasSuper, err := r.AdministratorHasRole(ctx, adminID, adminauth.RoleSuperAdmin)
	if err != nil {
		return nil, err
	}
	if hasSuper {
		var codes []string
		err := r.session(ctx).Model(&permissionModel{}).Order("code ASC").Pluck("code", &codes).Error
		if err != nil {
			return nil, wrapDBError(err)
		}
		if codes == nil {
			codes = []string{}
		}
		return codes, nil
	}

	// Union of permissions across the administrator's enabled roles only.
	var codes []string
	err = r.session(ctx).
		Table("permissions AS p").
		Joins("JOIN role_permissions AS rp ON rp.permission_id = p.id").
		Joins("JOIN roles AS r ON r.id = rp.role_id").
		Joins("JOIN administrator_roles AS ar ON ar.role_id = r.id").
		Where("ar.administrator_id = ? AND r.enabled = TRUE", adminID).
		Distinct("p.code").
		Order("p.code ASC").
		Pluck("p.code", &codes).Error
	if err != nil {
		return nil, wrapDBError(err)
	}
	if codes == nil {
		codes = []string{}
	}
	return codes, nil
}

// RoleCodesForAdmin returns enabled role codes assigned to an administrator.
// Disabled roles do not appear and therefore grant no identity or permissions.
func (r *AdminRepository) RoleCodesForAdmin(ctx context.Context, adminID int64) ([]string, error) {
	var codes []string
	err := r.session(ctx).
		Table("roles AS r").
		Joins("JOIN administrator_roles AS ar ON ar.role_id = r.id").
		Where("ar.administrator_id = ? AND r.enabled = TRUE", adminID).
		Order("r.code ASC").
		Pluck("r.code", &codes).Error
	if err != nil {
		return nil, wrapDBError(err)
	}
	if codes == nil {
		codes = []string{}
	}
	return codes, nil
}

func (r *AdminRepository) permissionCodesForRole(ctx context.Context, roleID int64) ([]string, error) {
	var codes []string
	err := r.session(ctx).
		Table("permissions AS p").
		Joins("JOIN role_permissions AS rp ON rp.permission_id = p.id").
		Where("rp.role_id = ?", roleID).
		Order("p.code ASC").
		Pluck("p.code", &codes).Error
	if err != nil {
		return nil, wrapDBError(err)
	}
	if codes == nil {
		codes = []string{}
	}
	return codes, nil
}

func replaceAdministratorRolesTx(tx *gorm.DB, adminID int64, roleIDs []int64, now time.Time) error {
	if err := tx.Where("administrator_id = ?", adminID).Delete(&administratorRoleModel{}).Error; err != nil {
		return err
	}
	if len(roleIDs) == 0 {
		return nil
	}
	rows := make([]administratorRoleModel, 0, len(roleIDs))
	for _, roleID := range roleIDs {
		rows = append(rows, administratorRoleModel{
			AdministratorID: adminID,
			RoleID:          roleID,
			CreatedAt:       now,
		})
	}
	return tx.Create(&rows).Error
}

func toDomainRole(model roleModel, permissionCodes []string) adminauth.Role {
	if permissionCodes == nil {
		permissionCodes = []string{}
	}
	return adminauth.Role{
		ID:              model.ID,
		Code:            model.Code,
		Name:            model.Name,
		Description:     model.Description,
		BuiltIn:         model.BuiltIn,
		Enabled:         model.Enabled,
		CreatedAt:       model.CreatedAt,
		UpdatedAt:       model.UpdatedAt,
		PermissionCodes: permissionCodes,
	}
}

func normalizePage(page, pageSize int) (int, int) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 20
	}
	if pageSize > 100 {
		pageSize = 100
	}
	return page, pageSize
}

func normalizeCodes(codes []string) []string {
	seen := make(map[string]struct{}, len(codes))
	out := make([]string, 0, len(codes))
	for _, code := range codes {
		n := strings.ToLower(strings.TrimSpace(code))
		if n == "" {
			continue
		}
		if _, ok := seen[n]; ok {
			continue
		}
		seen[n] = struct{}{}
		out = append(out, n)
	}
	return out
}

func escapeLike(s string) string {
	replacer := strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`)
	return replacer.Replace(s)
}
