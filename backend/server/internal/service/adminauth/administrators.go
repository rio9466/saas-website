package adminauth

import (
	"context"
	"strings"

	"github.com/rio9466/easy-admin/server/internal/domain/adminauth"
	"github.com/rio9466/easy-admin/server/internal/domain/apperr"
	"github.com/rio9466/easy-admin/server/internal/platform/auth"
)

// CreateAdministratorInput is the validated create payload from transport.
type CreateAdministratorInput struct {
	Username    string
	Password    string
	DisplayName string
	Enabled     bool
	RoleCodes   []string
}

// UpdateAdministratorInput updates mutable profile fields.
type UpdateAdministratorInput struct {
	DisplayName *string
}

// ListAdministrators returns a paginated administrator list.
func (s *Service) ListAdministrators(ctx context.Context, actor Actor, page, pageSize int, query string) (adminauth.Page[adminauth.Administrator], error) {
	if err := s.requireAuth(actor); err != nil {
		return adminauth.Page[adminauth.Administrator]{}, err
	}
	if err := s.requirePermission(ctx, actor, adminauth.PermAdminUserRead); err != nil {
		return adminauth.Page[adminauth.Administrator]{}, err
	}
	out, err := s.admins.ListAdministrators(ctx, page, pageSize, query)
	if err != nil {
		return adminauth.Page[adminauth.Administrator]{}, mapError(err)
	}
	return out, nil
}

// GetAdministrator returns one administrator by ID.
func (s *Service) GetAdministrator(ctx context.Context, actor Actor, id int64) (*adminauth.Administrator, error) {
	if err := s.requireAuth(actor); err != nil {
		return nil, err
	}
	if err := s.requirePermission(ctx, actor, adminauth.PermAdminUserRead); err != nil {
		return nil, err
	}
	if id <= 0 {
		return nil, validationError("invalid administrator id")
	}
	admin, err := s.admins.GetAdministratorByID(ctx, id)
	if err != nil {
		return nil, mapError(err)
	}
	return admin, nil
}

// CreateAdministrator creates an administrator and assigns roles.
func (s *Service) CreateAdministrator(ctx context.Context, actor Actor, in CreateAdministratorInput) (*adminauth.Administrator, error) {
	if err := s.requireAuth(actor); err != nil {
		return nil, err
	}
	if err := s.requirePermission(ctx, actor, adminauth.PermAdminUserCreate); err != nil {
		return nil, err
	}

	username := strings.TrimSpace(in.Username)
	if username == "" {
		return nil, validationError("username is required")
	}
	if err := auth.ValidatePasswordLength(in.Password); err != nil {
		return nil, mapError(err)
	}
	roleCodes := normalizeRoleCodes(in.RoleCodes)
	if err := guardAssignableRoles(actorIsSuper(actor), roleCodes); err != nil {
		return nil, err
	}
	if err := s.ensureActorCanGrantRoles(ctx, actor, roleCodes); err != nil {
		return nil, err
	}

	roleIDs, err := s.resolveRoleIDs(ctx, roleCodes)
	if err != nil {
		return nil, err
	}
	hash, err := s.passwords.Hash(in.Password)
	if err != nil {
		return nil, mapError(err)
	}

	admin := &adminauth.Administrator{
		Username:     username,
		PasswordHash: hash,
		DisplayName:  strings.TrimSpace(in.DisplayName),
		Enabled:      in.Enabled,
		RoleCodes:    roleCodes,
	}

	_, err = s.createAudited(ctx, actor, ActionAdminCreate, ResourceAdministrator, map[string]any{
		"username":   username,
		"enabled":    in.Enabled,
		"role_codes": roleCodes,
	}, func() (int64, error) {
		if err := s.admins.CreateAdministrator(ctx, admin, roleIDs); err != nil {
			return 0, err
		}
		return admin.ID, nil
	})
	if err != nil {
		return nil, err
	}

	created, err := s.admins.GetAdministratorByID(ctx, admin.ID)
	if err != nil {
		return nil, mapError(err)
	}
	return created, nil
}

// UpdateAdministrator updates mutable profile fields.
func (s *Service) UpdateAdministrator(ctx context.Context, actor Actor, id int64, in UpdateAdministratorInput) (*adminauth.Administrator, error) {
	if err := s.requireAuth(actor); err != nil {
		return nil, err
	}
	if err := s.requirePermission(ctx, actor, adminauth.PermAdminUserUpdate); err != nil {
		return nil, err
	}
	if id <= 0 {
		return nil, validationError("invalid administrator id")
	}

	target, err := s.admins.GetAdministratorByID(ctx, id)
	if err != nil {
		return nil, mapError(err)
	}
	if err := guardManageTarget(actorIsSuper(actor), containsRole(target.RoleCodes, adminauth.RoleSuperAdmin)); err != nil {
		return nil, err
	}

	details := map[string]any{
		"administrator_id": idString(id),
		"username":         target.Username,
	}
	if in.DisplayName != nil {
		details["display_name"] = strings.TrimSpace(*in.DisplayName)
	}

	err = s.withPendingAudit(ctx, actor, ActionAdminUpdate, ResourceAdministrator, idString(id), details, func() error {
		var displayName *string
		if in.DisplayName != nil {
			trimmed := strings.TrimSpace(*in.DisplayName)
			displayName = &trimmed
		}
		return s.admins.UpdateAdministrator(ctx, id, displayName)
	})
	if err != nil {
		return nil, err
	}
	return s.admins.GetAdministratorByID(ctx, id)
}

