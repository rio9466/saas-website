package adminauth

import (
	"context"
	"strconv"
	"strings"

	"github.com/rio9466/easy-admin/server/internal/domain/adminauth"
	"github.com/rio9466/easy-admin/server/internal/domain/apperr"
)

// AuthzPrincipal is live authorization state for transport middleware.
type AuthzPrincipal struct {
	AdminID         int64
	Username        string
	DisplayName     string
	Enabled         bool
	AuthEpoch       int64
	RoleCodes       []string
	PermissionCodes []string
}

// CreateRoleInput creates a custom (non-built-in) role.
type CreateRoleInput struct {
	Code        string
	Name        string
	Description string
}

// UpdateRoleInput updates mutable role fields.
type UpdateRoleInput struct {
	Name        *string
	Description *string
	Enabled     *bool
}

// LoadAuthz loads live enablement, roles, and effective permissions.
func (s *Service) LoadAuthz(ctx context.Context, adminID int64) (*AuthzPrincipal, error) {
	if adminID <= 0 {
		return nil, apperr.New(401, apperr.CodeUnauthorized, "unauthorized", apperr.ErrUnauthorized)
	}
	admin, err := s.admins.GetAdministratorByID(ctx, adminID)
	if err != nil {
		return nil, mapError(err)
	}
	perms, err := s.admins.EffectivePermissionCodes(ctx, adminID)
	if err != nil {
		return nil, mapError(err)
	}
	return &AuthzPrincipal{
		AdminID:         admin.ID,
		Username:        admin.Username,
		DisplayName:     admin.DisplayName,
		Enabled:         admin.Enabled,
		AuthEpoch:       admin.AuthEpoch,
		RoleCodes:       cloneStrings(admin.RoleCodes),
		PermissionCodes: cloneStrings(perms),
	}, nil
}

// ListRoles returns all roles with permission codes.
func (s *Service) ListRoles(ctx context.Context, actor Actor) ([]adminauth.Role, error) {
	if err := s.requireAuth(actor); err != nil {
		return nil, err
	}
	if err := s.requirePermission(ctx, actor, adminauth.PermAdminRoleRead); err != nil {
		return nil, err
	}
	roles, err := s.admins.ListRoles(ctx)
	if err != nil {
		return nil, mapError(err)
	}
	return roles, nil
}

// GetRole returns one role by ID.
func (s *Service) GetRole(ctx context.Context, actor Actor, id int64) (*adminauth.Role, error) {
	if err := s.requireAuth(actor); err != nil {
		return nil, err
	}
	if err := s.requirePermission(ctx, actor, adminauth.PermAdminRoleRead); err != nil {
		return nil, err
	}
	if id <= 0 {
		return nil, validationError("invalid role id")
	}
	role, err := s.admins.GetRoleByID(ctx, id)
	if err != nil {
		return nil, mapError(err)
	}
	return role, nil
}

// CreateRole creates a custom role. Requires admin.role.manage.
func (s *Service) CreateRole(ctx context.Context, actor Actor, in CreateRoleInput) (*adminauth.Role, error) {
	if err := s.requireAuth(actor); err != nil {
		return nil, err
	}
	if err := s.requirePermission(ctx, actor, adminauth.PermAdminRoleManage); err != nil {
		return nil, err
	}

	code := strings.ToLower(strings.TrimSpace(in.Code))
	name := strings.TrimSpace(in.Name)
	if code == "" || name == "" {
		return nil, validationError("role code and name are required")
	}
	if code == adminauth.RoleSuperAdmin || code == adminauth.RoleAdmin || code == adminauth.RoleFinance {
		return nil, apperr.New(409, apperr.CodeConflict, "conflict", apperr.ErrConflict)
	}

	role := &adminauth.Role{
		Code:        code,
		Name:        name,
		Description: strings.TrimSpace(in.Description),
		BuiltIn:     false,
		Enabled:     true,
	}

	_, err := s.createAudited(ctx, actor, ActionRoleCreate, ResourceRole, map[string]any{
		"role_code":   code,
		"name":        name,
		"description": strings.TrimSpace(in.Description),
		"username":    actor.Username,
	}, func() (int64, error) {
		if err := s.admins.CreateRole(ctx, role); err != nil {
			return 0, err
		}
		return role.ID, nil
	})
	if err != nil {
		return nil, err
	}
	return s.admins.GetRoleByID(ctx, role.ID)
}

