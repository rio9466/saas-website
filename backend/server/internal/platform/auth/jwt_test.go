package auth

import (
	"errors"
	"testing"
	"time"
)

func TestParseAndValidateRejectsWrongType(t *testing.T) {
	t.Parallel()

	svc, err := NewTokenService(JWTConfig{
		Secret:   "unit-test-secret-at-least-32-bytes!!",
		Issuer:   "easy-admin",
		Audience: "easy-admin-admin",
	})
	if err != nil {
		t.Fatalf("NewTokenService: %v", err)
	}

	access, _, err := svc.IssueAccess("1", "sid-1")
	if err != nil {
		t.Fatalf("IssueAccess: %v", err)
	}
	refresh, _, err := svc.IssueRefresh("1", "sid-1")
	if err != nil {
		t.Fatalf("IssueRefresh: %v", err)
	}

	if _, err := svc.ParseAndValidate(access, TokenTypeRefresh); !errors.Is(err, ErrTokenTypeMismatch) {
		t.Fatalf("access as refresh = %v, want %v", err, ErrTokenTypeMismatch)
	}
	if _, err := svc.ParseAndValidate(refresh, TokenTypeAccess); !errors.Is(err, ErrTokenTypeMismatch) {
		t.Fatalf("refresh as access = %v, want %v", err, ErrTokenTypeMismatch)
	}

	claims, err := svc.ParseAndValidate(access, TokenTypeAccess)
	if err != nil {
		t.Fatalf("ParseAndValidate access: %v", err)
	}
	if claims.TokenType != TokenTypeAccess || claims.SessionID != "sid-1" || claims.Subject != "1" {
		t.Fatalf("unexpected claims: %+v", claims)
	}
	if claims.ExpiresAt.Time.Sub(claims.IssuedAt.Time) != AccessTokenTTL {
		t.Fatalf("access TTL = %v, want %v", claims.ExpiresAt.Time.Sub(claims.IssuedAt.Time), AccessTokenTTL)
	}

	refreshClaims, err := svc.ParseAndValidate(refresh, TokenTypeRefresh)
	if err != nil {
		t.Fatalf("ParseAndValidate refresh: %v", err)
	}
	if refreshClaims.ExpiresAt.Time.Sub(refreshClaims.IssuedAt.Time) != RefreshTokenTTL {
		t.Fatalf("refresh TTL = %v, want %v", refreshClaims.ExpiresAt.Time.Sub(refreshClaims.IssuedAt.Time), RefreshTokenTTL)
	}
}

func TestParseAndValidateRejectsTamperedToken(t *testing.T) {
	t.Parallel()

	svc, err := NewTokenService(JWTConfig{
		Secret:   "unit-test-secret-at-least-32-bytes!!",
		Issuer:   "easy-admin",
		Audience: "easy-admin-admin",
	})
	if err != nil {
		t.Fatalf("NewTokenService: %v", err)
	}
	token, _, err := svc.IssueAccess("1", "sid-1")
	if err != nil {
		t.Fatalf("IssueAccess: %v", err)
	}
	tampered := token + "x"
	if _, err := svc.ParseAndValidate(tampered, TokenTypeAccess); !errors.Is(err, ErrTokenInvalid) {
		t.Fatalf("tampered = %v, want %v", err, ErrTokenInvalid)
	}
}

func TestIssueAccessUsesFixedNow(t *testing.T) {
	t.Parallel()

	svc, err := NewTokenService(JWTConfig{
		Secret:   "unit-test-secret-at-least-32-bytes!!",
		Issuer:   "easy-admin",
		Audience: "easy-admin-admin",
	})
	if err != nil {
		t.Fatalf("NewTokenService: %v", err)
	}
	fixed := time.Date(2026, 9, 6, 0, 0, 0, 0, time.UTC)
	svc.now = func() time.Time { return fixed }

	_, claims, err := svc.IssueAccess("42", "sess")
	if err != nil {
		t.Fatalf("IssueAccess: %v", err)
	}
	if !claims.IssuedAt.Time.Equal(fixed) {
		t.Fatalf("iat = %v, want %v", claims.IssuedAt.Time, fixed)
	}
	if !claims.NotBefore.Time.Equal(fixed) {
		t.Fatalf("nbf = %v, want %v", claims.NotBefore.Time, fixed)
	}
	if !claims.ExpiresAt.Time.Equal(fixed.Add(AccessTokenTTL)) {
		t.Fatalf("exp = %v, want %v", claims.ExpiresAt.Time, fixed.Add(AccessTokenTTL))
	}
}

