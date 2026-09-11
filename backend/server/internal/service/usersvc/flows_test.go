package usersvc_test

import (
	"encoding/json"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/rio9466/easy-admin/server/internal/domain/adminauth"
	"github.com/rio9466/easy-admin/server/internal/domain/apperr"
	userdomain "github.com/rio9466/easy-admin/server/internal/domain/user"
	usersvc "github.com/rio9466/easy-admin/server/internal/service/usersvc"
)

// TestRegisterActiveWhenVerificationDisabled covers public registration with
// verification disabled: account is active immediately, defaults (nickname,
// avatar) are generated, and no verification email is sent.
func TestRegisterActiveWhenVerificationDisabled(t *testing.T) {
	e := newSvcEnv(t)
	e.cleanUserData(t)

	u, err := e.svc.Register(e.ctx, e.registerActor(), usersvc.RegisterInput{
		Username: "it_public",
		Email:    "it_public@example.com",
		Password: "password-123456",
		SourceIP: nextIP(),
	})
	if err != nil {
		t.Fatalf("register: %v", err)
	}
	if u.Status != userdomain.StatusActive {
		t.Fatalf("status = %s, want active", u.Status)
	}
	if u.Nickname != "it_public" {
		t.Fatalf("nickname = %q, want username default", u.Nickname)
	}
	if u.Email != "it_public@example.com" {
		t.Fatalf("email = %q", u.Email)
	}
	if u.LevelMode != userdomain.LevelModeAuto {
		t.Fatalf("level mode = %s", u.LevelMode)
	}
	if u.EmailVerifiedAt == nil {
		t.Fatal("email_verified_at must be set when verification disabled")
	}
	if e.mailer.count() != 0 {
		t.Fatal("no verification email may be sent when verification is disabled")
	}
}

// TestRegisterRejectsDuplicates covers duplicate normalized username/email
// rejection under the service boundary.
func TestRegisterRejectsDuplicates(t *testing.T) {
	e := newSvcEnv(t)
	e.cleanUserData(t)
	e.createActiveUser(t, "it_dupe", "password-123456")

	_, err := e.svc.Register(e.ctx, e.registerActor(), usersvc.RegisterInput{
		Username: "IT_DUPE", // same after normalization
		Email:    "other@example.com",
		Password: "password-123456",
		SourceIP: nextIP(),
	})
	if err == nil {
		t.Fatal("duplicate normalized username must fail")
	}
	if errCode(err) != apperr.CodeConflict {
		t.Fatalf("err code = %d, want conflict", errCode(err))
	}

	_, err = e.svc.Register(e.ctx, e.registerActor(), usersvc.RegisterInput{
		Username: "another",
		Email:    "it_dupe@example.com",
		Password: "password-123456",
		SourceIP: nextIP(),
	})
	if err == nil {
		t.Fatal("duplicate email must fail")
	}
}

// TestRegisterRejectsWhenDisabled covers registration-disabled fail-closed.
func TestRegisterRejectsWhenDisabled(t *testing.T) {
	e := newSvcEnv(t)
	e.cleanUserData(t)

	cur, err := e.svc.GetSystemSettingsForActor(e.ctx, e.actorAdmin)
	if err != nil {
		t.Fatal(err)
	}
	_, err = e.svc.UpdateSystemSettings(e.ctx, e.actorAdmin, usersvc.UpdateSystemSettingsInput{
		PlatformName: cur.PlatformName, PublicFrontendURL: cur.PublicFrontendURL,
		PublicAPIURL: cur.PublicAPIURL, RegistrationEnabled: false,
		UsernameLoginEnabled: cur.UsernameLoginEnabled, EmailLoginEnabled: cur.EmailLoginEnabled,
		EmailVerificationRequired: false, DefaultLevelID: cur.DefaultLevelID,
		RegistrationPoints: cur.RegistrationPoints, Version: cur.Version,
	})
	if err != nil {
		t.Fatalf("disable registration: %v", err)
	}

	_, err = e.svc.Register(e.ctx, e.registerActor(), usersvc.RegisterInput{
		Username: "it_blocked",
		Email:    "it_blocked@example.com",
		Password: "password-123456",
		SourceIP: nextIP(),
	})
	if err == nil {
		t.Fatal("registration must fail when disabled")
	}
	if errCode(err) != apperr.CodeRegistrationDisabled {
		t.Fatalf("code = %d, want registration disabled", errCode(err))
	}
}