// UpdateRole updates a role. Built-in super_admin cannot be altered.
func (s *Service) UpdateRole(ctx context.Context, actor Actor, id int64, in UpdateRoleInput) (*adminauth.Role, error) {
	if err := s.requireAuth(actor); err != nil {
		return nil, err
	}
	if err := s.requirePermission(ctx, actor, adminauth.PermAdminRoleManage); err != nil {
		return nil, err
	}
	if id <= 0 {
		return nil, validationError("invalid role id")
	}

	existing, err := s.admins.GetRoleByID(ctx, id)
	if err != nil {
		return nil, mapError(err)
	}
	// The built-in super_admin role is permanent and non-disableable. Other
	// built-in roles (admin, finance) and custom roles may be enabled or
	// disabled; disabling one stops granting identity and permissions on the
	// next protected request.
	if strings.EqualFold(existing.Code, adminauth.RoleSuperAdmin) {
		return nil, apperr.New(403, apperr.CodeForbidden, "forbidden", apperr.ErrForbidden)
	}

	details := map[string]any{
		"role_id":   strconv.FormatInt(id, 10),
		"role_code": existing.Code,
	}
	if in.Name != nil {
		details["name"] = strings.TrimSpace(*in.Name)
	}
	if in.Description != nil {
		details["description"] = strings.TrimSpace(*in.Description)
	}
	if in.Enabled != nil {
		details["enabled"] = *in.Enabled
	}

	err = s.withPendingAudit(ctx, actor, ActionRoleUpdate, ResourceRole, strconv.FormatInt(id, 10), details, func() error {
		return s.admins.UpdateRole(ctx, id, in.Name, in.Description, in.Enabled)
	})
	if err != nil {
		return nil, err
	}
	return s.admins.GetRoleByID(ctx, id)
}

// ReplaceRolePermissions replaces permission assignments for a role.
func (s *Service) ReplaceRolePermissions(ctx context.Context, actor Actor, id int64, permissionCodes []string) (*adminauth.Role, error) {
	if err := s.requireAuth(actor); err != nil {
		return nil, err
	}
	if err := s.requirePermission(ctx, actor, adminauth.PermAdminRoleManage); err != nil {
		return nil, err
	}
	if id <= 0 {
		return nil, validationError("invalid role id")
	}

	existing, err := s.admins.GetRoleByID(ctx, id)
	if err != nil {
		return nil, mapError(err)
	}
	if strings.EqualFold(existing.Code, adminauth.RoleSuperAdmin) {
		return nil, apperr.New(403, apperr.CodeForbidden, "forbidden", apperr.ErrForbidden)
	}

	codes := normalizeRoleCodes(permissionCodes)
	if err := s.ensureActorCanGrantPermissions(ctx, actor, codes); err != nil {
		return nil, err
	}
	permIDs, err := s.admins.GetPermissionIDsByCodes(ctx, codes)
	if err != nil {
		return nil, mapError(err)
	}
	if len(permIDs) != len(codes) {
		return nil, validationError("unknown permission code")
	}

	err = s.withPendingAudit(ctx, actor, ActionRoleAssignPerms, ResourceRole, strconv.FormatInt(id, 10), map[string]any{
		"role_id":          strconv.FormatInt(id, 10),
		"role_code":        existing.Code,
		"permission_codes": codes,
	}, func() error {
		return s.admins.ReplaceRolePermissions(ctx, id, permIDs)
	})
	if err != nil {
		return nil, err
	}
	return s.admins.GetRoleByID(ctx, id)
}

// ListPermissions returns the permission catalog.
func (s *Service) ListPermissions(ctx context.Context, actor Actor) ([]adminauth.Permission, error) {
	if err := s.requireAuth(actor); err != nil {
		return nil, err
	}
	if err := s.requirePermission(ctx, actor, adminauth.PermAdminRoleRead); err != nil {
		return nil, err
	}
	perms, err := s.admins.ListPermissions(ctx)
	if err != nil {
		return nil, mapError(err)
	}
	return perms, nil
}

func (s *Service) ensureActorCanGrantPermissions(ctx context.Context, actor Actor, codes []string) error {
	if actorIsSuper(actor) {
		return nil
	}
	held, err := s.admins.EffectivePermissionCodes(ctx, actor.ID)
	if err != nil {
		return mapError(err)
	}
	for _, code := range codes {
		if !containsCode(held, code) {
			return apperr.New(403, apperr.CodeForbidden, "forbidden", apperr.ErrForbidden)
		}
	}
	return nil
}
