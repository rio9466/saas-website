package usersvc_test

import (
	"net/http"
	"strings"
	"testing"

	"github.com/rio9466/easy-admin/server/internal/domain/apperr"
)

// hasClearedUserRefreshCookie reports whether a response clears the business
// user refresh cookie (Max-Age=0).
func hasClearedUserRefreshCookie(res envResponse) bool {
	for _, h := range res.headers.Values("Set-Cookie") {
		if strings.Contains(h, "ea_user_refresh") && strings.Contains(h, "Max-Age=0") {
			return true
		}
	}
	return false
}

// TestUserConsoleProfileAndPasswordHTTP covers PATCH /me (nickname/avatar only),
// the change-password flow, session revocation, and the validation/current-
// password error codes through the real router.
func TestUserConsoleProfileAndPasswordHTTP(t *testing.T) {
	e := newRealHTTPEnv(t)
	origin := "http://localhost:3000"
	e.registerUser(t, "it_console", origin)
	access, _ := e.loginUser(t, "it_console", "password-123456", origin)

	before := e.request(http.MethodGet, "/api/v1/me", access, "", nil)
	if before.status != http.StatusOK {
		t.Fatalf("me before patch status=%d", before.status)
	}
	wantPoints, _ := before.body["points_balance"].(string)
	wantLevel, _ := before.body["level"].(map[string]any)

	// PATCH binds only nickname/avatar_url; hostile fields are ignored.
	patch := e.request(http.MethodPatch, "/api/v1/me", access, "", map[string]any{
		"nickname":       "Console Nick",
		"avatar_url":     "/media/avatar.png",
		"email":          "attacker@example.com",
		"username":       "attacker",
		"points_balance": "999999.0000",
		"status":         "disabled",
		"level":          map[string]any{"id": "1", "code": "hacked"},
	})
	if patch.status != http.StatusOK {
		t.Fatalf("patch status=%d code=%d body=%v", patch.status, patch.code, patch.body)
	}
	if patch.body["nickname"] != "Console Nick" {
		t.Fatalf("nickname = %v", patch.body["nickname"])
	}
	if patch.body["avatar_url"] != "/media/avatar.png" {
		t.Fatalf("avatar_url = %v", patch.body["avatar_url"])
	}
	if patch.body["email"] != "it_console@example.com" {
		t.Fatalf("email must be unchanged, got %v", patch.body["email"])
	}
	if patch.body["username"] != "it_console" {
		t.Fatalf("username must be unchanged, got %v", patch.body["username"])
	}
	if patch.body["status"] != "active" {
		t.Fatalf("status must be unchanged, got %v", patch.body["status"])
	}
	if got, _ := patch.body["points_balance"].(string); got != wantPoints {
		t.Fatalf("points_balance must be unchanged: %q -> %q", wantPoints, got)
	}
	patchedLevel, _ := patch.body["level"].(map[string]any)
	if wantLevel != nil && patchedLevel["code"] != wantLevel["code"] {
		t.Fatalf("level must be unchanged: %v -> %v", wantLevel["code"], patchedLevel["code"])
	}

	// A non-http(s), non-relative avatar is rejected.
	badAvatar := e.request(http.MethodPatch, "/api/v1/me", access, "", map[string]any{
		"avatar_url": "javascript:alert(1)",
	})
	if badAvatar.status != http.StatusBadRequest || badAvatar.code != apperr.CodeValidation {
		t.Fatalf("bad avatar status=%d code=%d, want 400/10001", badAvatar.status, badAvatar.code)
	}

	// Wrong current password -> 40005; too-short new password -> 10001.
	wrong := e.request(http.MethodPost, "/api/v1/me/password", access, "", map[string]any{
		"current_password": "not-the-password",
		"new_password":     "new-password-987",
	})
	if wrong.status != http.StatusBadRequest || wrong.code != apperr.CodeInvalidPassword {
		t.Fatalf("wrong current password status=%d code=%d, want 400/%d", wrong.status, wrong.code, apperr.CodeInvalidPassword)
	}
	short := e.request(http.MethodPost, "/api/v1/me/password", access, "", map[string]any{
		"current_password": "password-123456",
		"new_password":     "1234567",
	})
	if short.status != http.StatusBadRequest || short.code != apperr.CodeValidation {
		t.Fatalf("short new password status=%d code=%d, want 400/10001", short.status, short.code)
	}

	// Successful change: 200 {}, refresh cookie cleared, old access token gone.
	changed := e.request(http.MethodPost, "/api/v1/me/password", access, "", map[string]any{
		"current_password": "password-123456",
		"new_password":     "new-password-987",
	})
	if changed.status != http.StatusOK {
		t.Fatalf("change password status=%d code=%d body=%v", changed.status, changed.code, changed.body)
	}
	if len(changed.body) != 0 {
		t.Fatalf("change password data = %v, want {}", changed.body)
	}
	if !hasClearedUserRefreshCookie(changed) {
		t.Fatalf("change password must clear the refresh cookie: %v", changed.headers.Values("Set-Cookie"))
	}
	if after := e.request(http.MethodGet, "/api/v1/me", access, "", nil); after.status != http.StatusUnauthorized {
		t.Fatalf("/me with revoked access status=%d, want 401", after.status)
	}

	// The new password works; the old one does not.
	e.loginUser(t, "it_console", "new-password-987", origin)
	old := e.request(http.MethodPost, "/api/v1/auth/login", "", origin, map[string]any{
		"identifier": "it_console", "password": "password-123456",
	})
	if old.status != http.StatusUnauthorized {
		t.Fatalf("old password login status=%d, want 401", old.status)
	}
}