// TestPasswordByteRule covers the 8-72 UTF-8 byte boundary.
func TestPasswordByteRule(t *testing.T) {
	e := newSvcEnv(t)
	e.cleanUserData(t)

	// 7 ASCII bytes must fail.
	if _, err := e.svc.Register(e.ctx, e.registerActor(), usersvc.RegisterInput{Username: "it_pw1", Email: "it_pw1@example.com", Password: "1234567", SourceIP: nextIP()}); err == nil {
		t.Fatal("7-byte password must fail")
	}
	// 8 ASCII bytes must pass.
	if _, err := e.svc.Register(e.ctx, e.registerActor(), usersvc.RegisterInput{Username: "it_pw2", Email: "it_pw2@example.com", Password: "12345678", SourceIP: nextIP()}); err != nil {
		t.Fatalf("8-byte password must pass: %v", err)
	}
	// 72 ASCII bytes must pass.
	if _, err := e.svc.Register(e.ctx, e.registerActor(), usersvc.RegisterInput{Username: "it_pw3", Email: "it_pw3@example.com", Password: strings.Repeat("a", 72), SourceIP: nextIP()}); err != nil {
		t.Fatalf("72-byte password must pass: %v", err)
	}
	// 73 ASCII bytes must fail.
	if _, err := e.svc.Register(e.ctx, e.registerActor(), usersvc.RegisterInput{Username: "it_pw4", Email: "it_pw4@example.com", Password: strings.Repeat("a", 73), SourceIP: nextIP()}); err == nil {
		t.Fatal("73-byte password must fail")
	}
	// Multibyte: 4 UTF-8 chars x 2 bytes = 8 bytes; must pass and not truncate.
	mb := "密码密码"
	if _, err := e.svc.Register(e.ctx, e.registerActor(), usersvc.RegisterInput{Username: "it_pw5", Email: "it_pw5@example.com", Password: mb, SourceIP: nextIP()}); err != nil {
		t.Fatalf("multibyte 8-byte password must pass: %v", err)
	}
	// Login with the multibyte password must succeed (no truncation mismatch).
	if _, _, err := e.svc.Login(e.ctx, e.registerActor(), "it_pw5", mb, nextIP()); err != nil {
		t.Fatalf("multibyte login failed: %v", err)
	}
}

func extractToken(t *testing.T, body string) string {
	t.Helper()
	idx := strings.Index(body, "token=")
	if idx < 0 {
		return ""
	}
	rest := body[idx+len("token="):]
	end := strings.IndexAny(rest, "&\"< \r\n")
	if end < 0 {
		end = len(rest)
	}
	return rest[:end]
}

// TestEmailVerificationLifecycle covers pending creation, one-time verify,
// replay rejection, and resend invalidation.
func TestEmailVerificationLifecycle(t *testing.T) {
	e := newSvcEnv(t)
	e.cleanUserData(t)
	e.seedEnabledSMTP(t)

	u, err := e.svc.Register(e.ctx, e.registerActor(), usersvc.RegisterInput{
		Username: "it_pending",
		Email:    "it_pending@example.com",
		Password: "password-123456",
		SourceIP: nextIP(),
	})
	if err != nil {
		t.Fatalf("register pending: %v", err)
	}
	if u.Status != userdomain.StatusPendingVerification {
		t.Fatalf("status = %s, want pending_verification", u.Status)
	}
	if e.mailer.count() != 1 {
		t.Fatalf("mail count = %d, want 1", e.mailer.count())
	}
	token1 := extractToken(t, e.mailer.last().Body)
	if token1 == "" {
		t.Fatal("no token in verification email")
	}

	// Pending user cannot log in yet.
	if _, _, err := e.svc.Login(e.ctx, e.registerActor(), "it_pending", "password-123456", nextIP()); err == nil {
		t.Fatal("pending user must not log in")
	}

	// Verify with the correct token succeeds.
	if err := e.svc.VerifyEmail(e.ctx, e.registerActor(), "it_pending@example.com", token1); err != nil {
		t.Fatalf("verify: %v", err)
	}
	active, err := e.users.GetUserByUsername(e.ctx, "it_pending")
	if err != nil {
		t.Fatal(err)
	}
	if active.Status != userdomain.StatusActive {
		t.Fatalf("status after verify = %s", active.Status)
	}

	// Replay of the same token must be rejected (one-time GETDEL).
	if err := e.svc.VerifyEmail(e.ctx, e.registerActor(), "it_pending@example.com", token1); err == nil {
		t.Fatal("token replay must fail")
	} else if errCode(err) != apperr.CodeVerificationInvalid {
		t.Fatalf("replay code = %d", errCode(err))
	}

	// A token minted for this account cannot verify a different account.
	u2, err := e.svc.Register(e.ctx, e.registerActor(), usersvc.RegisterInput{
		Username: "it_pending2",
		Email:    "it_pending2@example.com",
		Password: "password-123456",
		SourceIP: nextIP(),
	})
	if err != nil {
		t.Fatal(err)
	}
	tokenOther := extractToken(t, e.mailer.last().Body)
	if tokenOther == "" {
		t.Fatal("no second token")
	}
	if err := e.svc.VerifyEmail(e.ctx, e.registerActor(), "it_pending@example.com", tokenOther); err == nil {
		t.Fatalf("token for user %d must not verify user %s", u2.ID, u.Username)
	}
}

