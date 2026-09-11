package adminauth_test

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"testing"

	"github.com/rio9466/easy-admin/server/internal/domain/adminauth"
	"github.com/rio9466/easy-admin/server/internal/domain/apperr"
	"github.com/rio9466/easy-admin/server/internal/platform/auth"
	svc "github.com/rio9466/easy-admin/server/internal/service/adminauth"
)

// TestUpdateMyProfileWorksForAllRoles proves super_admin, admin, and finance can
// each change their own display name via PATCH /api/v1/admin/me without holding
// admin.user.update (finance only holds dashboard.view), and the response is the
// full /me profile with roles and permissions unchanged.
func TestUpdateMyProfileWorksForAllRoles(t *testing.T) {
	env := newHTTPEnv(t)
	superID := env.createAdmin(t, "it_prof_super", "super-pass-0001", []string{adminauth.RoleSuperAdmin})
	adminID := env.createAdmin(t, "it_prof_admin", "admin-pass-0001", []string{adminauth.RoleAdmin})
	financeID := env.createAdmin(t, "it_prof_fin", "fin-pass-000001", []string{adminauth.RoleFinance})

	cases := []struct {
		name     string
		username string
		password string
		id       int64
		roleCode string
		newName  string
	}{
		{name: "super_admin", username: "it_prof_super", password: "super-pass-0001", id: superID, roleCode: adminauth.RoleSuperAdmin, newName: "超级管理员本人"},
		{name: "admin", username: "it_prof_admin", password: "admin-pass-0001", id: adminID, roleCode: adminauth.RoleAdmin, newName: "运营管理员本人"},
		{name: "finance", username: "it_prof_fin", password: "fin-pass-000001", id: financeID, roleCode: adminauth.RoleFinance, newName: "财务专用账号"},
	}

	for _, tt := range cases {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			token, _ := env.login(t, tt.username, tt.password)

			res := env.do(t, http.MethodPatch, "/api/v1/admin/me", token, map[string]any{
				"display_name": "  " + tt.newName + "  ",
			})
			if res.status != http.StatusOK {
				t.Fatalf("PATCH /me status=%d body=%v", res.status, res.body)
			}
			data, _ := res.body["data"].(map[string]any)
			if got := data["display_name"]; got != tt.newName {
				t.Fatalf("display_name after trim = %q, want %q", got, tt.newName)
			}
			if got := data["username"]; got != tt.username {
				t.Fatalf("username = %v, want %v", got, tt.username)
			}
			if got := data["id"]; got != idString(tt.id) {
				t.Fatalf("id = %v, want %v", got, idString(tt.id))
			}
			roleCodes := stringSlice(data["role_codes"])
			if len(roleCodes) != 1 || roleCodes[0] != tt.roleCode {
				t.Fatalf("role_codes = %v, want [%s]", roleCodes, tt.roleCode)
			}

			// The write is durable in primary and the audit row exists.
			admin, err := env.admins.GetAdministratorByID(env.ctx, tt.id)
			if err != nil {
				t.Fatalf("reload admin: %v", err)
			}
			if admin.DisplayName != tt.newName {
				t.Fatalf("persisted display_name = %q, want %q", admin.DisplayName, tt.newName)
			}
			event := findAuditEvent(t, env, tt.id, svc.ActionAdminProfileUpdate, svc.ResourceAdministrator, idString(tt.id))
			assertOnlySafeDetailKeys(t, event.Action, event.Details, "administrator_id", "username", "display_name")
		})
	}
}

// TestUpdateMyProfileIgnoresOtherFields proves the DTO accepts only display_name:
// extra username/role_codes/enabled fields in the request body cannot alter the
// current account's identity, roles, or enabled state.
func TestUpdateMyProfileIgnoresOtherFields(t *testing.T) {
	env := newHTTPEnv(t)
	adminID := env.createAdmin(t, "it_prof_guard", "guard-pass-0001", []string{adminauth.RoleAdmin})
	token, _ := env.login(t, "it_prof_guard", "guard-pass-0001")

	res := env.do(t, http.MethodPatch, "/api/v1/admin/me", token, map[string]any{
		"display_name": "改我自己的名字",
		"username":     "hacked_name",
		"role_codes":   []string{adminauth.RoleSuperAdmin},
		"enabled":      false,
	})
	if res.status != http.StatusOK {
		t.Fatalf("PATCH /me status=%d body=%v", res.status, res.body)
	}

	admin, err := env.admins.GetAdministratorByID(env.ctx, adminID)
	if err != nil {
		t.Fatalf("reload admin: %v", err)
	}
	if admin.Username != "it_prof_guard" {
		t.Fatalf("username changed to %q", admin.Username)
	}
	if !admin.Enabled {
		t.Fatal("enabled changed to false")
	}
	codes, err := env.admins.RoleCodesForAdmin(env.ctx, adminID)
	if err != nil {
		t.Fatalf("roles: %v", err)
	}
	if len(codes) != 1 || codes[0] != adminauth.RoleAdmin {
		t.Fatalf("role_codes = %v, want [admin]", codes)
	}
	if admin.DisplayName != "改我自己的名字" {
		t.Fatalf("display_name = %q", admin.DisplayName)
	}
}

