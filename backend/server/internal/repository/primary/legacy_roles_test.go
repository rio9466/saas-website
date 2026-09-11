package primary_test

import (
	"context"
	"errors"
	"testing"

	"github.com/rio9466/easy-admin/server/internal/domain/adminauth"
	"github.com/rio9466/easy-admin/server/internal/repository/primary"
)

// seedReservedRole returns the role for code, creating it when missing.
// Startup migrations seed only super_admin since work order
// admin-platform-startup-20260907; tests that need the reserved admin/finance
// roles create them on demand instead of relying on seeds.
func seedReservedRole(
	t *testing.T,
	repo *primary.AdminRepository,
	ctx context.Context,
	code string,
) *adminauth.Role {
	t.Helper()
	role, err := repo.GetRoleByCode(ctx, code)
	if err == nil {
		return role
	}
	if !errors.Is(err, primary.ErrNotFound) {
		t.Fatalf("load role %s: %v", code, err)
	}
	created := &adminauth.Role{Code: code, Name: code, BuiltIn: true, Enabled: true}
	if err := repo.CreateRole(ctx, created); err != nil {
		t.Fatalf("create role %s: %v", code, err)
	}
	return created
}