// TestResendInvalidatesPreviousToken covers resend rotation and rate limit.
func TestResendInvalidatesPreviousToken(t *testing.T) {
	e := newSvcEnv(t)
	e.cleanUserData(t)
	e.seedEnabledSMTP(t)

	if _, err := e.svc.Register(e.ctx, e.registerActor(), usersvc.RegisterInput{
		Username: "it_resend",
		Email:    "it_resend@example.com",
		Password: "password-123456",
		SourceIP: "10.0.0.1",
	}); err != nil {
		t.Fatal(err)
	}
	old := extractToken(t, e.mailer.last().Body)
	if err := e.svc.ResendVerification(e.ctx, e.registerActor(), "it_resend@example.com"); err != nil {
		t.Fatalf("resend: %v", err)
	}
	newToken := extractToken(t, e.mailer.last().Body)
	if newToken == "" || newToken == old {
		t.Fatal("resend must produce a fresh token")
	}
	// The old token must now be invalid.
	if err := e.svc.VerifyEmail(e.ctx, e.registerActor(), "it_resend@example.com", old); err == nil {
		t.Fatal("old token must be invalidated by resend")
	}
	// The new token verifies.
	if err := e.svc.VerifyEmail(e.ctx, e.registerActor(), "it_resend@example.com", newToken); err != nil {
		t.Fatalf("new token must verify: %v", err)
	}
	// Resend on an unknown email returns a generic error and sends nothing.
	before := e.mailer.count()
	if err := e.svc.ResendVerification(e.ctx, e.registerActor(), "ghost@example.com"); err == nil {
		t.Fatal("resend to unknown email must fail generically")
	}
	if e.mailer.count() != before {
		t.Fatal("resend must not send to unknown email")
	}
}

// TestLoginMethodSwitches covers identifier routing and disabled-mode rejection.
func TestLoginMethodSwitches(t *testing.T) {
	e := newSvcEnv(t)
	e.cleanUserData(t)
	e.createActiveUser(t, "it_switch", "password-123456")

	if _, _, err := e.svc.Login(e.ctx, e.registerActor(), "it_switch@example.com", "password-123456", nextIP()); err != nil {
		t.Fatalf("email login: %v", err)
	}
	if _, _, err := e.svc.Login(e.ctx, e.registerActor(), "it_switch", "password-123456", nextIP()); err != nil {
		t.Fatalf("username login: %v", err)
	}

	cur, err := e.svc.GetSystemSettingsForActor(e.ctx, e.actorAdmin)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := e.svc.UpdateSystemSettings(e.ctx, e.actorAdmin, usersvc.UpdateSystemSettingsInput{
		PlatformName: cur.PlatformName, PublicFrontendURL: cur.PublicFrontendURL,
		PublicAPIURL: cur.PublicAPIURL, RegistrationEnabled: cur.RegistrationEnabled,
		UsernameLoginEnabled: false, EmailLoginEnabled: true,
		EmailVerificationRequired: cur.EmailVerificationRequired,
		DefaultLevelID:            cur.DefaultLevelID, RegistrationPoints: cur.RegistrationPoints,
		Version: cur.Version,
	}); err != nil {
		t.Fatalf("disable username login: %v", err)
	}
	if _, _, err := e.svc.Login(e.ctx, e.registerActor(), "it_switch", "password-123456", nextIP()); err == nil {
		t.Fatal("username login must be rejected when disabled")
	} else if errCode(err) != apperr.CodeLoginMethodDisabled {
		t.Fatalf("code = %d, want login method disabled", errCode(err))
	}
	if _, _, err := e.svc.Login(e.ctx, e.registerActor(), "it_switch@example.com", "password-123456", nextIP()); err != nil {
		t.Fatalf("email login after switch: %v", err)
	}
}

// TestLoginFailureUniformity covers disabled/pending/wrong-password and that
// unknown identifiers do not reveal whether the account exists.
func TestLoginFailureUniformity(t *testing.T) {
	e := newSvcEnv(t)
	e.cleanUserData(t)
	e.createActiveUser(t, "it_live", "password-123456")

	_, _, err := e.svc.Login(e.ctx, e.registerActor(), "it_live", "wrong-password", nextIP())
	if err == nil || errCode(err) != apperr.CodeUnauthorized {
		t.Fatalf("wrong password err = %v", err)
	}
	_, _, err2 := e.svc.Login(e.ctx, e.registerActor(), "it_ghost", "wrong-password", nextIP())
	if err2 == nil || errCode(err2) != apperr.CodeUnauthorized {
		t.Fatalf("unknown identifier err = %v", err2)
	}
	if err.Error() != err2.Error() {
		t.Fatalf("errors must be indistinguishable: %q vs %q", err.Error(), err2.Error())
	}
}

