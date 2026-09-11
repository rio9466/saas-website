package adminauth_test

import (
	"net/http"
	"testing"

	"github.com/rio9466/easy-admin/server/internal/domain/adminauth"
)

// TestChangePasswordInvalidatesAllOwnSessions proves that changing one's own
// password invalidates every own access and refresh session, not only the
// current one, and that only the new password can create a new session.
func TestChangePasswordInvalidatesAllOwnSessions(t *testing.T) {
	env := newHTTPEnv(t)
	env.resetRBAC(t)
	const (
		username    = "it_pwuser"
		password    = "it-password-1234"
		newPassword = "it-new-password-99"
	)
	env.createAdmin(t, username, password, []string{adminauth.RoleAdmin})

	tokenA, refreshA := env.login(t, username, password)
	tokenB, refreshB := env.login(t, username, password)

	res := env.do(t, http.MethodPost, "/api/v1/admin/me/password", tokenA, map[string]any{
		"current_password": password,
		"new_password":     newPassword,
	})
	if res.status != http.StatusOK {
		t.Fatalf("change password status=%d body=%v", res.status, res.body)
	}

	for name, token := range map[string]string{"sessionA": tokenA, "sessionB": tokenB} {
		me := env.me(t, token)
		if me.status != http.StatusUnauthorized {
			t.Fatalf("/me after password change with %s status=%d, want 401", name, me.status)
		}
	}
	for name, refresh := range map[string]string{"refreshA": refreshA, "refreshB": refreshB} {
		rr := env.refresh(t, refresh)
		if rr.status != http.StatusUnauthorized {
			t.Fatalf("refresh after password change with %s status=%d, want 401", name, rr.status)
		}
	}

	if res := env.loginExpect(t, username, password, http.StatusUnauthorized); res.status != http.StatusUnauthorized {
		t.Fatalf("login with old password status=%d, want 401", res.status)
	}
	tokenC, _ := env.login(t, username, newPassword)
	if me := env.me(t, tokenC); me.status != http.StatusOK {
		t.Fatalf("session created with new password rejected: %d", me.status)
	}
}

// TestResetAdministratorPasswordInvalidatesAllTargetSessions proves resetting
// another administrator's password invalidates every existing target access
// and refresh session, and that a session that a Redis revocation scan could
// not observe (simulated with an old auth epoch) is still rejected by the
// primary-database epoch check.
func TestResetAdministratorPasswordInvalidatesAllTargetSessions(t *testing.T) {
	env := newHTTPEnv(t)
	env.resetRBAC(t)

	const (
		actorUsername  = "it_reset_actor"
		actorPassword  = "it-actor-pass-1234"
		targetUsername = "it_reset_target"
		targetPassword = "it-target-pass-1234"
		newPassword    = "it-target-new-99"
	)
	env.createAdmin(t, actorUsername, actorPassword, []string{adminauth.RoleSuperAdmin})
	targetID := env.createAdmin(t, targetUsername, targetPassword, []string{adminauth.RoleAdmin})

	tokenA, refreshA := env.login(t, targetUsername, targetPassword)
	tokenB, refreshB := env.login(t, targetUsername, targetPassword)
	actorToken, _ := env.login(t, actorUsername, actorPassword)

	res := env.do(t, http.MethodPost, "/api/v1/admin/administrators/"+idString(targetID)+"/reset-password", actorToken, map[string]any{
		"new_password": newPassword,
	})
	if res.status != http.StatusOK {
		t.Fatalf("reset-password status=%d body=%v", res.status, res.body)
	}

	for name, token := range map[string]string{"targetSessionA": tokenA, "targetSessionB": tokenB} {
		if me := env.me(t, token); me.status != http.StatusUnauthorized {
			t.Fatalf("/me with %s after reset status=%d, want 401", name, me.status)
		}
	}
	for name, refresh := range map[string]string{"targetRefreshA": refreshA, "targetRefreshB": refreshB} {
		if rr := env.refresh(t, refresh); rr.status != http.StatusUnauthorized {
			t.Fatalf("refresh with %s after reset status=%d, want 401", name, rr.status)
		}
	}

	// Race proof: a session minted under the pre-reset epoch (0) and never
	// removed by Redis revocation must still fail every protected request.
	oldEpochAccess, oldEpochRefresh, _ := env.mintSessionForEpoch(t, targetID, 0)
	if me := env.me(t, oldEpochAccess); me.status != http.StatusUnauthorized {
		t.Fatalf("/me with old-epoch synthetic session status=%d, want 401", me.status)
	}
	if rr := env.refresh(t, oldEpochRefresh); rr.status != http.StatusUnauthorized {
		t.Fatalf("refresh with old-epoch synthetic session status=%d, want 401", rr.status)
	}

	if res := env.loginExpect(t, targetUsername, targetPassword, http.StatusUnauthorized); res.status != http.StatusUnauthorized {
		t.Fatalf("login with old password status=%d, want 401", res.status)
	}
	targetToken, _ := env.login(t, targetUsername, newPassword)
	if me := env.me(t, targetToken); me.status != http.StatusOK {
		t.Fatalf("login with reset password cannot reach /me: %d", me.status)
	}
}

