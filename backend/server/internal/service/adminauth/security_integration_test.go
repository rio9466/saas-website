package adminauth_test

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	goredis "github.com/redis/go-redis/v9"
	"github.com/rio9466/easy-admin/server/internal/config"
	"github.com/rio9466/easy-admin/server/internal/domain/adminauth"
	"github.com/rio9466/easy-admin/server/internal/domain/apperr"
	platformauth "github.com/rio9466/easy-admin/server/internal/platform/auth"
	platformpostgres "github.com/rio9466/easy-admin/server/internal/platform/postgres"
	"github.com/rio9466/easy-admin/server/internal/repository/logdb"
	"github.com/rio9466/easy-admin/server/internal/repository/primary"
	svc "github.com/rio9466/easy-admin/server/internal/service/adminauth"
	"golang.org/x/crypto/bcrypt"
)

type failingAuditStore struct{}

func (failingAuditStore) CreatePending(context.Context, *adminauth.AuditEvent) (int64, error) {
	return 0, errors.New("log postgres unavailable")
}
func (failingAuditStore) Finalize(context.Context, int64, string, string, map[string]any) error {
	return errors.New("log postgres unavailable")
}
func (failingAuditStore) List(context.Context, logdb.AuditListFilter) (adminauth.Page[adminauth.AuditEvent], error) {
	return adminauth.Page[adminauth.AuditEvent]{}, errors.New("log postgres unavailable")
}
func (failingAuditStore) GetByID(context.Context, int64) (*adminauth.AuditEvent, error) {
	return nil, errors.New("log postgres unavailable")
}

func TestLoginFailsClosedWhenAuditUnavailable(t *testing.T) {
	primaryHandle, adminRepo, sessions, tokens, passwords, cleanup := openAuthDeps(t)
	defer cleanup()
	ctx := context.Background()
	truncateAdmins(t, primaryHandle)

	hash, err := bcrypt.GenerateFromPassword([]byte("audit-fail-pass1"), 12)
	if err != nil {
		t.Fatalf("hash: %v", err)
	}
	role, err := adminRepo.GetRoleByCode(ctx, adminauth.RoleSuperAdmin)
	if err != nil {
		t.Fatalf("role: %v", err)
	}
	admin := &adminauth.Administrator{
		Username:     "audit_fail_user",
		PasswordHash: string(hash),
		DisplayName:  "Audit Fail",
		Enabled:      true,
	}
	if err := adminRepo.CreateAdministrator(ctx, admin, []int64{role.ID}); err != nil {
		t.Fatalf("create admin: %v", err)
	}

	service, err := svc.New(adminRepo, failingAuditStore{}, tokens, sessions, passwords, slog.Default(), svc.AuthOptions{
		Environment: "development",
	})
	if err != nil {
		t.Fatalf("new auth service: %v", err)
	}

	_, err = service.Login(ctx, svc.Actor{RequestID: "req-audit-down"}, "audit_fail_user", "audit-fail-pass1")
	if err == nil {
		t.Fatal("login succeeded despite audit outage")
	}
	var appErr *apperr.AppError
	if !errors.As(err, &appErr) || appErr.Code != apperr.CodeAuditUnavailable {
		t.Fatalf("err = %v, want audit unavailable", err)
	}
}