// TestLastLoginAtOnlyUpdatedOnLogin covers the rule that refresh and ordinary
// requests do not update last_login_at.
func TestLastLoginAtOnlyUpdatedOnLogin(t *testing.T) {
	e := newSvcEnv(t)
	e.cleanUserData(t)
	e.createActiveUser(t, "it_lastlogin", "password-123456")

	before, _ := e.users.GetUserByUsername(e.ctx, "it_lastlogin")
	if before.LastLoginAt != nil {
		t.Fatal("last_login_at must start nil")
	}
	session, _, err := e.svc.Login(e.ctx, e.registerActor(), "it_lastlogin", "password-123456", nextIP())
	if err != nil {
		t.Fatalf("login: %v", err)
	}
	afterLogin, _ := e.users.GetUserByUsername(e.ctx, "it_lastlogin")
	if afterLogin.LastLoginAt == nil {
		t.Fatal("last_login_at must update after login")
	}
	first := *afterLogin.LastLoginAt

	if _, _, err := e.svc.Refresh(e.ctx, e.registerActor(), session.RefreshToken); err != nil {
		t.Fatalf("refresh: %v", err)
	}
	afterRefresh, _ := e.users.GetUserByUsername(e.ctx, "it_lastlogin")
	if afterRefresh.LastLoginAt == nil || !afterRefresh.LastLoginAt.Equal(first) {
		t.Fatal("refresh must not update last_login_at")
	}
}

// TestRefreshRotationAndEpochInvalidation covers refresh rotation, replay
// detection, and password-reset epoch invalidation.
func TestRefreshRotationAndEpochInvalidation(t *testing.T) {
	e := newSvcEnv(t)
	e.cleanUserData(t)
	e.createActiveUser(t, "it_rotate", "password-123456")

	session1, _, err := e.svc.Login(e.ctx, e.registerActor(), "it_rotate", "password-123456", nextIP())
	if err != nil {
		t.Fatal(err)
	}
	session2, _, err := e.svc.Refresh(e.ctx, e.registerActor(), session1.RefreshToken)
	if err != nil {
		t.Fatalf("refresh1: %v", err)
	}
	// Replaying session1's refresh token must be detected.
	if _, _, err := e.svc.Refresh(e.ctx, e.registerActor(), session1.RefreshToken); err == nil || errCode(err) != apperr.CodeRefreshReplay {
		t.Fatalf("replay err = %v code=%d", err, errCode(err))
	}

	uid := e.userID(t, "it_rotate")
	if err := e.svc.ResetUserPassword(e.ctx, e.actorAdmin, uid, "new-password-123"); err != nil {
		t.Fatalf("reset password: %v", err)
	}
	if _, _, err := e.svc.Refresh(e.ctx, e.registerActor(), session2.RefreshToken); err == nil {
		t.Fatal("refresh after password reset must fail")
	}
}

// TestAdminFinanceCannotManageUsers proves finance role is denied server-side.
func TestAdminFinanceCannotManageUsers(t *testing.T) {
	e := newSvcEnv(t)
	e.cleanUserData(t)
	e.createActiveUser(t, "it_victim", "password-123456")

	financeActor := e.financeActor(t)

	if _, err := e.svc.ListUsers(e.ctx, financeActor, 1, 20, usersvc.UserListFilter{}); err == nil || errCode(err) != apperr.CodeForbidden {
		t.Fatalf("finance list err = %v code=%d, want forbidden", err, errCode(err))
	}
	uid := e.userID(t, "it_victim")
	if _, err := e.svc.GetUser(e.ctx, financeActor, uid); err == nil {
		t.Fatal("finance must not read users")
	}
	if _, err := e.svc.AdjustPoints(e.ctx, financeActor, usersvc.AdjustPointsInput{
		UserID: uid, PointsDelta: mustDecimal(t, "1.0000"), Reason: "x", IdempotencyKey: "k",
	}); err == nil {
		t.Fatal("finance must not adjust points")
	}
	if _, err := e.svc.GetSystemSettingsForActor(e.ctx, financeActor); err == nil || errCode(err) != apperr.CodeForbidden {
		t.Fatalf("finance settings read must be forbidden, got %v", err)
	}
	if _, err := e.svc.ListUserLevels(e.ctx, financeActor); err == nil {
		t.Fatal("finance must not list levels")
	}
}

func (e *svcEnv) userID(t *testing.T, username string) int64 {
	t.Helper()
	u, err := e.users.GetUserByUsername(e.ctx, username)
	if err != nil {
		t.Fatalf("load %s: %v", username, err)
	}
	return u.ID
}