// EnableAdministrator enables an administrator account.
func (s *Service) EnableAdministrator(ctx context.Context, actor Actor, id int64) (*adminauth.Administrator, error) {
	return s.setAdministratorEnabled(ctx, actor, id, true)
}

// DisableAdministrator disables an administrator account.
func (s *Service) DisableAdministrator(ctx context.Context, actor Actor, id int64) (*adminauth.Administrator, error) {
	return s.setAdministratorEnabled(ctx, actor, id, false)
}

func (s *Service) setAdministratorEnabled(ctx context.Context, actor Actor, id int64, enabled bool) (*adminauth.Administrator, error) {
	if err := s.requireAuth(actor); err != nil {
		return nil, err
	}
	if err := s.requirePermission(ctx, actor, adminauth.PermAdminUserDisable); err != nil {
		return nil, err
	}
	if id <= 0 {
		return nil, validationError("invalid administrator id")
	}
	if id == actor.ID {
		return nil, apperr.New(409, apperr.CodeSelfProtection, "operation not allowed on the current administrator", apperr.ErrSelfProtection)
	}

	target, err := s.admins.GetAdministratorByID(ctx, id)
	if err != nil {
		return nil, mapError(err)
	}
	targetIsSuper := containsRole(target.RoleCodes, adminauth.RoleSuperAdmin)
	if err := guardManageTarget(actorIsSuper(actor), targetIsSuper); err != nil {
		return nil, err
	}
	if !enabled && targetIsSuper && target.Enabled {
		err = s.withPendingAudit(ctx, actor, ActionAdminDisable, ResourceAdministrator, idString(id), map[string]any{
			"administrator_id": idString(id),
			"username":         target.Username,
			"enabled":          false,
		}, func() error {
			return s.admins.DisableAdministratorGuardingLastSuper(ctx, id)
		})
		if err != nil {
			return nil, err
		}
		s.revokeSessionsBestEffort(ctx, id, "administrator disable")
		return s.GetAdministrator(ctx, actor, id)
	}

	action := ActionAdminEnable
	if !enabled {
		action = ActionAdminDisable
	}
	err = s.withPendingAudit(ctx, actor, action, ResourceAdministrator, idString(id), map[string]any{
		"administrator_id": idString(id),
		"username":         target.Username,
		"enabled":          enabled,
	}, func() error {
		return s.admins.SetAdministratorEnabled(ctx, id, enabled)
	})
	if err != nil {
		return nil, err
	}
	s.revokeSessionsBestEffort(ctx, id, "administrator enable state change")
	return s.GetAdministrator(ctx, actor, id)
}

// ResetAdministratorPassword sets a new password hash for another administrator.
func (s *Service) ResetAdministratorPassword(ctx context.Context, actor Actor, id int64, newPassword string) error {
	if err := s.requireAuth(actor); err != nil {
		return err
	}
	if err := s.requirePermission(ctx, actor, adminauth.PermAdminUserResetPwd); err != nil {
		return err
	}
	if id <= 0 {
		return validationError("invalid administrator id")
	}
	if id == actor.ID {
		return apperr.New(409, apperr.CodeSelfProtection, "operation not allowed on the current administrator", apperr.ErrSelfProtection)
	}
	if err := auth.ValidatePasswordLength(newPassword); err != nil {
		return mapError(err)
	}

	target, err := s.admins.GetAdministratorByID(ctx, id)
	if err != nil {
		return mapError(err)
	}
	if err := guardManageTarget(actorIsSuper(actor), containsRole(target.RoleCodes, adminauth.RoleSuperAdmin)); err != nil {
		return err
	}

	hash, err := s.passwords.Hash(newPassword)
	if err != nil {
		return mapError(err)
	}

	// The repository write bumps the target's auth_epoch so every existing
	// target session is immediately invalid. Redis cleanup follows the audit.
	if err := s.withPendingAudit(ctx, actor, ActionAdminResetPassword, ResourceAdministrator, idString(id), map[string]any{
		"administrator_id": idString(id),
		"username":         target.Username,
	}, func() error {
		return s.admins.SetAdministratorPassword(ctx, id, hash)
	}); err != nil {
		return err
	}
	s.revokeSessionsBestEffort(ctx, id, "administrator password reset")
	return nil
}

