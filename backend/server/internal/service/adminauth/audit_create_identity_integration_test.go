package adminauth_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"testing"

	"github.com/rio9466/easy-admin/server/internal/domain/adminauth"
	"github.com/rio9466/easy-admin/server/internal/repository/logdb"
)

// findAuditEvent returns the single finalized audit event for an action and
// decimal-string resource id performed by actorID.
func findAuditEvent(t *testing.T, env *httpEnv, actorID int64, action, resourceType, resourceID string) *adminauth.AuditEvent {
	t.Helper()
	page, err := env.audits.List(env.ctx, logdb.AuditListFilter{
		ActorID:      &actorID,
		Action:       action,
		ResourceType: resourceType,
		Outcome:      adminauth.AuditOutcomeSucceeded,
		Page:         1,
		PageSize:     100,
	})
	if err != nil {
		t.Fatalf("list audit %s: %v", action, err)
	}
	var matched *adminauth.AuditEvent
	for i := range page.Items {
		if page.Items[i].ResourceID == resourceID {
			if matched != nil {
				t.Fatalf("multiple %s events for resource %s", action, resourceID)
			}
			matched = &page.Items[i]
		}
	}
	if matched == nil {
		t.Fatalf("no succeeded %s audit event found for resource %s", action, resourceID)
	}
	if matched.Outcome != adminauth.AuditOutcomeSucceeded {
		t.Fatalf("%s outcome = %s, want succeeded", action, matched.Outcome)
	}
	return matched
}

func assertOnlySafeDetailKeys(t *testing.T, action string, details map[string]any, allowed ...string) {
	t.Helper()
	allow := map[string]struct{}{}
	for _, k := range allowed {
		allow[k] = struct{}{}
	}
	raw, err := json.Marshal(details)
	if err != nil {
		t.Fatalf("marshal details: %v", err)
	}
	text := string(raw)
	for _, secret := range []string{"password", "authorization", "token", "secret"} {
		if containsFold(text, secret) {
			t.Fatalf("%s audit details contain secret-like text %q: %s", action, secret, text)
		}
	}
	for key := range details {
		if _, ok := allow[key]; !ok {
			t.Fatalf("%s audit details contain non-allowlisted key %q: %s", action, key, text)
		}
	}
}

func containsFold(haystack, needle string) bool {
	for i := 0; i+len(needle) <= len(haystack); i++ {
		match := true
		for j := 0; j < len(needle); j++ {
			if lowerByte(haystack[i+j]) != lowerByte(needle[j]) {
				match = false
				break
			}
		}
		if match {
			return true
		}
	}
	return false
}

func lowerByte(b byte) byte {
	if b >= 'A' && b <= 'Z' {
		return b + ('a' - 'A')
	}
	return b
}

func stringDetail(details map[string]any, key string) string {
	s, _ := details[key].(string)
	return s
}

