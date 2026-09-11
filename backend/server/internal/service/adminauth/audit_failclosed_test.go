package adminauth_test

import (
	"context"
	"errors"
	"log/slog"
	"testing"
	"time"

	"github.com/rio9466/easy-admin/server/internal/domain/adminauth"
	"github.com/rio9466/easy-admin/server/internal/domain/apperr"
	platformauth "github.com/rio9466/easy-admin/server/internal/platform/auth"
	"github.com/rio9466/easy-admin/server/internal/repository/primary"
	svc "github.com/rio9466/easy-admin/server/internal/service/adminauth"
	"golang.org/x/crypto/bcrypt"
)

func newFailingAuditService(
	t *testing.T,
	adminRepo *primary.AdminRepository,
	sessions *platformauth.SessionStore,
	tokens *platformauth.TokenService,
	passwords *platformauth.PasswordHasher,
) *svc.Service {
	t.Helper()
	service, err := svc.New(adminRepo, failingAuditStore{}, tokens, sessions, passwords, slog.Default(), svc.AuthOptions{
		Environment: "development",
	})
	if err != nil {
		t.Fatalf("new failing audit service: %v", err)
	}
	return service
}

// TestLoginAuditUnavailableIsUniformAndSafe proves a login whose security
// audit cannot be persisted never succeeds and never reveals whether the
// username exists: unknown user, wrong password, and disabled account all
// surface the same audit-unavailable error.
func TestLoginAuditUnavailableIsUniformAndSafe(t *testing.T) {
	primaryDB, adminRepo, sessions, tokens, passwords, cleanup := openAuthDeps(t)
	defer cleanup()
	ctx := context.Background()
	truncateAdmins(t, primaryDB)

	role, err := adminRepo.GetRoleByCode(ctx, adminauth.RoleSuperAdmin)
	if err != nil {
		t.Fatalf("role: %v", err)
	}
	hash, err := bcrypt.GenerateFromPassword([]byte("correct-pass-1234"), 12)
	if err != nil {
		t.Fatalf("hash: %v", err)
	}
	disabled := &adminauth.Administrator{Username: "it_audit_disabled", PasswordHash: string(hash), DisplayName: "Disabled", Enabled: false}
	enabled := &adminauth.Administrator{Username: "it_audit_enabled", PasswordHash: string(hash), DisplayName: "Enabled", Enabled: true}
	if err := adminRepo.CreateAdministrator(ctx, disabled, []int64{role.ID}); err != nil {
		t.Fatalf("create disabled: %v", err)
	}
	if err := adminRepo.CreateAdministrator(ctx, enabled, []int64{role.ID}); err != nil {
		t.Fatalf("create enabled: %v", err)
	}

	service := newFailingAuditService(t, adminRepo, sessions, tokens, passwords)

	cases := []struct {
		name     string
		username string
		password string
	}{
		{name: "unknown user", username: "it_audit_ghost", password: "whatever-1234"},
		{name: "wrong password", username: "it_audit_enabled", password: "wrong-pass-1234"},
		{name: "disabled account", username: "it_audit_disabled", password: "correct-pass-1234"},
	}

	for _, tt := range cases {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			_, err := service.Login(ctx, svc.Actor{RequestID: "req-uniform"}, tt.username, tt.password)
			if err == nil {
				t.Fatal("login succeeded although audit is unavailable")
			}
			var appErr *apperr.AppError
			if !errors.As(err, &appErr) || appErr.Code != apperr.CodeAuditUnavailable {
				t.Fatalf("err = %v, want code %d", err, apperr.CodeAuditUnavailable)
			}
		})
	}
}