// TestUpdateMyProfileValidation covers unauthenticated, empty, and whitespace-only
// display names.
func TestUpdateMyProfileValidation(t *testing.T) {
	env := newHTTPEnv(t)
	env.createAdmin(t, "it_prof_val", "val-pass-000001", []string{adminauth.RoleAdmin})

	// Unauthenticated -> 401.
	if res := env.do(t, http.MethodPatch, "/api/v1/admin/me", "", map[string]any{"display_name": "x"}); res.status != http.StatusUnauthorized {
		t.Fatalf("PATCH /me unauthenticated status=%d", res.status)
	}

	token, _ := env.login(t, "it_prof_val", "val-pass-000001")
	for _, name := range []string{"", "   "} {
		res := env.do(t, http.MethodPatch, "/api/v1/admin/me", token, map[string]any{"display_name": name})
		if res.status != http.StatusBadRequest {
			t.Fatalf("PATCH /me with display_name=%q status=%d body=%v", name, res.status, res.body)
		}
	}
	// Missing body field -> 400 as well.
	res := env.do(t, http.MethodPatch, "/api/v1/admin/me", token, map[string]any{})
	if res.status != http.StatusBadRequest {
		t.Fatalf("PATCH /me with empty body status=%d", res.status)
	}
}

// TestUpdateMyProfileAuditUnavailableFailClosed proves that when the log store is
// unavailable the profile update is rejected and primary data is not modified.
func TestUpdateMyProfileAuditUnavailableFailClosed(t *testing.T) {
	// openAuthDeps returns (primaryDB, adminRepo, sessions, tokens, passwords, cleanup).
	primaryDB, adminRepo, sessions, tokens, passwords, cleanup := openAuthDeps(t)
	defer cleanup()
	ctx := context.Background()
	truncateAdmins(t, primaryDB)

	role := seedReservedRole(t, adminRepo, ctx, adminauth.RoleAdmin)
	hasher, err := auth.NewPasswordHasher(auth.MinBcryptCost)
	if err != nil {
		t.Fatalf("hasher: %v", err)
	}
	hashed, err := hasher.Hash("failclosed-pass1")
	if err != nil {
		t.Fatalf("hash: %v", err)
	}
	admin := &adminauth.Administrator{
		Username:     "it_prof_fail",
		PasswordHash: hashed,
		DisplayName:  "Original Name",
		Enabled:      true,
	}
	if err := adminRepo.CreateAdministrator(ctx, admin, []int64{role.ID}); err != nil {
		t.Fatalf("create admin: %v", err)
	}

	service, err := svc.New(adminRepo, failingAuditStore{}, tokens, sessions, passwords, slog.Default(), svc.AuthOptions{
		Environment: "development",
	})
	if err != nil {
		t.Fatalf("new service: %v", err)
	}

	_, err = service.UpdateMyProfile(ctx, svc.Actor{ID: admin.ID, Username: admin.Username, SessionID: "sess-fail-closed", RequestID: "req-fail"}, "New Name")
	if err == nil {
		t.Fatal("UpdateMyProfile succeeded although audit is unavailable")
	}
	var appErr *apperr.AppError
	if !errors.As(err, &appErr) || appErr.Code != apperr.CodeAuditUnavailable {
		t.Fatalf("err = %v, want code %d", err, apperr.CodeAuditUnavailable)
	}

	reloaded, err := adminRepo.GetAdministratorByID(ctx, admin.ID)
	if err != nil {
		t.Fatalf("reload: %v", err)
	}
	if reloaded.DisplayName != "Original Name" {
		t.Fatalf("display_name changed to %q although audit failed", reloaded.DisplayName)
	}
}

func stringSlice(v any) []string {
	raw, ok := v.([]any)
	if !ok {
		return nil
	}
	out := make([]string, 0, len(raw))
	for _, item := range raw {
		if s, ok := item.(string); ok {
			out = append(out, s)
		}
	}
	return out
}