// TestMyPointTransactionsHTTP covers "only my own ledger" and pagination.
func TestMyPointTransactionsHTTP(t *testing.T) {
	e := newRealHTTPEnv(t)
	origin := "http://localhost:3000"
	e.registerUser(t, "it_points_mine", origin)
	e.registerUser(t, "it_points_other", origin)
	e.seedAdmin(t, "it_console_points_super", "super_admin")
	admin := e.adminToken(t, "it_console_points_super", origin)

	mineAccess, _ := e.loginUser(t, "it_points_mine", "password-123456", origin)
	me := e.request(http.MethodGet, "/api/v1/me", mineAccess, "", nil)
	myID, _ := me.body["id"].(string)
	if myID == "" {
		t.Fatal("no user id")
	}
	other := e.request(http.MethodGet, "/api/v1/admin/users?q=it_points_other", admin, origin, nil)
	otherItems, _ := other.body["items"].([]any)
	if len(otherItems) == 0 {
		t.Fatal("other user not found")
	}
	otherID, _ := otherItems[0].(map[string]any)["id"].(string)

	adjust := func(userID, delta, key, reason string) {
		t.Helper()
		res := e.request(http.MethodPost, "/api/v1/admin/users/"+userID+"/points-adjust", admin, origin, map[string]any{
			"points_delta": delta, "consumption_delta": "0.0000",
			"reason": reason, "idempotency_key": key,
		})
		if res.status != http.StatusOK {
			t.Fatalf("adjust %s status=%d code=%d body=%v", userID, res.status, res.code, res.body)
		}
	}
	adjust(myID, "10.0000", "it-console-mine-1", "console-mine-1")
	adjust(myID, "5.0000", "it-console-mine-2", "console-mine-2")
	adjust(otherID, "77.0000", "it-console-other-1", "console-other-secret")

	// Page 1 of size 1 returns the newest entry; only my ledger is visible.
	page1 := e.request(http.MethodGet, "/api/v1/me/point-transactions?page=1&page_size=1", mineAccess, "", nil)
	if page1.status != http.StatusOK {
		t.Fatalf("point-transactions status=%d code=%d body=%v", page1.status, page1.code, page1.body)
	}
	if total, _ := page1.body["total"].(float64); total != 2 {
		t.Fatalf("total = %v, want 2 (only my ledger)", page1.body["total"])
	}
	if ps, _ := page1.body["page_size"].(float64); ps != 1 {
		t.Fatalf("page_size = %v, want 1", page1.body["page_size"])
	}
	items, _ := page1.body["items"].([]any)
	if len(items) != 1 {
		t.Fatalf("page1 items = %d, want 1", len(items))
	}
	first, _ := items[0].(map[string]any)
	if first["reason"] != "console-mine-2" {
		t.Fatalf("newest-first order violated: %v", first["reason"])
	}
	if delta, _ := first["points_delta"].(string); delta != "5.0000" {
		t.Fatalf("points_delta = %q, want fixed 5.0000", delta)
	}

	page2 := e.request(http.MethodGet, "/api/v1/me/point-transactions?page=2&page_size=1", mineAccess, "", nil)
	items2, _ := page2.body["items"].([]any)
	if len(items2) != 1 {
		t.Fatalf("page2 items = %d, want 1", len(items2))
	}
	second, _ := items2[0].(map[string]any)
	if second["reason"] != "console-mine-1" {
		t.Fatalf("page2 reason = %v, want console-mine-1", second["reason"])
	}

	// The other user's ledger never leaks in.
	all := e.request(http.MethodGet, "/api/v1/me/point-transactions?page=1&page_size=20", mineAccess, "", nil)
	allItems, _ := all.body["items"].([]any)
	for _, raw := range allItems {
		if reason, _ := raw.(map[string]any)["reason"].(string); strings.Contains(reason, "other") {
			t.Fatalf("leaked another user's transaction: %v", reason)
		}
	}
}