// TestManagementWritesFailClosedWhenAuditUnavailable proves that a disable,
// password reset, or password change cannot modify primary data when the
// initial audit insert fails.
func TestManagementWritesFailClosedWhenAuditUnavailable(t *testing.T) {
	primaryDB, adminRepo, sessions, tokens, passwords, cleanup := openAuthDeps(t)
	defer cleanup()
	ctx := context.Background()
	truncateAdmins(t, primaryDB)

	superRole, err := adminRepo.GetRoleByCode(ctx, adminauth.RoleSuperAdmin)
	if err != nil {
		t.Fatalf("super role: %v", err)
	}
	adminRole := seedReservedRole(t, adminRepo, ctx, adminauth.RoleAdmin)
	hashActor, _ := bcrypt.GenerateFromPassword([]byte("actor-pass-1234"), 12)
	hashTarget, _ := bcrypt.GenerateFromPassword([]byte("target-pass-1234"), 12)

	actor := &adminauth.Administrator{Username: "it_audit_actor", PasswordHash: string(hashActor), DisplayName: "Actor", Enabled: true}
	target := &adminauth.Administrator{Username: "it_audit_target", PasswordHash: string(hashTarget), DisplayName: "Target", Enabled: true}
	if err := adminRepo.CreateAdministrator(ctx, actor, []int64{superRole.ID}); err != nil {
		t.Fatalf("create actor: %v", err)
	}
	if err := adminRepo.CreateAdministrator(ctx, target, []int64{adminRole.ID}); err != nil {
		t.Fatalf("create target: %v", err)
	}

	svcActor := svc.Actor{
		ID:        actor.ID,
		Username:  actor.Username,
		RoleCodes: []string{adminauth.RoleSuperAdmin},
		SessionID: "it-actor-session",
		RequestID: "req-failclosed",
	}

	t.Run("disable does not modify target", func(t *testing.T) {
		service := newFailingAuditService(t, adminRepo, sessions, tokens, passwords)
		_, err := service.DisableAdministrator(ctx, svcActor, target.ID)
		assertAuditUnavailable(t, err)
		after, loadErr := adminRepo.GetAdministratorByID(ctx, target.ID)
		if loadErr != nil {
			t.Fatalf("reload target: %v", loadErr)
		}
		if !after.Enabled {
			t.Fatal("target was disabled although the audit insert failed")
		}
		if after.AuthEpoch != 0 {
			t.Fatalf("target auth_epoch = %d, want 0 (no write should occur)", after.AuthEpoch)
		}
	})

	t.Run("password reset does not modify target", func(t *testing.T) {
		service := newFailingAuditService(t, adminRepo, sessions, tokens, passwords)
		err := service.ResetAdministratorPassword(ctx, svcActor, target.ID, "brand-new-pass-123")
		assertAuditUnavailable(t, err)
		after, loadErr := adminRepo.GetAdministratorByUsername(ctx, target.Username)
		if loadErr != nil {
			t.Fatalf("reload target: %v", loadErr)
		}
		if after.PasswordHash != string(hashTarget) {
			t.Fatal("target password hash changed although the audit insert failed")
		}
		if after.AuthEpoch != 0 {
			t.Fatalf("target auth_epoch = %d, want 0", after.AuthEpoch)
		}
	})

	t.Run("own password change keeps current session valid", func(t *testing.T) {
		service := newFailingAuditService(t, adminRepo, sessions, tokens, passwords)
		sid := "it-own-session"
		if err := sessions.Create(ctx, sid, actor.ID, platformauth.HashRefreshVerifier("jti"), 0, nowPlusHour()); err != nil {
			t.Fatalf("create session: %v", err)
		}
		me := svc.Actor{ID: actor.ID, Username: actor.Username, SessionID: sid, RequestID: "req-change"}
		err := service.ChangePassword(ctx, me, "actor-pass-1234", "actor-new-pass-99")
		assertAuditUnavailable(t, err)
		if _, err := sessions.Get(ctx, sid); err != nil {
			t.Fatalf("session must stay valid when the audited change did not run: %v", err)
		}
	})
}