// TestIssueRefreshExpiringUsesAbsoluteExpiry proves rotated refresh JWTs are
// minted with the session's absolute expiry and never extend past it.
func TestIssueRefreshExpiringUsesAbsoluteExpiry(t *testing.T) {
	t.Parallel()

	svc, err := NewTokenService(JWTConfig{
		Secret:   "unit-test-secret-at-least-32-bytes!!",
		Issuer:   "easy-admin",
		Audience: "easy-admin-admin",
	})
	if err != nil {
		t.Fatalf("NewTokenService: %v", err)
	}
	fixed := time.Date(2026, 9, 6, 0, 0, 0, 0, time.UTC)
	svc.now = func() time.Time { return fixed }

	// Original login refresh: exactly 30 days after issue.
	loginRefresh, loginClaims, err := svc.IssueRefresh("42", "sess-abs")
	if err != nil {
		t.Fatalf("IssueRefresh: %v", err)
	}
	absolute := loginClaims.ExpiresAt.Time.UTC()
	if !absolute.Equal(fixed.Add(RefreshTokenTTL)) {
		t.Fatalf("login refresh exp = %v, want %v", absolute, fixed.Add(RefreshTokenTTL))
	}

	// Simulate repeated rotations from later moments; each token must expire at
	// the same absolute instant, not now+30d.
	for _, daysLater := range []int{1, 10, 29} {
		rotated := fixed.Add(time.Duration(daysLater) * 24 * time.Hour)
		svc.now = func() time.Time { return rotated }
		_, rotatedClaims, err := svc.IssueRefreshExpiring("42", "sess-abs", absolute)
		if err != nil {
			t.Fatalf("IssueRefreshExpiring at day %d: %v", daysLater, err)
		}
		if !rotatedClaims.ExpiresAt.Time.Equal(absolute) {
			t.Fatalf("rotated exp at day %d = %v, want absolute %v", daysLater, rotatedClaims.ExpiresAt.Time, absolute)
		}
		if rotatedClaims.ExpiresAt.Time.After(absolute) {
			t.Fatalf("rotated token exp %v extends beyond absolute %v", rotatedClaims.ExpiresAt.Time, absolute)
		}
	}
	_ = loginRefresh
}

// TestIssueRefreshExpiringRejectsNonPositiveAndOversizedLifetimes proves
// expiring issuance rejects a past or non-positive expiry and refuses to mint a
// token that would outlive the product refresh window.
func TestIssueRefreshExpiringRejectsNonPositiveAndOversizedLifetimes(t *testing.T) {
	t.Parallel()

	svc, err := NewTokenService(JWTConfig{
		Secret:   "unit-test-secret-at-least-32-bytes!!",
		Issuer:   "easy-admin",
		Audience: "easy-admin-admin",
	})
	if err != nil {
		t.Fatalf("NewTokenService: %v", err)
	}
	fixed := time.Date(2026, 9, 6, 12, 0, 0, 0, time.UTC)
	svc.now = func() time.Time { return fixed }

	if _, _, err := svc.IssueRefreshExpiring("1", "s", fixed); !errors.Is(err, ErrTokenInvalid) {
		t.Fatalf("expiry at now = %v, want %v", err, ErrTokenInvalid)
	}
	if _, _, err := svc.IssueRefreshExpiring("1", "s", fixed.Add(-time.Minute)); !errors.Is(err, ErrTokenInvalid) {
		t.Fatalf("expiry in past = %v, want %v", err, ErrTokenInvalid)
	}
	if _, _, err := svc.IssueRefreshExpiring("1", "s", fixed.Add(RefreshTokenTTL+time.Hour)); !errors.Is(err, ErrTokenInvalid) {
		t.Fatalf("expiry beyond window = %v, want %v", err, ErrTokenInvalid)
	}
}