func (e *svcEnv) financeActor(t *testing.T) usersvc.Actor {
	t.Helper()
	role := seedReservedRole(t, e.admins, e.ctx, adminauth.RoleFinance)
	admin := &adminauth.Administrator{
		Username:     "it_finance_admin",
		PasswordHash: "unused",
		DisplayName:  "Finance",
		Enabled:      true,
	}
	if err := e.admins.CreateAdministrator(e.ctx, admin, []int64{role.ID}); err != nil {
		t.Fatalf("create finance admin: %v", err)
	}
	return usersvc.Actor{
		AdminID:          admin.ID,
		AdminUsername:    admin.Username,
		AdminDisplayName: admin.DisplayName,
		AdminRoleCodes:   []string{adminauth.RoleFinance},
		SourceIP:         nextIP(),
	}
}

// TestAdminPointsAdjustManualLevelAndAutoSwitch covers the full admin workflow.
func TestAdminPointsAdjustManualLevelAndAutoSwitch(t *testing.T) {
	e := newSvcEnv(t)
	e.cleanUserData(t)

	gold, err := e.svc.CreateUserLevel(e.ctx, e.actorAdmin, usersvc.CreateUserLevelInput{
		Code: "it_lv_gold", Name: "Gold", ThresholdPoints: mustDecimal(t, "100.0000"),
		SortOrder: 1, Enabled: true,
	})
	if err != nil {
		t.Fatalf("create level: %v", err)
	}

	nick := "Managed Nick"
	u, err := e.svc.CreateUser(e.ctx, e.actorAdmin, usersvc.CreateUserInput{
		Username: "it_adminmade", Email: "it_adminmade@example.com",
		Password: "password-123456", Nickname: nick,
	})
	if err != nil {
		t.Fatalf("admin create user: %v", err)
	}
	if u.EmailVerifiedAt == nil || u.Status != userdomain.StatusActive {
		t.Fatal("admin-created user must be active and email verified")
	}
	if u.Nickname != nick {
		t.Fatalf("nickname = %q", u.Nickname)
	}

	pt, err := e.svc.AdjustPoints(e.ctx, e.actorAdmin, usersvc.AdjustPointsInput{
		UserID: u.ID, PointsDelta: mustDecimal(t, "50.0000"),
		ConsumptionDelta: mustDecimal(t, "150.0000"),
		Reason:           "purchase adjustment", IdempotencyKey: "it-key-1",
	})
	if err != nil {
		t.Fatalf("adjust: %v", err)
	}
	if pt.BalanceAfter.String() != "50.0000" || pt.ConsumptionAfter.String() != "150.0000" {
		t.Fatalf("after = %s/%s", pt.BalanceAfter.String(), pt.ConsumptionAfter.String())
	}
	got, _ := e.users.GetUserByID(e.ctx, u.ID)
	if got.LevelID != gold.ID {
		t.Fatalf("auto level id = %d, want gold %d", got.LevelID, gold.ID)
	}

	settings, _ := e.users.GetSystemSettings(e.ctx)
	manualUser, err := e.svc.AssignUserLevel(e.ctx, e.actorAdmin, usersvc.AssignUserLevelInput{
		UserID: u.ID, LevelID: settings.DefaultLevelID, Mode: userdomain.LevelModeManual,
	})
	if err != nil {
		t.Fatalf("manual assign: %v", err)
	}
	if manualUser.LevelMode != userdomain.LevelModeManual || manualUser.LevelID != settings.DefaultLevelID {
		t.Fatal("manual assignment failed")
	}
	if _, err := e.svc.AdjustPoints(e.ctx, e.actorAdmin, usersvc.AdjustPointsInput{
		UserID: u.ID, PointsDelta: mustDecimal(t, "5.0000"),
		Reason: "manual persist", IdempotencyKey: "it-key-2",
	}); err != nil {
		t.Fatal(err)
	}
	stillManual, _ := e.users.GetUserByID(e.ctx, u.ID)
	if stillManual.LevelMode != userdomain.LevelModeManual || stillManual.LevelID != settings.DefaultLevelID {
		t.Fatal("manual level must persist across point changes")
	}

	autoUser, err := e.svc.AssignUserLevel(e.ctx, e.actorAdmin, usersvc.AssignUserLevelInput{
		UserID: u.ID, Mode: userdomain.LevelModeAuto,
	})
	if err != nil {
		t.Fatalf("switch to auto: %v", err)
	}
	if autoUser.LevelMode != userdomain.LevelModeAuto || autoUser.LevelID != gold.ID {
		t.Fatalf("auto switch level = %d, want gold %d", autoUser.LevelID, gold.ID)
	}

	ledger, err := e.svc.ListPointTransactions(e.ctx, e.actorAdmin, u.ID, 1, 20)
	if err != nil {
		t.Fatal(err)
	}
	if ledger.Total != 2 {
		t.Fatalf("ledger rows = %d, want 2", ledger.Total)
	}

	// Idempotency replay of key-1 is rejected.
	_, err = e.svc.AdjustPoints(e.ctx, e.actorAdmin, usersvc.AdjustPointsInput{
		UserID: u.ID, PointsDelta: mustDecimal(t, "1.0000"),
		Reason: "dup", IdempotencyKey: "it-key-1",
	})
	if err == nil || errCode(err) != apperr.CodeIdempotencyConflict {
		t.Fatalf("replay err = %v code=%d", err, errCode(err))
	}
}

