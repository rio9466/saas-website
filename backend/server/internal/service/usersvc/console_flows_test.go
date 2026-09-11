package usersvc_test

import (
	"fmt"
	"net/url"
	"strings"
	"testing"

	"github.com/rio9466/easy-admin/server/internal/domain/apperr"
	usersvc "github.com/rio9466/easy-admin/server/internal/service/usersvc"
)

// assertSelfContainedLink checks that an email body carries the route path plus
// the url-encoded recipient and a token, so the link works from any device.
func assertSelfContainedLink(t *testing.T, body, path, email string) {
	t.Helper()
	if !strings.Contains(body, path) {
		t.Fatalf("email body missing path %q", path)
	}
	wantEmail := "email=" + url.QueryEscape(email)
	if !strings.Contains(body, wantEmail) {
		t.Fatalf("email body missing %q", wantEmail)
	}
	if extractToken(t, body) == "" {
		t.Fatal("email body missing token")
	}
}

// TestVerificationAndResetLinksCarryEmailAndToken locks the contract §5.7 link
// shape for both verification and reset emails.
func TestVerificationAndResetLinksCarryEmailAndToken(t *testing.T) {
	e := newSvcEnv(t)
	defer func() {
		e.cleanUserData(t)
		e.resetSettingsBaseline(t)
	}()
	e.seedEnabledSMTP(t)

	e.createActiveUser(t, "it_link_user", "password-123456")
	if e.mailer.count() == 0 {
		t.Fatal("verification email must be produced")
	}
	assertSelfContainedLink(t, e.mailer.last().Body, "/verify-email", "it_link_user@example.com")

	e.mailer.reset()
	if err := e.svc.ForgotPassword(e.ctx, e.anonymousActor(), "it_link_user@example.com"); err != nil {
		t.Fatalf("forgot-password: %v", err)
	}
	if e.mailer.count() != 1 {
		t.Fatalf("reset mail count = %d, want 1", e.mailer.count())
	}
	assertSelfContainedLink(t, e.mailer.last().Body, "/reset-password", "it_link_user@example.com")
}

// TestForgotRateLimitFailsClosed covers the per-IP fixed window.
func TestForgotRateLimitFailsClosed(t *testing.T) {
	e := newSvcEnv(t)
	actor := usersvc.Actor{SourceIP: "10.202.0.1", UserAgent: "test"}
	limited := false
	for i := 0; i < 8; i++ {
		err := e.svc.ForgotPassword(e.ctx, actor, fmt.Sprintf("ghost%d@example.com", i))
		switch {
		case errCode(err) == apperr.CodeRateLimited:
			limited = true
		case err != nil:
			t.Fatalf("unexpected error: %v", err)
		}
		if limited {
			break
		}
	}
	if !limited {
		t.Fatal("forgot-password never hit the per-IP rate limit")
	}
}

// TestResetRateLimitFailsClosed covers the per-IP fixed window.
func TestResetRateLimitFailsClosed(t *testing.T) {
	e := newSvcEnv(t)
	e.cleanUserData(t)
	e.createActiveUser(t, "it_reset_rl", "password-123456")

	actor := usersvc.Actor{SourceIP: "10.203.0.1", UserAgent: "test"}
	limited := false
	for i := 0; i < 15; i++ {
		err := e.svc.ResetPassword(e.ctx, actor, "it_reset_rl@example.com", "wrong-token", "new-password-123")
		switch {
		case errCode(err) == apperr.CodeRateLimited:
			limited = true
		case errCode(err) != apperr.CodeVerificationInvalid:
			t.Fatalf("unexpected error: %v (code %d)", err, errCode(err))
		}
		if limited {
			break
		}
	}
	if !limited {
		t.Fatal("reset-password never hit the per-IP rate limit")
	}
}