// TestCreateAuditCarriesResourceIdentityAndSafeDetails proves administrator and
// role create audit rows are finalized with the created resource's decimal
// resource id and complete, safe details, and never contain credentials.
func TestCreateAuditCarriesResourceIdentityAndSafeDetails(t *testing.T) {
	env := newHTTPEnv(t)
	env.resetRBAC(t)

	const (
		actorUsername = "it_audit_id_actor"
		actorPassword = "it-actor-pass-1234"
	)
	actorID := env.createAdmin(t, actorUsername, actorPassword, []string{adminauth.RoleSuperAdmin})
	actorToken, _ := env.login(t, actorUsername, actorPassword)

	// The startup migrations seed only super_admin; tests exercising the
	// reserved admin role create it on demand.
	seedReservedRole(t, env.admins, env.ctx, adminauth.RoleAdmin)

	// --- administrator.create ---
	create := env.do(t, http.MethodPost, "/api/v1/admin/administrators", actorToken, map[string]any{
		"username":     "it_audit_id_admin",
		"password":     "it-created-pass-1234",
		"display_name": "Created Admin",
		"role_codes":   []string{adminauth.RoleAdmin},
	})
	if create.status != http.StatusCreated {
		t.Fatalf("create administrator status=%d body=%v", create.status, create.body)
	}
	adminID := dataString(create.body, "id")
	if adminID == "" {
		t.Fatal("create administrator returned no id")
	}

	event := findAuditEvent(t, env, actorID, "administrator.create", "administrator", adminID)
	if event.ResourceID != adminID {
		t.Fatalf("administrator.create resource_id = %q, want %q", event.ResourceID, adminID)
	}
	if event.ResourceType != "administrator" {
		t.Fatalf("administrator.create resource_type = %q", event.ResourceType)
	}
	if stringDetail(event.Details, "username") != "it_audit_id_admin" {
		t.Fatalf("administrator.create details missing username: %v", event.Details)
	}
	if stringDetail(event.Details, "display_name") != "" && stringDetail(event.Details, "display_name") != "Created Admin" {
		t.Fatalf("unexpected display_name detail: %v", event.Details)
	}
	assertOnlySafeDetailKeys(t, "administrator.create", event.Details,
		"username", "display_name", "enabled", "role_codes", "administrator_id", "role_code", "role_id", "name", "description", "target_username", "reason", "outcome_hint")

	// --- role.create ---
	roleCode := fmt.Sprintf("it_audit_role_%d", timeNowNanos())
	roleCreate := env.do(t, http.MethodPost, "/api/v1/admin/roles", actorToken, map[string]any{
		"code":        roleCode,
		"name":        "Audit Role Name",
		"description": "Audit role description",
	})
	if roleCreate.status != http.StatusCreated {
		t.Fatalf("create role status=%d body=%v", roleCreate.status, roleCreate.body)
	}
	roleID := dataString(roleCreate.body, "id")
	if roleID == "" {
		t.Fatal("create role returned no id")
	}

	roleEvent := findAuditEvent(t, env, actorID, "role.create", "role", roleID)
	if roleEvent.ResourceID != roleID {
		t.Fatalf("role.create resource_id = %q, want %q", roleEvent.ResourceID, roleID)
	}
	if stringDetail(roleEvent.Details, "role_code") != roleCode ||
		stringDetail(roleEvent.Details, "name") != "Audit Role Name" ||
		stringDetail(roleEvent.Details, "description") != "Audit role description" {
		t.Fatalf("role.create details missing safe fields: %v", roleEvent.Details)
	}
	assertOnlySafeDetailKeys(t, "role.create", roleEvent.Details,
		"username", "display_name", "enabled", "role_codes", "administrator_id", "role_code", "role_id", "name", "description", "target_username", "reason", "outcome_hint")

	// --- role.update keeps safe name/description/enabled details ---
	update := env.do(t, http.MethodPatch, "/api/v1/admin/roles/"+roleID, actorToken, map[string]any{
		"name":        "Audit Role Renamed",
		"description": "Updated description",
		"enabled":     false,
	})
	if update.status != http.StatusOK {
		t.Fatalf("update role status=%d body=%v", update.status, update.body)
	}

	updatedEvent := findAuditEvent(t, env, actorID, "role.update", "role", roleID)
	if stringDetail(updatedEvent.Details, "name") != "Audit Role Renamed" {
		t.Fatalf("role.update details missing renamed name: %v", updatedEvent.Details)
	}
	if stringDetail(updatedEvent.Details, "description") != "Updated description" {
		t.Fatalf("role.update details missing description: %v", updatedEvent.Details)
	}
	if enabled, ok := updatedEvent.Details["enabled"].(bool); !ok || enabled {
		t.Fatalf("role.update details missing enabled=false: %v", updatedEvent.Details)
	}
	assertOnlySafeDetailKeys(t, "role.update", updatedEvent.Details,
		"username", "display_name", "enabled", "role_codes", "administrator_id", "role_code", "role_id", "name", "description", "target_username", "reason", "outcome_hint")
}