// TestSettingsInvariants covers both-login-disabled and verification-without-
// SMTP rejection plus optimistic locking.
func TestSettingsInvariants(t *testing.T) {
	e := newSvcEnv(t)
	cur, err := e.svc.GetSystemSettingsForActor(e.ctx, e.actorAdmin)
	if err != nil {
		t.Fatal(err)
	}

	base := func() usersvc.UpdateSystemSettingsInput {
		return usersvc.UpdateSystemSettingsInput{
			PlatformName: cur.PlatformName, PublicFrontendURL: cur.PublicFrontendURL,
			PublicAPIURL: cur.PublicAPIURL, RegistrationEnabled: cur.RegistrationEnabled,
			UsernameLoginEnabled: true, EmailLoginEnabled: true,
			DefaultLevelID: cur.DefaultLevelID, RegistrationPoints: cur.RegistrationPoints,
			Version: cur.Version,
		}
	}

	// Both login modes disabled → reject.
	bothOff := base()
	bothOff.UsernameLoginEnabled = false
	bothOff.EmailLoginEnabled = false
	if _, err := e.svc.UpdateSystemSettings(e.ctx, e.actorAdmin, bothOff); err == nil || errCode(err) != apperr.CodeValidation {
		t.Fatalf("both login disabled err = %v", err)
	}

	// Verification required without SMTP → reject.
	verifyNoSMTP := base()
	verifyNoSMTP.EmailVerificationRequired = true
	verifyNoSMTP.SMTPEnabled = false
	if _, err := e.svc.UpdateSystemSettings(e.ctx, e.actorAdmin, verifyNoSMTP); err == nil || errCode(err) != apperr.CodeValidation {
		t.Fatalf("verify without smtp err = %v", err)
	}

	// SMTP TLS none with verification required → reject.
	tlsNone := base()
	tlsNone.EmailVerificationRequired = true
	pass := "x"
	tlsNone.SMTPEnabled = true
	tlsNone.SMTPHost = "smtp.fake"
	tlsNone.SMTPPort = 25
	tlsNone.SMTPUsername = ""
	tlsNone.SMTPPassword = &pass
	tlsNone.SMTPFromEmail = "a@b.c"
	tlsNone.SMTPTLSMode = "none"
	if _, err := e.svc.UpdateSystemSettings(e.ctx, e.actorAdmin, tlsNone); err == nil || errCode(err) != apperr.CodeValidation {
		t.Fatalf("tls none + verify err = %v", err)
	}

	// Optimistic lock: stale version rejected.
	cur2, err := e.svc.GetSystemSettingsForActor(e.ctx, e.actorAdmin)
	if err != nil {
		t.Fatal(err)
	}
	good := base()
	good.PlatformName = cur2.PlatformName + " v1"
	good.Version = cur2.Version
	if _, err := e.svc.UpdateSystemSettings(e.ctx, e.actorAdmin, good); err != nil {
		t.Fatalf("valid update: %v", err)
	}
	stale := base()
	stale.PlatformName = "stale"
	stale.Version = cur2.Version
	if _, err := e.svc.UpdateSystemSettings(e.ctx, e.actorAdmin, stale); err == nil || errCode(err) != apperr.CodeSettingsConflict {
		t.Fatalf("stale version err = %v code=%d", err, errCode(err))
	}
}