// TestForgotResetPasswordHTTP covers existence-uniformity, the self-contained
// reset link, one-time consumption, session revocation, and 40016 replay.
func TestForgotResetPasswordHTTP(t *testing.T) {
	e := newRealHTTPEnv(t)
	origin := "http://localhost:3000"
	e.registerUser(t, "it_forgot", origin)
	access, _ := e.loginUser(t, "it_forgot", "password-123456", origin)

	// Unknown and existing addresses return an identical response.
	e.mailer.reset()
	missing := e.request(http.MethodPost, "/api/v1/auth/forgot-password", "", "", map[string]any{
		"email": "nobody-here@example.com",
	})
	e.mailer.reset()
	existing := e.request(http.MethodPost, "/api/v1/auth/forgot-password", "", "", map[string]any{
		"email": "it_forgot@example.com",
	})
	if missing.status != existing.status || missing.code != existing.code {
		t.Fatalf("forgot responses differ: missing=%d/%d existing=%d/%d", missing.status, missing.code, existing.status, existing.code)
	}
	if existing.status != http.StatusOK {
		t.Fatalf("forgot status=%d code=%d body=%v", existing.status, existing.code, existing.body)
	}
	if e.mailer.count() != 1 {
		t.Fatalf("only an existing address may send mail, count=%d", e.mailer.count())
	}

	body := e.mailer.last().Body
	if !strings.Contains(body, "/reset-password") {
		t.Fatalf("reset link missing route: %s", body)
	}
	if !strings.Contains(body, "email=it_forgot%40example.com") {
		t.Fatalf("reset link must carry the encoded email: %s", body)
	}
	token := extractToken(t, body)
	if token == "" {
		t.Fatal("reset email carried no token")
	}

	// Wrong token is rejected.
	bad := e.request(http.MethodPost, "/api/v1/auth/reset-password", "", "", map[string]any{
		"email": "it_forgot@example.com", "token": "deadbeef", "new_password": "reset-password-1",
	})
	if bad.status != http.StatusBadRequest || bad.code != apperr.CodeVerificationInvalid {
		t.Fatalf("wrong token status=%d code=%d, want 400/%d", bad.status, bad.code, apperr.CodeVerificationInvalid)
	}

	// Correct token resets the password and revokes the prior session.
	reset := e.request(http.MethodPost, "/api/v1/auth/reset-password", "", "", map[string]any{
		"email": "it_forgot@example.com", "token": token, "new_password": "reset-password-1",
	})
	if reset.status != http.StatusOK {
		t.Fatalf("reset status=%d code=%d body=%v", reset.status, reset.code, reset.body)
	}
	if me := e.request(http.MethodGet, "/api/v1/me", access, "", nil); me.status != http.StatusUnauthorized {
		t.Fatalf("/me after reset status=%d, want 401", me.status)
	}

	// Replay of the same token is rejected (single-use).
	replay := e.request(http.MethodPost, "/api/v1/auth/reset-password", "", "", map[string]any{
		"email": "it_forgot@example.com", "token": token, "new_password": "reset-password-2",
	})
	if replay.status != http.StatusBadRequest || replay.code != apperr.CodeVerificationInvalid {
		t.Fatalf("replay status=%d code=%d, want 400/%d", replay.status, replay.code, apperr.CodeVerificationInvalid)
	}

	// The new password works.
	e.loginUser(t, "it_forgot", "reset-password-1", origin)
}

// TestForgotPasswordRateLimitedHTTP proves the per-address window fails closed
// with 42901.
func TestForgotPasswordRateLimitedHTTP(t *testing.T) {
	e := newRealHTTPEnv(t)
	e.registerUser(t, "it_forgot_rl", "http://localhost:3000")
	for i := 0; i < 6; i++ {
		res := e.request(http.MethodPost, "/api/v1/auth/forgot-password", "", "", map[string]any{
			"email": "it_forgot_rl@example.com",
		})
		if res.code == apperr.CodeRateLimited {
			return
		}
	}
	t.Fatal("forgot-password never hit the rate limit (fail-closed required)")
}
