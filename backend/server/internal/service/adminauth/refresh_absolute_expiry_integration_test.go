package adminauth_test

import (
	"net/http"
	"testing"
	"time"

	"github.com/rio9466/easy-admin/server/internal/domain/adminauth"
	platformauth "github.com/rio9466/easy-admin/server/internal/platform/auth"
)

// TestRefreshRotationsPreserveAbsoluteExpiry proves a session has one absolute
// refresh expiry (set at login) and that repeated refresh rotations neither
// move the Redis absolute expiry nor mint rotated JWT/cookie lifetimes past
// the original login expiry.
func TestRefreshRotationsPreserveAbsoluteExpiry(t *testing.T) {
	env := newHTTPEnv(t)
	env.resetRBAC(t)

	const (
		username = "it_abs_user"
		password = "it-abs-pass-1234"
	)
	env.createAdmin(t, username, password, []string{adminauth.RoleAdmin})

	access, cookie := env.login(t, username, password)
	if cookie == "" {
		t.Fatal("login returned no refresh cookie")
	}

	accessClaims, err := env.tokens.ParseAndValidate(access, platformauth.TokenTypeAccess)
	if err != nil {
		t.Fatalf("parse access: %v", err)
	}
	sid := accessClaims.SessionID

	loginRefreshClaims, err := env.tokens.ParseAndValidate(cookie, platformauth.TokenTypeRefresh)
	if err != nil {
		t.Fatalf("parse login refresh: %v", err)
	}
	absolute := loginRefreshClaims.ExpiresAt.Time.UTC()
	if absolute.After(time.Now().UTC().Add(platformauth.RefreshTokenTTL)) {
		t.Fatalf("login refresh expiry %v exceeds 30-day window", absolute)
	}

	prevMaxAge := env.maxAgeFromLogin(t, cookie, absolute)
	prev := cookie
	for i := 1; i <= 4; i++ {
		rr := env.refresh(t, prev)
		if rr.status != http.StatusOK {
			t.Fatalf("refresh %d status=%d body=%v", i, rr.status, rr.body)
		}
		if rr.cookie == "" {
			t.Fatalf("refresh %d returned no rotated cookie", i)
		}
		claims, err := env.tokens.ParseAndValidate(rr.cookie, platformauth.TokenTypeRefresh)
		if err != nil {
			t.Fatalf("parse rotated refresh %d: %v", i, err)
		}
		got := claims.ExpiresAt.Time.UTC()
		if !got.Equal(absolute) {
			t.Fatalf("rotated refresh %d exp = %v, want absolute %v (must not slide)", i, got, absolute)
		}
		if got.After(absolute) {
			t.Fatalf("rotated refresh %d exp %v extends beyond absolute %v", i, got, absolute)
		}
		if rr.maxAge > prevMaxAge+2 {
			t.Fatalf("refresh %d cookie Max-Age increased from %d to %d", i, prevMaxAge, rr.maxAge)
		}
		prevMaxAge = rr.maxAge
		prev = rr.cookie
	}

	// The Redis session still carries the login-time absolute expiry.
	sess, err := env.sessions.Get(env.ctx, sid)
	if err != nil {
		t.Fatalf("session missing after rotations: %v", err)
	}
	if !sess.ExpiresAt.Equal(absolute) {
		t.Fatalf("session absolute expiry after rotations = %v, want %v", sess.ExpiresAt, absolute)
	}
}

// maxAgeFromLogin returns the cookie Max-Age the server would set for a token
// expiring at absolute, used to sanity-check the login cookie lifetime.
func (e *httpEnv) maxAgeFromLogin(t *testing.T, refreshCookie string, absolute time.Time) int {
	t.Helper()
	claims, err := e.tokens.ParseAndValidate(refreshCookie, platformauth.TokenTypeRefresh)
	if err != nil {
		t.Fatalf("parse refresh for max-age: %v", err)
	}
	remain := time.Until(claims.ExpiresAt.Time.UTC())
	if remain <= 0 {
		t.Fatal("login refresh already expired")
	}
	return int(remain.Seconds())
}
