package usersvc_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/rio9466/easy-admin/server/internal/domain/adminauth"
	"github.com/rio9466/easy-admin/server/internal/domain/apperr"
	platformauth "github.com/rio9466/easy-admin/server/internal/platform/auth"
	"github.com/rio9466/easy-admin/server/internal/repository/primary"
	usersvc "github.com/rio9466/easy-admin/server/internal/service/usersvc"
)

// This file exercises the complete business-user contract through the real
// Gin router against isolated real PostgreSQL + Redis: administrator user
// lifecycle, points exactness/idempotency, level auto/manual rules, typed
// system settings with encrypted SMTP secrets, the verification-required
// registration flow, refresh replay protection, cookie attributes, and
// server-side role denial.

const httpAdminPassword = "http-admin-pass-123"

// seedAdmin creates an administrator with a known password and one role.
func (e *realHTTPEnv) seedAdmin(t *testing.T, username, roleCode string) {
	t.Helper()
	repo := primary.NewAdminRepository(e.primaryDB.GORM())
	role := seedReservedRole(t, repo, e.ctx, roleCode)
	hasher, err := platformauth.NewPasswordHasher(12)
	if err != nil {
		t.Fatal(err)
	}
	hash, err := hasher.Hash(httpAdminPassword)
	if err != nil {
		t.Fatal(err)
	}
	admin := &adminauth.Administrator{
		Username:     username,
		PasswordHash: hash,
		DisplayName:  username,
		Enabled:      true,
	}
	if err := repo.CreateAdministrator(e.ctx, admin, []int64{role.ID}); err != nil {
		t.Fatalf("seed admin %q: %v", username, err)
	}
}

// adminToken logs in through the real admin auth endpoint and returns the
// access token (never a fabricated actor).
func (e *realHTTPEnv) adminToken(t *testing.T, username, origin string) string {
	t.Helper()
	res := e.request(http.MethodPost, "/api/v1/admin/auth/login", "", origin, map[string]any{
		"username": username,
		"password": httpAdminPassword,
	})
	if res.status != http.StatusOK {
		t.Fatalf("admin login status=%d code=%d body=%v", res.status, res.code, res.body)
	}
	token, _ := res.body["access_token"].(string)
	if token == "" {
		t.Fatal("admin login returned no access token")
	}
	return token
}

// settingsPayload reads the current typed settings and returns a ready-to-write
// payload carrying the live optimistic-lock version.
func (e *realHTTPEnv) settingsPayload(t *testing.T, admin string) map[string]any {
	t.Helper()
	res := e.request(http.MethodGet, "/api/v1/admin/system-settings", admin, "", nil)
	if res.status != http.StatusOK {
		t.Fatalf("settings read status=%d code=%d body=%v", res.status, res.code, res.body)
	}
	raw, err := json.Marshal(res.body)
	if err != nil {
		t.Fatal(err)
	}
	var payload map[string]any
	if err := json.Unmarshal(raw, &payload); err != nil {
		t.Fatal(err)
	}
	delete(payload, "password_configured")
	delete(payload, "updated_by")
	delete(payload, "updated_at")
	payload["smtp_password"] = nil
	return payload
}