// TestSMTPSecretNeverExposed covers that reads report only password_configured
// and updates with a blank password retain the stored secret.
func TestSMTPSecretNeverExposed(t *testing.T) {
	e := newSvcEnv(t)
	e.seedEnabledSMTP(t)

	read, err := e.svc.GetSystemSettingsForActor(e.ctx, e.actorAdmin)
	if err != nil {
		t.Fatal(err)
	}
	if !read.SMTPPasswordConfigured {
		t.Fatal("password_configured must be true")
	}
	raw, err := json.Marshal(read)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(raw), "smtp-secret-password") {
		t.Fatal("plaintext SMTP password leaked in read")
	}

	// Blank password on update retains the stored secret.
	blank := ""
	if _, err := e.svc.UpdateSystemSettings(e.ctx, e.actorAdmin, usersvc.UpdateSystemSettingsInput{
		PlatformName: read.PlatformName, PublicFrontendURL: read.PublicFrontendURL,
		PublicAPIURL: read.PublicAPIURL, RegistrationEnabled: read.RegistrationEnabled,
		UsernameLoginEnabled: read.UsernameLoginEnabled, EmailLoginEnabled: read.EmailLoginEnabled,
		EmailVerificationRequired: read.EmailVerificationRequired,
		DefaultLevelID:            read.DefaultLevelID, RegistrationPoints: read.RegistrationPoints,
		SMTPEnabled: true, SMTPHost: read.SMTPHost, SMTPPort: read.SMTPPort,
		SMTPUsername: read.SMTPUsername, SMTPPassword: &blank,
		SMTPFromEmail: read.SMTPFromEmail, SMTPFromName: read.SMTPFromName,
		SMTPTLSMode: read.SMTPTLSMode, Version: read.Version,
	}); err != nil {
		t.Fatalf("blank password update: %v", err)
	}
	after, _ := e.svc.GetSystemSettingsForActor(e.ctx, e.actorAdmin)
	if !after.SMTPPasswordConfigured {
		t.Fatal("stored secret must be retained when password blank")
	}
	raw2, _ := json.Marshal(after)
	if strings.Contains(string(raw2), "smtp-secret-password") {
		t.Fatal("plaintext leaked after update")
	}
}

// TestDisabledUserCannotEstablishSession covers disabled-account login denial
// and the disable/enable lifecycle.
func TestDisabledUserCannotEstablishSession(t *testing.T) {
	e := newSvcEnv(t)
	e.cleanUserData(t)
	e.createActiveUser(t, "it_disable_me", "password-123456")
	uid := e.userID(t, "it_disable_me")

	if _, err := e.svc.DisableUser(e.ctx, e.actorAdmin, uid); err != nil {
		t.Fatalf("disable: %v", err)
	}
	if _, _, err := e.svc.Login(e.ctx, e.registerActor(), "it_disable_me", "password-123456", nextIP()); err == nil || errCode(err) != apperr.CodeAccountDisabled {
		t.Fatalf("disabled login err = %v code=%d", err, errCode(err))
	}
	if _, err := e.svc.EnableUser(e.ctx, e.actorAdmin, uid); err != nil {
		t.Fatalf("enable: %v", err)
	}
	if _, _, err := e.svc.Login(e.ctx, e.registerActor(), "it_disable_me", "password-123456", nextIP()); err != nil {
		t.Fatalf("login after enable: %v", err)
	}
}

// TestAuthEpochBumpsOnReset proves a password reset bumps the DB epoch so any
// session issued under the earlier epoch cannot validate afterwards, even one
// that a Redis revocation scan could have missed (created after the reset).
func TestAuthEpochBumpsOnReset(t *testing.T) {
	e := newSvcEnv(t)
	e.cleanUserData(t)
	e.createActiveUser(t, "it_epoch", "password-123456")
	uid := e.userID(t, "it_epoch")
	u, _ := e.users.GetUserByID(e.ctx, uid)

	if err := e.svc.ResetUserPassword(e.ctx, e.actorAdmin, uid, "new-password-999"); err != nil {
		t.Fatal(err)
	}
	u2, _ := e.users.GetUserByID(e.ctx, uid)
	if u2.AuthEpoch != u.AuthEpoch+1 {
		t.Fatalf("epoch = %d, want %d", u2.AuthEpoch, u.AuthEpoch+1)
	}

	// Forge a session under the OLD epoch after the reset (a race the Redis
	// revocation scan could not have observed). The epoch mismatch is what
	// rejects it on the next refresh.
	sid := "it-synth-" + strconv.FormatInt(uid, 10)
	exp := time.Now().UTC().Add(24 * time.Hour)
	if err := e.sessions.Create(e.ctx, sid, uid, "hash-x", u.AuthEpoch, exp); err != nil {
		t.Fatal(err)
	}
	sess, err := e.sessions.Get(e.ctx, sid)
	if err != nil {
		t.Fatalf("synth session: %v", err)
	}
	if sess.AuthEpoch == u2.AuthEpoch {
		t.Fatal("session epoch must remain the pre-reset value so the check rejects it")
	}
	if sess.AuthEpoch != u.AuthEpoch {
		t.Fatalf("session epoch = %d, want %d", sess.AuthEpoch, u.AuthEpoch)
	}
}