func TestResetPasswordRevokesExistingSessions(t *testing.T) {
	primaryHandle, adminRepo, sessions, tokens, passwords, cleanup := openAuthDeps(t)
	defer cleanup()
	ctx := context.Background()
	truncateAdmins(t, primaryHandle)

	cfg := mustLoadConfig(t)
	logDB, err := platformpostgres.Open(ctx, cfg.Database.Log)
	if err != nil {
		t.Skipf("log postgres unavailable: %v", err)
	}
	defer func() { _ = logDB.Close() }()
	audits := logdb.NewAuditRepository(logDB.GORM())
	service, err := svc.New(adminRepo, audits, tokens, sessions, passwords, slog.Default(), svc.AuthOptions{
		Environment: "development",
	})
	if err != nil {
		t.Fatalf("new auth service: %v", err)
	}

	superHash, err := bcrypt.GenerateFromPassword([]byte("super-pass-1234"), 12)
	if err != nil {
		t.Fatalf("super hash: %v", err)
	}
	targetHash, err := bcrypt.GenerateFromPassword([]byte("target-pass-1234"), 12)
	if err != nil {
		t.Fatalf("target hash: %v", err)
	}
	superRole, err := adminRepo.GetRoleByCode(ctx, adminauth.RoleSuperAdmin)
	if err != nil {
		t.Fatalf("super role: %v", err)
	}
	adminRole := seedReservedRole(t, adminRepo, ctx, adminauth.RoleAdmin)

	super := &adminauth.Administrator{Username: "reset_actor", PasswordHash: string(superHash), DisplayName: "Actor", Enabled: true}
	target := &adminauth.Administrator{Username: "reset_target", PasswordHash: string(targetHash), DisplayName: "Target", Enabled: true}
	if err := adminRepo.CreateAdministrator(ctx, super, []int64{superRole.ID}); err != nil {
		t.Fatalf("create super: %v", err)
	}
	if err := adminRepo.CreateAdministrator(ctx, target, []int64{adminRole.ID}); err != nil {
		t.Fatalf("create target: %v", err)
	}

	login, err := service.Login(ctx, svc.Actor{RequestID: "req-login-target"}, "reset_target", "target-pass-1234")
	if err != nil {
		t.Fatalf("target login: %v", err)
	}
	oldRefresh := login.RefreshToken
	oldSID := login.SessionID

	actor := svc.Actor{
		ID:        super.ID,
		Username:  super.Username,
		RoleCodes: []string{adminauth.RoleSuperAdmin},
		SessionID: "actor-session",
		RequestID: "req-reset",
	}
	if err := service.ResetAdministratorPassword(ctx, actor, target.ID, "target-pass-9999"); err != nil {
		t.Fatalf("reset password: %v", err)
	}

	if _, err := sessions.Get(ctx, oldSID); !errors.Is(err, platformauth.ErrSessionInvalid) {
		t.Fatalf("old session still valid: %v", err)
	}
	if _, err := service.Refresh(ctx, svc.Actor{RequestID: "req-refresh-old"}, oldRefresh); err == nil {
		t.Fatal("old refresh still works after password reset")
	}
}

func openAuthDeps(t *testing.T) (*platformpostgres.DB, *primary.AdminRepository, *platformauth.SessionStore, *platformauth.TokenService, *platformauth.PasswordHasher, func()) {
	t.Helper()
	cfg := mustLoadConfig(t)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	primaryDB, err := platformpostgres.Open(ctx, cfg.Database.Primary)
	if err != nil {
		t.Skipf("primary postgres unavailable: %v", err)
	}

	mr, err := miniredis.Run()
	if err != nil {
		t.Fatalf("miniredis: %v", err)
	}
	rdb := goredis.NewClient(&goredis.Options{Addr: mr.Addr()})
	sessions, err := platformauth.NewSessionStore(rdb)
	if err != nil {
		t.Fatalf("session store: %v", err)
	}
	tokens, err := platformauth.NewTokenService(platformauth.JWTConfig{
		Secret:   cfg.Auth.JWTSecret,
		Issuer:   cfg.Auth.JWTIssuer,
		Audience: cfg.Auth.JWTAudience,
	})
	if err != nil {
		t.Fatalf("tokens: %v", err)
	}
	passwords, err := platformauth.NewPasswordHasher(cfg.Auth.BcryptCost)
	if err != nil {
		t.Fatalf("passwords: %v", err)
	}

	cleanup := func() {
		_ = rdb.Close()
		mr.Close()
		_ = primaryDB.Close()
	}
	return primaryDB, primary.NewAdminRepository(primaryDB.GORM()), sessions, tokens, passwords, cleanup
}

func truncateAdmins(t *testing.T, primaryDB *platformpostgres.DB) {
	t.Helper()
	if err := primaryDB.GORM().Exec(`TRUNCATE administrator_roles, administrators RESTART IDENTITY CASCADE`).Error; err != nil {
		t.Fatalf("truncate: %v", err)
	}
}

func mustLoadConfig(t *testing.T) config.Config {
	t.Helper()
	if msg := isolationReason(); msg != "" {
		t.Skipf("%s", msg)
	}
	cfgPath := os.Getenv("EASY_ADMIN_CONFIG")
	if cfgPath == "" {
		cfgPath = findConfig(t)
	}
	cfg, err := config.Load(cfgPath)
	if err != nil {
		t.Skipf("load config: %v", err)
	}
	return cfg
}

func findConfig(t *testing.T) string {
	t.Helper()
	wd, _ := os.Getwd()
	dir := wd
	for i := 0; i < 6; i++ {
		candidate := filepath.Join(dir, "configs", "config.local.toml")
		if _, err := os.Stat(candidate); err == nil {
			return candidate
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return "configs/config.local.toml"
}