// TestDisableAndReenableNeverResurrectsOldSessions proves disabling an
// administrator invalidates all sessions and re-enabling does not make any
// pre-disable session valid again.
func TestDisableAndReenableNeverResurrectsOldSessions(t *testing.T) {
	env := newHTTPEnv(t)
	env.resetRBAC(t)

	const (
		actorUsername  = "it_disable_actor"
		actorPassword  = "it-actor-pass-1234"
		targetUsername = "it_disable_target"
		targetPassword = "it-target-pass-1234"
	)
	env.createAdmin(t, actorUsername, actorPassword, []string{adminauth.RoleSuperAdmin})
	targetID := env.createAdmin(t, targetUsername, targetPassword, []string{adminauth.RoleAdmin})

	token, refresh := env.login(t, targetUsername, targetPassword)
	actorToken, _ := env.login(t, actorUsername, actorPassword)
	path := "/api/v1/admin/administrators/" + idString(targetID)

	if res := env.do(t, http.MethodPost, path+"/disable", actorToken, nil); res.status != http.StatusOK {
		t.Fatalf("disable status=%d body=%v", res.status, res.body)
	}
	if me := env.me(t, token); me.status != http.StatusUnauthorized {
		t.Fatalf("/me after disable status=%d, want 401", me.status)
	}
	if rr := env.refresh(t, refresh); rr.status == http.StatusOK {
		t.Fatal("refresh after disable unexpectedly succeeded")
	}

	if res := env.do(t, http.MethodPost, path+"/enable", actorToken, nil); res.status != http.StatusOK {
		t.Fatalf("re-enable status=%d body=%v", res.status, res.body)
	}
	if me := env.me(t, token); me.status != http.StatusUnauthorized {
		t.Fatalf("/me after re-enable with old token status=%d, want 401 (old sessions must stay dead)", me.status)
	}
	if rr := env.refresh(t, refresh); rr.status != http.StatusUnauthorized {
		t.Fatalf("refresh after re-enable with old cookie status=%d, want 401", rr.status)
	}

	// Even an un-revoked pre-disable session (epoch 0; current epoch is 2)
	// cannot become valid after re-enable.
	oldEpochAccess, oldEpochRefresh, _ := env.mintSessionForEpoch(t, targetID, 0)
	if me := env.me(t, oldEpochAccess); me.status != http.StatusUnauthorized {
		t.Fatalf("/me with pre-disable synthetic session after re-enable status=%d, want 401", me.status)
	}
	if rr := env.refresh(t, oldEpochRefresh); rr.status != http.StatusUnauthorized {
		t.Fatalf("refresh with pre-disable synthetic session after re-enable status=%d, want 401", rr.status)
	}

	// A fresh login after re-enable works normally.
	_, _ = env.login(t, targetUsername, targetPassword)
}