func TestAdminBusinessUserLifecycleHTTP(t *testing.T) {
	e := newRealHTTPEnv(t)
	origin := "http://localhost:3000"
	e.seedAdmin(t, "it_http_super", adminauth.RoleSuperAdmin)
	admin := e.adminToken(t, "it_http_super", origin)

	// --- administrator-created account -------------------------------------
	created := e.request(http.MethodPost, "/api/v1/admin/users", admin, origin, map[string]any{
		"username": "it_admin_made",
		"email":    "IT_Admin_Made@Example.com",
		"password": "password-123456",
		"nickname": "Admin Made",
	})
	if created.status != http.StatusCreated {
		t.Fatalf("admin create status=%d code=%d body=%v", created.status, created.code, created.body)
	}
	userID, _ := created.body["id"].(string)
	if userID == "" {
		t.Fatal("created user has no id")
	}
	if got := created.body["email"]; got != "it_admin_made@example.com" {
		t.Fatalf("email must be trimmed+lowercased, got %v", got)
	}
	if got := created.body["status"]; got != "active" {
		t.Fatalf("admin-created status = %v, want active", got)
	}
	if v, _ := created.body["email_verified_at"].(string); v == "" {
		t.Fatal("admin-created account must count as email verified")
	}
	if v, _ := created.body["points_balance"].(string); v != "0.0000" {
		t.Fatalf("admin-created points = %q, want exact 0.0000", v)
	}
	if ip, _ := created.body["registration_ip"].(string); ip == "" {
		t.Fatal("registration_ip must be recorded from the server request context")
	}
	level, _ := created.body["level"].(map[string]any)
	if mode, _ := level["mode"].(string); mode != "auto" {
		t.Fatalf("level mode = %v, want auto", level["mode"])
	}
	if code, _ := level["code"].(string); code != "default" {
		t.Fatalf("created response must carry the persisted default level, got %v", level)
	}
	if lid, _ := level["id"].(string); lid == "" || lid == "0" {
		t.Fatalf("created response level id = %v, want the seeded default level", level)
	}

	// --- list / detail ------------------------------------------------------
	list := e.request(http.MethodGet, "/api/v1/admin/users?page=1&page_size=20&q=it_admin_made", admin, origin, nil)
	if list.status != http.StatusOK {
		t.Fatalf("list status=%d code=%d", list.status, list.code)
	}
	items, _ := list.body["items"].([]any)
	if len(items) != 1 {
		t.Fatalf("list items = %d, want exactly 1 for the unique query", len(items))
	}
	detail := e.request(http.MethodGet, "/api/v1/admin/users/"+userID, admin, origin, nil)
	if detail.status != http.StatusOK {
		t.Fatalf("detail status=%d code=%d", detail.status, detail.code)
	}

	// --- generic update cannot touch protected fields ----------------------
	patched := e.request(http.MethodPatch, "/api/v1/admin/users/"+userID, admin, origin, map[string]any{
		"remark":         "vip, handled by IT",
		"nickname":       "Renamed",
		"username":       "evil_rename",
		"status":         "disabled",
		"points_balance": "9999.9999",
		"level_mode":     "manual",
		"password":       "evil-password-123",
	})
	if patched.status != http.StatusOK {
		t.Fatalf("update status=%d code=%d body=%v", patched.status, patched.code, patched.body)
	}
	if patched.body["username"] != "it_admin_made" {
		t.Fatalf("username changed through generic update: %v", patched.body["username"])
	}
	if patched.body["status"] != "active" {
		t.Fatalf("status changed through generic update: %v", patched.body["status"])
	}
	if v, _ := patched.body["points_balance"].(string); v != "0.0000" {
		t.Fatalf("points changed through generic update: %q", v)
	}
	if patched.body["remark"] != "vip, handled by IT" {
		t.Fatalf("remark not applied: %v", patched.body["remark"])
	}

	// --- points: exact, ledger-backed, idempotent ---------------------------
	adjusted := e.request(http.MethodPost, "/api/v1/admin/users/"+userID+"/points-adjust", admin, origin, map[string]any{
		"points_delta":      "100.0075",
		"consumption_delta": "50.0001",
		"reason":            "manual grant for IT acceptance",
		"idempotency_key":   "it-http-points-1",
	})
	if adjusted.status != http.StatusOK {
		t.Fatalf("points adjust status=%d code=%d body=%v", adjusted.status, adjusted.code, adjusted.body)
	}
	if v, _ := adjusted.body["balance_after"].(string); v != "100.0075" {
		t.Fatalf("balance_after = %q, want 100.0075", v)
	}
	if v, _ := adjusted.body["consumption_after"].(string); v != "50.0001" {
		t.Fatalf("consumption_after = %q, want 50.0001", v)
	}
	replay := e.request(http.MethodPost, "/api/v1/admin/users/"+userID+"/points-adjust", admin, origin, map[string]any{
		"points_delta":      "100.0075",
		"consumption_delta": "50.0001",
		"reason":            "manual grant for IT acceptance",
		"idempotency_key":   "it-http-points-1",
	})
	if replay.status != http.StatusConflict || replay.code != apperr.CodeIdempotencyConflict {
		t.Fatalf("idempotency replay status=%d code=%d, want 409/%d", replay.status, replay.code, apperr.CodeIdempotencyConflict)
	}
	afterReplay := e.request(http.MethodGet, "/api/v1/admin/users/"+userID, admin, origin, nil)
	if v, _ := afterReplay.body["points_balance"].(string); v != "100.0075" {
		t.Fatalf("replayed adjustment changed balance: %q", v)
	}

	// Available points can never go negative.
	overdraw := e.request(http.MethodPost, "/api/v1/admin/users/"+userID+"/points-adjust", admin, origin, map[string]any{
		"points_delta":      "-1000.0000",
		"consumption_delta": "0.0000",
		"reason":            "must fail",
		"idempotency_key":   "it-http-points-negative",
	})
	if overdraw.status != http.StatusBadRequest && overdraw.status != http.StatusConflict {
		t.Fatalf("negative balance status=%d code=%d, want 4xx", overdraw.status, overdraw.code)
	}
	// Consumption points are cumulative and can never be reduced.
	reduce := e.request(http.MethodPost, "/api/v1/admin/users/"+userID+"/points-adjust", admin, origin, map[string]any{
		"points_delta":      "0.0000",
		"consumption_delta": "-0.0001",
		"reason":            "must fail",
		"idempotency_key":   "it-http-points-reduce",
	})
	if reduce.status != http.StatusBadRequest {
		t.Fatalf("reduce consumption status=%d code=%d, want 400", reduce.status, reduce.code)
	}
	ledger := e.request(http.MethodGet, "/api/v1/admin/users/"+userID+"/point-transactions?page=1&page_size=20", admin, origin, nil)
	if ledger.status != http.StatusOK {
		t.Fatalf("ledger status=%d code=%d", ledger.status, ledger.code)
	}
	entries, _ := ledger.body["items"].([]any)
	if len(entries) != 1 {
		t.Fatalf("ledger entries = %d, want exactly 1 (replay must not append)", len(entries))
	}

	// --- levels: auto recalculation, manual persistence, switch back --------
	vip := e.request(http.MethodPost, "/api/v1/admin/user-levels", admin, origin, map[string]any{
		"code":             "it-vip",
		"name":             "IT VIP",
		"threshold_points": "50.0000",
		"sort_order":       10,
		"enabled":          true,
	})
	if vip.status != http.StatusCreated {
		t.Fatalf("create level status=%d code=%d body=%v", vip.status, vip.code, vip.body)
	}
	vipBody, _ := vip.body["id"].(string)
	if vipBody == "" {
		t.Fatal("created level has no id")
	}
	// Any later consumption change recalculates an auto-mode user immediately.
	e.request(http.MethodPost, "/api/v1/admin/users/"+userID+"/points-adjust", admin, origin, map[string]any{
		"points_delta":      "0.0000",
		"consumption_delta": "0.0001",
		"reason":            "trigger auto level recalculation",
		"idempotency_key":   "it-http-points-2",
	})
	recalc := e.request(http.MethodGet, "/api/v1/admin/users/"+userID, admin, origin, nil)
	recalcLevel, _ := recalc.body["level"].(map[string]any)
	if got, _ := recalcLevel["code"].(string); got != "it-vip" {
		t.Fatalf("auto level after consumption increase = %v, want it-vip", recalcLevel["code"])
	}

	// Manual assignment stays authoritative across later point changes.
	manual := e.request(http.MethodPost, "/api/v1/admin/users/"+userID+"/level", admin, origin, map[string]any{
		"level_mode": "manual",
		"level_id":   "1",
	})
	if manual.status != http.StatusOK {
		t.Fatalf("manual assign status=%d code=%d body=%v", manual.status, manual.code, manual.body)
	}
	e.request(http.MethodPost, "/api/v1/admin/users/"+userID+"/points-adjust", admin, origin, map[string]any{
		"points_delta":      "0.0000",
		"consumption_delta": "500.0000",
		"reason":            "manual mode must not auto change",
		"idempotency_key":   "it-http-points-3",
	})
	stillManual := e.request(http.MethodGet, "/api/v1/admin/users/"+userID, admin, origin, nil)
	manualLevel, _ := stillManual.body["level"].(map[string]any)
	if mode, _ := manualLevel["mode"].(string); mode != "manual" {
		t.Fatalf("manual mode lost after points change: %v", manualLevel)
	}
	if code, _ := manualLevel["code"].(string); code != "default" {
		t.Fatalf("manual level changed by points: %v", code)
	}
	backToAuto := e.request(http.MethodPost, "/api/v1/admin/users/"+userID+"/level", admin, origin, map[string]any{
		"level_mode": "auto",
	})
	if backToAuto.status != http.StatusOK {
		t.Fatalf("switch to auto status=%d code=%d body=%v", backToAuto.status, backToAuto.code, backToAuto.body)
	}
	autoLevel, _ := backToAuto.body["level"].(map[string]any)
	if autoLevel == nil {
		t.Fatalf("auto switch returned no level: %v", backToAuto.body["level"])
	}
	if m, _ := autoLevel["mode"].(string); m != "auto" {
		t.Fatalf("mode after switch = %v, want auto", autoLevel["mode"])
	}
	if c, _ := autoLevel["code"].(string); c != "it-vip" {
		t.Fatalf("switch to auto must recalculate immediately, got %v", autoLevel["code"])
	}

	// --- lifecycle: disable kills the live session, reset restores access ---
	userAccess, userRefresh := e.loginUser(t, "it_admin_made", "password-123456", origin)
	_ = userRefresh
	if me := e.request(http.MethodGet, "/api/v1/me", userAccess, "", nil); me.status != http.StatusOK {
		t.Fatalf("user /me before disable status=%d", me.status)
	}
	disabled := e.request(http.MethodPost, "/api/v1/admin/users/"+userID+"/disable", admin, origin, nil)
	if disabled.status != http.StatusOK || disabled.body["status"] != "disabled" {
		t.Fatalf("disable status=%d body=%v", disabled.status, disabled.body)
	}
	if me := e.request(http.MethodGet, "/api/v1/me", userAccess, "", nil); me.status != http.StatusUnauthorized {
		t.Fatalf("/me after disable status=%d, want 401 (auth epoch invalidation)", me.status)
	}
	if login := e.request(http.MethodPost, "/api/v1/auth/login", "", origin, map[string]any{
		"identifier": "it_admin_made", "password": "password-123456",
	}); login.status != http.StatusForbidden || login.code != apperr.CodeAccountDisabled {
		t.Fatalf("disabled login status=%d code=%d, want 403/%d", login.status, login.code, apperr.CodeAccountDisabled)
	}
	reset := e.request(http.MethodPost, "/api/v1/admin/users/"+userID+"/reset-password", admin, origin, map[string]any{
		"new_password": "rotated-password-1",
	})
	if reset.status != http.StatusOK {
		t.Fatalf("reset password status=%d code=%d body=%v", reset.status, reset.code, reset.body)
	}
	enabled := e.request(http.MethodPost, "/api/v1/admin/users/"+userID+"/enable", admin, origin, nil)
	if enabled.status != http.StatusOK || enabled.body["status"] != "active" {
		t.Fatalf("enable status=%d body=%v", enabled.status, enabled.body)
	}
	rotated, _ := e.loginUser(t, "it_admin_made", "rotated-password-1", origin)
	me := e.request(http.MethodGet, "/api/v1/me", rotated, "", nil)
	if me.status != http.StatusOK {
		t.Fatalf("/me after reset+enable status=%d", me.status)
	}
	// Four-decimal values survive the whole round trip unchanged.
	if v, _ := me.body["points_balance"].(string); v != "100.0075" {
		t.Fatalf("/me points_balance = %q, want 100.0075", v)
	}
	if v, _ := me.body["consumption_points"].(string); v != "550.0002" {
		t.Fatalf("/me consumption_points = %q, want 550.0002", v)
	}
}

