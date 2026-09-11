package adminauth_test

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/rio9466/easy-admin/server/internal/domain/adminauth"
)

func codeList(data map[string]any, key string) []string {
	raw, _ := data[key].([]any)
	out := make([]string, 0, len(raw))
	for _, v := range raw {
		if s, ok := v.(string); ok {
			out = append(out, s)
		}
	}
	return out
}

func containsStr(items []string, want string) bool {
	for _, item := range items {
		if item == want {
			return true
		}
	}
	return false
}

// TestDisabledRoleStopsGrantingWithoutTokenExpiry proves that disabling a role
// immediately removes its role identity and permissions from /me and from the
// authorization decision of the very next protected request while the same
// unexpired access token is still presented. It also proves a disabled role
// cannot be assigned again.
func TestDisabledRoleStopsGrantingWithoutTokenExpiry(t *testing.T) {
	env := newHTTPEnv(t)
	env.resetRBAC(t)

	roleCode := fmt.Sprintf("it_auditor_%d", timeNowNanos())
	const (
		actorUsername = "it_role_actor"
		actorPassword = "it-actor-pass-1234"
		memberName    = "it_role_member"
		memberPass    = "it-member-pass-1234"
	)
	env.createAdmin(t, actorUsername, actorPassword, []string{adminauth.RoleSuperAdmin})
	actorToken, _ := env.login(t, actorUsername, actorPassword)

	create := env.do(t, http.MethodPost, "/api/v1/admin/roles", actorToken, map[string]any{
		"code": roleCode,
		"name": "Integration Auditor",
	})
	if create.status != http.StatusCreated {
		t.Fatalf("create role status=%d body=%v", create.status, create.body)
	}
	roleID := dataString(create.body, "id")
	if roleID == "" {
		t.Fatalf("create role returned no id: %v", create.body)
	}

	perms := env.do(t, http.MethodPut, "/api/v1/admin/roles/"+roleID+"/permissions", actorToken, map[string]any{
		"permission_codes": []string{adminauth.PermAdminUserRead},
	})
	if perms.status != http.StatusOK {
		t.Fatalf("assign permissions status=%d body=%v", perms.status, perms.body)
	}

	env.createAdmin(t, memberName, memberPass, []string{roleCode})
	memberToken, _ := env.login(t, memberName, memberPass)

	data := env.me(t, memberToken)
	if data.status != http.StatusOK {
		t.Fatalf("member /me before disable status=%d", data.status)
	}
	meData, _ := data.body["data"].(map[string]any)
	if !containsStr(codeList(meData, "role_codes"), roleCode) {
		t.Fatalf("member /me before disable missing role %s: %v", roleCode, meData)
	}
	if !containsStr(codeList(meData, "permission_codes"), adminauth.PermAdminUserRead) {
		t.Fatalf("member /me before disable missing admin.user.read: %v", meData)
	}

	if list := env.do(t, http.MethodGet, "/api/v1/admin/administrators", memberToken, nil); list.status != http.StatusOK {
		t.Fatalf("member list before disable status=%d, want 200", list.status)
	}

	disable := env.do(t, http.MethodPatch, "/api/v1/admin/roles/"+roleID, actorToken, map[string]any{
		"enabled": false,
	})
	if disable.status != http.StatusOK {
		t.Fatalf("disable role status=%d body=%v", disable.status, disable.body)
	}

	// Same unexpired access token: /me must drop the role and permissions.
	after := env.me(t, memberToken)
	if after.status != http.StatusOK {
		t.Fatalf("member /me after disable status=%d, want 200", after.status)
	}
	afterData, _ := after.body["data"].(map[string]any)
	if containsStr(codeList(afterData, "role_codes"), roleCode) {
		t.Fatalf("member /me after disable still lists role %s: %v", roleCode, afterData)
	}
	if containsStr(codeList(afterData, "permission_codes"), adminauth.PermAdminUserRead) {
		t.Fatalf("member /me after disable still lists admin.user.read: %v", afterData)
	}

	// Authorization middleware reads effective permissions per request, so the
	// next protected call is denied without waiting for token expiry.
	if list := env.do(t, http.MethodGet, "/api/v1/admin/administrators", memberToken, nil); list.status != http.StatusForbidden {
		t.Fatalf("member list after disable status=%d, want 403", list.status)
	}

	// Disabled roles cannot be assigned to new administrators.
	createMember := env.do(t, http.MethodPost, "/api/v1/admin/administrators", actorToken, map[string]any{
		"username":   "it_disabled_role_assign",
		"password":   "it-new-pass-1234",
		"role_codes": []string{roleCode},
	})
	if createMember.status != http.StatusBadRequest {
		t.Fatalf("assigning disabled role status=%d, want 400: %v", createMember.status, createMember.body)
	}
}

// TestSuperAdminRoleCannotBeDisabled verifies the built-in super_admin role is
// protected from disable while a built-in non-super role (finance) can be
// disabled and immediately stops granting its permissions.
func TestSuperAdminRoleCannotBeDisabledButFinanceCan(t *testing.T) {
	env := newHTTPEnv(t)
	env.resetRBAC(t)

	const (
		actorUsername = "it_role_actor2"
		actorPassword = "it-actor-pass-1234"
		finUser       = "it_fin_user"
		finPass       = "it-fin-pass-1234"
	)
	env.createAdmin(t, actorUsername, actorPassword, []string{adminauth.RoleSuperAdmin})
	actorToken, _ := env.login(t, actorUsername, actorPassword)

	superRole, err := env.admins.GetRoleByCode(env.ctx, adminauth.RoleSuperAdmin)
	if err != nil {
		t.Fatalf("load super_admin role: %v", err)
	}
	financeRole := seedReservedRole(t, env.admins, env.ctx, adminauth.RoleFinance)

	if res := env.do(t, http.MethodPatch, "/api/v1/admin/roles/"+idString(superRole.ID), actorToken, map[string]any{
		"enabled": false,
	}); res.status != http.StatusForbidden {
		t.Fatalf("disable super_admin status=%d, want 403", res.status)
	}

	env.createAdmin(t, finUser, finPass, []string{adminauth.RoleFinance})
	token, _ := env.login(t, finUser, finPass)

	me := env.me(t, token)
	meData, _ := me.body["data"].(map[string]any)
	if me.status != http.StatusOK || !containsStr(codeList(meData, "permission_codes"), adminauth.PermDashboardView) {
		t.Fatalf("finance member before disable lacks dashboard.view: status=%d data=%v", me.status, meData)
	}

	if res := env.do(t, http.MethodPatch, "/api/v1/admin/roles/"+idString(financeRole.ID), actorToken, map[string]any{
		"enabled": false,
	}); res.status != http.StatusOK {
		t.Fatalf("disable built-in finance role status=%d body=%v", res.status, res.body)
	}
	defer func() {
		// Restore finance for any subsequent test in this process.
		_ = env.admins.UpdateRole(env.ctx, financeRole.ID, nil, nil, boolPtr(true))
	}()

	after := env.me(t, token)
	afterData, _ := after.body["data"].(map[string]any)
	if after.status != http.StatusOK {
		t.Fatalf("finance /me after disable status=%d", after.status)
	}
	if containsStr(codeList(afterData, "role_codes"), adminauth.RoleFinance) {
		t.Fatalf("finance /me after disable still lists finance role: %v", afterData)
	}
	if containsStr(codeList(afterData, "permission_codes"), adminauth.PermDashboardView) {
		t.Fatalf("finance /me after disable still lists dashboard.view: %v", afterData)
	}
}

func boolPtr(b bool) *bool { return &b }