// TestLogoutFailsClosedWhenAuditUnavailable proves logout never reports
// success when its required audit insert fails.
func TestLogoutFailsClosedWhenAuditUnavailable(t *testing.T) {
	primaryDB, adminRepo, sessions, tokens, passwords, cleanup := openAuthDeps(t)
	defer cleanup()
	ctx := context.Background()
	truncateAdmins(t, primaryDB)

	role, err := adminRepo.GetRoleByCode(ctx, adminauth.RoleSuperAdmin)
	if err != nil {
		t.Fatalf("role: %v", err)
	}
	hash, _ := bcrypt.GenerateFromPassword([]byte("actor-pass-1234"), 12)
	admin := &adminauth.Administrator{Username: "it_audit_logout", PasswordHash: string(hash), DisplayName: "Actor", Enabled: true}
	if err := adminRepo.CreateAdministrator(ctx, admin, []int64{role.ID}); err != nil {
		t.Fatalf("create admin: %v", err)
	}

	sid := "it-logout-session"
	if err := sessions.Create(ctx, sid, admin.ID, platformauth.HashRefreshVerifier("jti"), 0, nowPlusHour()); err != nil {
		t.Fatalf("create session: %v", err)
	}

	service := newFailingAuditService(t, adminRepo, sessions, tokens, passwords)
	err = service.Logout(ctx, svc.Actor{ID: admin.ID, Username: admin.Username, SessionID: sid, RequestID: "req-logout"})
	assertAuditUnavailable(t, err)
	if _, err := sessions.Get(ctx, sid); err != nil {
		t.Fatalf("logout must not report success when its audit insert failed: %v", err)
	}
}

// TestRefreshReplayStillRevokesWhenAuditUnavailable proves a detected refresh
// replay revokes the session even when the replay audit event cannot be
// persisted, and the request is not reported as successful.
func TestRefreshReplayStillRevokesWhenAuditUnavailable(t *testing.T) {
	primaryDB, adminRepo, sessions, tokens, passwords, cleanup := openAuthDeps(t)
	defer cleanup()
	ctx := context.Background()
	truncateAdmins(t, primaryDB)

	role, err := adminRepo.GetRoleByCode(ctx, adminauth.RoleSuperAdmin)
	if err != nil {
		t.Fatalf("role: %v", err)
	}
	hash, _ := bcrypt.GenerateFromPassword([]byte("actor-pass-1234"), 12)
	admin := &adminauth.Administrator{Username: "it_audit_replay", PasswordHash: string(hash), DisplayName: "Actor", Enabled: true}
	if err := adminRepo.CreateAdministrator(ctx, admin, []int64{role.ID}); err != nil {
		t.Fatalf("create admin: %v", err)
	}

	subject := idString(admin.ID)
	sid := "it-replay-session"
	refresh, claims, err := tokens.IssueRefresh(subject, sid)
	if err != nil {
		t.Fatalf("issue refresh: %v", err)
	}
	oldHash := platformauth.HashRefreshVerifier(claims.ID)
	if err := sessions.Create(ctx, sid, admin.ID, oldHash, 0, nowPlusHour()); err != nil {
		t.Fatalf("create session: %v", err)
	}
	// First rotation succeeds and consumes the original verifier.
	_, newClaims, err := tokens.IssueRefresh(subject, sid)
	if err != nil {
		t.Fatalf("issue new refresh: %v", err)
	}
	if err := sessions.RotateRefresh(ctx, sid, oldHash, platformauth.HashRefreshVerifier(newClaims.ID)); err != nil {
		t.Fatalf("rotate: %v", err)
	}

	service := newFailingAuditService(t, adminRepo, sessions, tokens, passwords)
	// Replaying the original refresh token is a replay: the session is revoked,
	// the audit insert fails, and the request must not look successful.
	_, err = service.Refresh(ctx, svc.Actor{RequestID: "req-replay"}, refresh)
	assertAuditUnavailable(t, err)
	if _, err := sessions.Get(ctx, sid); !errors.Is(err, platformauth.ErrSessionInvalid) {
		t.Fatalf("session must be revoked after replay regardless of audit failure: %v", err)
	}
}

func assertAuditUnavailable(t *testing.T, err error) {
	t.Helper()
	if err == nil {
		t.Fatal("operation succeeded although the audit insert failed")
	}
	var appErr *apperr.AppError
	if !errors.As(err, &appErr) || appErr.Code != apperr.CodeAuditUnavailable {
		t.Fatalf("err = %v, want audit-unavailable code", err)
	}
}

func nowPlusHour() time.Time { return time.Now().UTC().Add(time.Hour) }