func TestSystemSettingsAndVerificationFlowHTTP(t *testing.T) {
	e := newRealHTTPEnv(t)
	origin := "http://localhost:3000"
	e.seedAdmin(t, "it_http_settings_super", adminauth.RoleSuperAdmin)
	admin := e.adminToken(t, "it_http_settings_super", origin)

	payload := e.settingsPayload(t, admin)
	version, _ := payload["version"].(float64)

	// Both login modes disabled is rejected.
	both := clonePayload(payload)
	both["username_login_enabled"] = false
	both["email_login_enabled"] = false
	res := e.request(http.MethodPut, "/api/v1/admin/system-settings", admin, origin, both)
	if res.status != http.StatusBadRequest || res.code != apperr.CodeValidation {
		t.Fatalf("both-login-disabled status=%d code=%d, want 400/%d", res.status, res.code, apperr.CodeValidation)
	}

	// Verification required without a usable SMTP setup is rejected.
	noSMTP := clonePayload(payload)
	noSMTP["email_verification_required"] = true
	noSMTP["smtp_enabled"] = false
	res = e.request(http.MethodPut, "/api/v1/admin/system-settings", admin, origin, noSMTP)
	if res.status != http.StatusBadRequest || res.code != apperr.CodeValidation {
		t.Fatalf("verification-without-smtp status=%d code=%d, want 400/%d", res.status, res.code, apperr.CodeValidation)
	}

	// Stale optimistic-lock version is rejected instead of last-write-wins.
	stale := clonePayload(payload)
	stale["version"] = int(version) + 7
	res = e.request(http.MethodPut, "/api/v1/admin/system-settings", admin, origin, stale)
	if res.status != http.StatusConflict || res.code != apperr.CodeSettingsConflict {
		t.Fatalf("stale version status=%d code=%d, want 409/%d", res.status, res.code, apperr.CodeSettingsConflict)
	}

	// Configure SMTP (with a secret) and require verification.
	const smtpSecret = "it-smtp-secret-do-not-leak"
	withSMTP := clonePayload(payload)
	withSMTP["smtp_enabled"] = true
	withSMTP["smtp_host"] = "127.0.0.1"
	withSMTP["smtp_port"] = 2525
	withSMTP["smtp_username"] = "it-smtp-user"
	withSMTP["smtp_password"] = smtpSecret
	withSMTP["smtp_from_email"] = "verify@example.com"
	withSMTP["smtp_from_name"] = "easy-admin IT"
	withSMTP["smtp_tls_mode"] = "starttls"
	withSMTP["email_verification_required"] = true
	withSMTP["registration_enabled"] = true
	withSMTP["username_login_enabled"] = true
	withSMTP["email_login_enabled"] = true
	withSMTP["registration_points"] = "12.3456"
	saved := e.request(http.MethodPut, "/api/v1/admin/system-settings", admin, origin, withSMTP)
	if saved.status != http.StatusOK {
		t.Fatalf("settings update status=%d code=%d body=%v", saved.status, saved.code, saved.body)
	}
	if saved.body["password_configured"] != true {
		t.Fatalf("password_configured = %v, want true", saved.body["password_configured"])
	}
	rawSaved, _ := json.Marshal(saved.body)
	if strings.Contains(strings.ToLower(string(rawSaved)), strings.ToLower(smtpSecret)) {
		t.Fatal("settings response leaked the SMTP secret")
	}
	if strings.Contains(strings.ToLower(string(rawSaved)), "smtp_password") {
		t.Fatal("settings response must not expose an smtp_password field")
	}

	// The stored value is ciphertext, never the plaintext.
	var stored string
	if err := e.primaryDB.GORM().Raw("SELECT smtp_password_encrypted FROM system_settings WHERE id = 1").Scan(&stored).Error; err != nil {
		t.Fatal(err)
	}
	if stored == "" || stored == smtpSecret || strings.Contains(stored, smtpSecret) {
		t.Fatalf("stored smtp password is not encrypted: %q", redact(stored))
	}

	// Registration now produces a pending account plus a one-time link.
	e.mailer.reset()
	reg := e.request(http.MethodPost, "/api/v1/auth/register", "", origin, map[string]any{
		"username": "it_verify_user",
		"email":    "it_verify_user@example.com",
		"password": "password-123456",
	})
	if reg.status != http.StatusCreated {
		t.Fatalf("register (verification required) status=%d code=%d body=%v", reg.status, reg.code, reg.body)
	}
	if reg.body["status"] != "pending_verification" {
		t.Fatalf("status = %v, want pending_verification", reg.body["status"])
	}
	if v, _ := reg.body["email_verified_at"].(string); v != "" {
		t.Fatalf("pending account must not be verified yet: %q", v)
	}
	if lv, _ := reg.body["level"].(map[string]any); lv == nil || lv["code"] != "default" {
		t.Fatalf("register response must carry the generated default level, got %v", reg.body["level"])
	}
	if e.mailer.count() != 1 {
		t.Fatalf("verification mail count = %d, want 1", e.mailer.count())
	}
	token := extractToken(e.t, e.mailer.last().Body)
	if token == "" {
		t.Fatal("verification email contained no token")
	}

	// Pending accounts cannot log in.
	if login := e.request(http.MethodPost, "/api/v1/auth/login", "", origin, map[string]any{
		"identifier": "it_verify_user", "password": "password-123456",
	}); login.status != http.StatusForbidden || login.code != apperr.CodeEmailNotVerified {
		t.Fatalf("pending login status=%d code=%d, want 403/%d", login.status, login.code, apperr.CodeEmailNotVerified)
	}

	// Resending invalidates the previous token.
	resend := e.request(http.MethodPost, "/api/v1/auth/resend-verification", "", origin, map[string]any{
		"email": "it_verify_user@example.com",
	})
	if resend.status != http.StatusOK {
		t.Fatalf("resend status=%d code=%d body=%v", resend.status, resend.code, resend.body)
	}
	newToken := extractToken(e.t, e.mailer.last().Body)
	if newToken == "" || newToken == token {
		t.Fatalf("resend must rotate the token (old %q new %q)", redact(token), redact(newToken))
	}
	if old := e.request(http.MethodPost, "/api/v1/auth/verify-email", "", origin, map[string]any{
		"email": "it_verify_user@example.com", "token": token,
	}); old.status != http.StatusBadRequest || old.code != apperr.CodeVerificationInvalid {
		t.Fatalf("superseded token status=%d code=%d, want 400/%d", old.status, old.code, apperr.CodeVerificationInvalid)
	}

	// The current token verifies exactly once.
	verify := e.request(http.MethodPost, "/api/v1/auth/verify-email", "", origin, map[string]any{
		"email": "it_verify_user@example.com", "token": newToken,
	})
	if verify.status != http.StatusOK {
		t.Fatalf("verify status=%d code=%d body=%v", verify.status, verify.code, verify.body)
	}
	replay := e.request(http.MethodPost, "/api/v1/auth/verify-email", "", origin, map[string]any{
		"email": "it_verify_user@example.com", "token": newToken,
	})
	if replay.status != http.StatusBadRequest || replay.code != apperr.CodeVerificationInvalid {
		t.Fatalf("verification replay status=%d code=%d, want 400/%d", replay.status, replay.code, apperr.CodeVerificationInvalid)
	}

	// Verified accounts can log in and receive the registration starting points.
	access, _ := e.loginUser(t, "it_verify_user", "password-123456", origin)
	me := e.request(http.MethodGet, "/api/v1/me", access, "", nil)
	if me.status != http.StatusOK {
		t.Fatalf("/me after verification status=%d", me.status)
	}
	if v, _ := me.body["points_balance"].(string); v != "12.3456" {
		t.Fatalf("registration starting points = %q, want 12.3456", v)
	}

	// Public settings still expose only the whitelist.
	pub := e.request(http.MethodGet, "/api/v1/public/settings", "", "", nil)
	pubRaw, _ := json.Marshal(pub.body)
	if strings.Contains(strings.ToLower(string(pubRaw)), "smtp") {
		t.Fatalf("public settings leaked smtp configuration: %s", pubRaw)
	}
	if pub.body["email_verification_required"] != true {
		t.Fatalf("public settings must advertise the verification switch: %s", pubRaw)
	}

	// Restore the migrated baseline so later tests stay deterministic.
	if err := e.primaryDB.GORM().Exec(
		"UPDATE system_settings SET registration_enabled = TRUE, username_login_enabled = TRUE, " +
			"email_login_enabled = TRUE, email_verification_required = FALSE, smtp_enabled = FALSE, " +
			"smtp_host = '', smtp_port = 0, smtp_username = '', smtp_password_encrypted = '', " +
			"smtp_from_email = '', smtp_from_name = '', smtp_tls_mode = 'starttls', " +
			"registration_points = 0.0000, version = version + 1 WHERE id = 1").Error; err != nil {
		t.Fatalf("restore settings baseline: %v", err)
	}
}