// AssignAdministratorRoles replaces role assignments for an administrator.
func (s *Service) AssignAdministratorRoles(ctx context.Context, actor Actor, id int64, roleCodes []string) (*adminauth.Administrator, error) {
	if err := s.requireAuth(actor); err != nil {
		return nil, err
	}
	if err := s.requirePermission(ctx, actor, adminauth.PermAdminUserAssignRole); err != nil {
		return nil, err
	}
	if id <= 0 {
		return nil, validationError("invalid administrator id")
	}

	target, err := s.admins.GetAdministratorByID(ctx, id)
	if err != nil {
		return nil, mapError(err)
	}
	targetIsSuper := containsRole(target.RoleCodes, adminauth.RoleSuperAdmin)
	if err := guardManageTarget(actorIsSuper(actor), targetIsSuper); err != nil {
		return nil, err
	}

	normalized := normalizeRoleCodes(roleCodes)
	if err := guardAssignableRoles(actorIsSuper(actor), normalized); err != nil {
		return nil, err
	}
	if err := s.ensureActorCanGrantRoles(ctx, actor, normalized); err != nil {
		return nil, err
	}

	removingSuper := targetIsSuper && !containsRole(normalized, adminauth.RoleSuperAdmin)
	roleIDs, err := s.resolveRoleIDs(ctx, normalized)
	if err != nil {
		return nil, err
	}

	err = s.withPendingAudit(ctx, actor, ActionAdminAssignRoles, ResourceAdministrator, idString(id), map[string]any{
		"administrator_id": idString(id),
		"username":         target.Username,
		"role_codes":       normalized,
	}, func() error {
		if removingSuper && target.Enabled {
			return s.admins.ReplaceAdministratorRolesGuardingLastSuper(ctx, id, roleIDs)
		}
		return s.admins.ReplaceAdministratorRoles(ctx, id, roleIDs)
	})
	if err != nil {
		return nil, err
	}
	return s.admins.GetAdministratorByID(ctx, id)
}

func (s *Service) resolveRoleIDs(ctx context.Context, roleCodes []string) ([]int64, error) {
	if len(roleCodes) == 0 {
		return []int64{}, nil
	}
	ids, err := s.admins.GetRoleIDsByCodes(ctx, roleCodes)
	if err != nil {
		return nil, mapError(err)
	}
	if len(ids) != len(roleCodes) {
		return nil, validationError("one or more role codes are invalid")
	}
	return ids, nil
}

// guardManageTarget rejects non-super actors managing a super administrator.
func guardManageTarget(actorIsSuper, targetIsSuper bool) error {
	if targetIsSuper && !actorIsSuper {
		return forbidden()
	}
	return nil
}

// guardAssignableRoles rejects ordinary actors assigning the super_admin role.
func guardAssignableRoles(actorIsSuper bool, roleCodes []string) error {
	if actorIsSuper {
		return nil
	}
	if containsRole(roleCodes, adminauth.RoleSuperAdmin) {
		return forbidden()
	}
	return nil
}

// ensureActorCanGrantRoles ensures each assigned role's permissions are held by the actor.
// super_admin bypasses. Roles with no permissions are allowed.
func (s *Service) ensureActorCanGrantRoles(ctx context.Context, actor Actor, roleCodes []string) error {
	if actorIsSuper(actor) || len(roleCodes) == 0 {
		return nil
	}
	actorPerms, err := s.admins.EffectivePermissionCodes(ctx, actor.ID)
	if err != nil {
		return mapError(err)
	}
	actorSet := toCodeSet(actorPerms)
	for _, code := range roleCodes {
		role, err := s.admins.GetRoleByCode(ctx, code)
		if err != nil {
			return mapError(err)
		}
		if missing := missingCodes(actorSet, role.PermissionCodes); len(missing) > 0 {
			return forbidden()
		}
	}
	return nil
}

func toCodeSet(codes []string) map[string]struct{} {
	out := make(map[string]struct{}, len(codes))
	for _, code := range codes {
		n := strings.ToLower(strings.TrimSpace(code))
		if n == "" {
			continue
		}
		out[n] = struct{}{}
	}
	return out
}

func missingCodes(have map[string]struct{}, want []string) []string {
	var missing []string
	for _, code := range want {
		n := strings.ToLower(strings.TrimSpace(code))
		if n == "" {
			continue
		}
		if _, ok := have[n]; !ok {
			missing = append(missing, n)
		}
	}
	return missing
}

// permissionGrantAllowed reports whether actorPerms covers every permission in grant.
func permissionGrantAllowed(actorIsSuper bool, actorPerms, grant []string) error {
	if actorIsSuper {
		return nil
	}
	missing := missingCodes(toCodeSet(actorPerms), grant)
	if len(missing) > 0 {
		return forbidden()
	}
	return nil
}