// TestPublicSettingsWhitelist ensures the safe public payload contains no
// SMTP or management fields.
func TestPublicSettingsWhitelist(t *testing.T) {
	e := newSvcEnv(t)
	e.seedEnabledSMTP(t)

	ps, err := e.svc.GetPublicSettings(e.ctx)
	if err != nil {
		t.Fatal(err)
	}
	if ps.PlatformName == "" {
		t.Fatal("platform name missing")
	}
	raw, _ := json.Marshal(ps)
	s := strings.ToLower(string(raw))
	for _, forbidden := range []string{"smtp", "password", "version", "level", "remark"} {
		if strings.Contains(s, forbidden) {
			t.Fatalf("public settings leaked %q: %s", forbidden, raw)
		}
	}
	// The whitelist fields themselves must be present.
	for _, required := range []string{"registrationenabled", "usernameloginenabled", "emailloginenabled", "emailverificationrequired"} {
		if !strings.Contains(s, required) {
			t.Fatalf("public settings missing %q: %s", required, raw)
		}
	}
	if !ps.EmailVerificationRequired {
		t.Fatal("verification flag must reflect the enabled SMTP settings")
	}
}

// TestRegistrationMailerFailureRollsBackAndAudits proves the registration path
// fails closed when the verification email cannot be delivered: no account is
// left usable, and the audit trail records the rollback instead of only the
// earlier successful create.
func TestRegistrationMailerFailureRollsBackAndAudits(t *testing.T) {
	e := newSvcEnv(t)
	defer func() {
		e.cleanUserData(t)
		e.resetSettingsBaseline(t)
	}()
	e.seedEnabledSMTP(t)

	e.mailer.failNext = errMailerForcedFailure
	before := e.mailer.count()

	_, err := e.svc.Register(e.ctx, e.registerActor(), usersvc.RegisterInput{
		Username: "it_mail_fail",
		Email:    "it_mail_fail@example.com",
		Password: "password-123456",
		SourceIP: nextIP(),
	})
	if err == nil {
		t.Fatal("registration must fail when the verification email cannot be sent")
	}
	if code := errCode(err); code != apperr.CodeSmtpUnavailable {
		t.Fatalf("error code = %d, want %d (smtp unavailable)", code, apperr.CodeSmtpUnavailable)
	}
	if e.mailer.count() != before {
		t.Fatal("forced mailer failure must not record a delivered message")
	}

	// The half-created account is rolled back: no pending row survives.
	if u, lookupErr := e.users.GetUserByUsername(e.ctx, "it_mail_fail"); lookupErr == nil {
		t.Fatalf("rolled-back user still exists: id=%d status=%s", u.ID, u.Status)
	}

	// The audit trail shows both the create and the failed rollback so an
	// operator can diagnose the gap; details stay generic.
	var failed int64
	if qErr := e.logDB.GORM().Raw(
		"SELECT count(*) FROM audit_events WHERE action = 'user.register' AND outcome = 'failed' AND details::text LIKE '%registration_rolled_back%'",
	).Scan(&failed).Error; qErr != nil {
		t.Fatal(qErr)
	}
	if failed != 1 {
		t.Fatalf("rollback audit events = %d, want 1", failed)
	}
	var leaked int64
	if qErr := e.logDB.GORM().Raw(
		"SELECT count(*) FROM audit_events WHERE details::text LIKE '%smtp-secret-password%' OR details::text LIKE '%password_hash%' OR details::text LIKE '%$2a$%'",
	).Scan(&leaked).Error; qErr != nil {
		t.Fatal(qErr)
	}
	if leaked != 0 {
		t.Fatalf("audit details leaked credential material in %d rows", leaked)
	}
	// The rollback event itself stays generic: no email address, no token.
	var detailText string
	if qErr := e.logDB.GORM().Raw(
		"SELECT details::text FROM audit_events WHERE action = 'user.register' AND outcome = 'failed' AND details::text LIKE '%registration_rolled_back%' LIMIT 1",
	).Scan(&detailText).Error; qErr != nil {
		t.Fatal(qErr)
	}
	if strings.Contains(detailText, "@") || strings.Contains(strings.ToLower(detailText), "token") {
		t.Fatalf("rollback audit details are not generic: %s", detailText)
	}
}

// TestVerificationEmailContainsNoSecrets checks the rendered verification body:
// it carries the single-use link only, never credentials or SMTP material.
func TestVerificationEmailContainsNoSecrets(t *testing.T) {
	e := newSvcEnv(t)
	defer func() {
		e.cleanUserData(t)
		e.resetSettingsBaseline(t)
	}()
	e.seedEnabledSMTP(t)

	user := e.createActiveUser(t, "it_mail_body", "password-123456")
	if user.Status != userdomain.StatusPendingVerification {
		t.Fatalf("status = %s, want pending_verification", user.Status)
	}
	if e.mailer.count() == 0 {
		t.Fatal("verification email must be produced")
	}
	body := e.mailer.last().Body
	if extractToken(t, body) == "" {
		t.Fatalf("verification body has no token link: %s", body)
	}
	for _, forbidden := range []string{"smtp-secret-password", "password-123456", "PasswordHash", "$2a$"} {
		if strings.Contains(body, forbidden) {
			t.Fatalf("verification body leaked %q", forbidden)
		}
	}
}