func TestFinanceRoleDeniedBusinessUserAPIsHTTP(t *testing.T) {
	e := newRealHTTPEnv(t)
	origin := "http://localhost:3000"
	e.seedAdmin(t, "it_http_finance", adminauth.RoleFinance)
	finance := e.adminToken(t, "it_http_finance", origin)

	for _, route := range []struct {
		method string
		path   string
	}{
		{http.MethodGet, "/api/v1/admin/users"},
		{http.MethodPost, "/api/v1/admin/users"},
		{http.MethodGet, "/api/v1/admin/user-levels"},
		{http.MethodPost, "/api/v1/admin/user-levels"},
		{http.MethodGet, "/api/v1/admin/system-settings"},
		{http.MethodPut, "/api/v1/admin/system-settings"},
	} {
		res := e.request(route.method, route.path, finance, origin, map[string]any{})
		if res.status != http.StatusForbidden {
			t.Fatalf("%s %s as finance status=%d code=%d, want 403", route.method, route.path, res.status, res.code)
		}
	}
}

func TestUserRefreshReplayAndCookieAttributesHTTP(t *testing.T) {
	e := newRealHTTPEnv(t)
	origin := "http://localhost:3000"
	e.registerUser(t, "it_replay_user", origin)

	login := e.request(http.MethodPost, "/api/v1/auth/login", "", origin, map[string]any{
		"identifier": "it_replay_user",
		"password":   "password-123456",
	})
	if login.status != http.StatusOK {
		t.Fatalf("login status=%d code=%d", login.status, login.code)
	}
	userCookie := findCookie(t, login.headers, "ea_user_refresh")
	if userCookie == nil {
		t.Fatal("login must set the business-user refresh cookie")
	}
	if !userCookie.HttpOnly {
		t.Fatal("user refresh cookie must be HttpOnly")
	}
	if userCookie.SameSite != http.SameSiteStrictMode {
		t.Fatalf("user refresh cookie SameSite = %v, want Strict", userCookie.SameSite)
	}
	if userCookie.Path != "/api/v1/auth" {
		t.Fatalf("user refresh cookie path = %q, want /api/v1/auth", userCookie.Path)
	}
	if userCookie.Value == "" {
		t.Fatal("user refresh cookie must carry a verifier")
	}

	// Admin login uses a different cookie name and path: no namespace overlap.
	e.seedAdmin(t, "it_http_cookie_super", adminauth.RoleSuperAdmin)
	adminLogin := e.request(http.MethodPost, "/api/v1/admin/auth/login", "", origin, map[string]any{
		"username": "it_http_cookie_super", "password": httpAdminPassword,
	})
	adminCookie := findCookie(t, adminLogin.headers, "ea_admin_refresh")
	if adminCookie == nil {
		t.Fatal("admin login must set the administrator refresh cookie")
	}
	if adminCookie.Path != "/api/v1/admin/auth" {
		t.Fatalf("admin refresh cookie path = %q", adminCookie.Path)
	}

	refreshWith := func(value string) envResponse {
		req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/refresh", nil)
		req.AddCookie(&http.Cookie{Name: "ea_user_refresh", Value: value})
		req.Header.Set("Origin", origin)
		req.RemoteAddr = nextHTTPAddr()
		rec := httptest.NewRecorder()
		e.router.ServeHTTP(rec, req)
		out := envResponse{status: rec.Code, headers: rec.Header()}
		var envelope struct {
			Code int            `json:"code"`
			Data map[string]any `json:"data"`
		}
		if rec.Body.Len() > 0 {
			_ = json.Unmarshal(rec.Body.Bytes(), &envelope)
			out.code = envelope.Code
			out.body = envelope.Data
		}
		return out
	}

	rotated := refreshWith(userCookie.Value)
	if rotated.status != http.StatusOK {
		t.Fatalf("refresh status=%d code=%d body=%v", rotated.status, rotated.code, rotated.body)
	}
	next := findCookie(t, rotated.headers, "ea_user_refresh")
	if next == nil || next.Value == userCookie.Value {
		t.Fatal("refresh must rotate the cookie verifier")
	}

	// Replaying the consumed verifier revokes the whole session...
	replay := refreshWith(userCookie.Value)
	if replay.status == http.StatusOK {
		t.Fatal("replayed refresh verifier must not succeed")
	}
	// ...so even the newest verifier is dead.
	after := refreshWith(next.Value)
	if after.status == http.StatusOK {
		t.Fatal("session must be revoked after a refresh replay")
	}
}

// --- helpers ---------------------------------------------------------------

func clonePayload(in map[string]any) map[string]any {
	out := make(map[string]any, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}

func findCookie(t *testing.T, headers http.Header, name string) *http.Cookie {
	t.Helper()
	for _, raw := range headers.Values("Set-Cookie") {
		c, err := http.ParseSetCookie(raw)
		if err != nil {
			t.Fatalf("parse set-cookie %q: %v", raw, err)
		}
		if c.Name == name {
			copy := *c
			return &copy
		}
	}
	return nil
}

func redact(v string) string {
	if len(v) <= 6 {
		return "***"
	}
	return v[:3] + "***"
}

// compile-time guard: the usersvc actor type stays usable from these tests.
var _ = usersvc.Actor{}
