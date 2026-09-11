package adminauth

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
	domainadmin "github.com/rio9466/easy-admin/server/internal/domain/adminauth"
	"github.com/rio9466/easy-admin/server/internal/domain/apperr"
	platformauth "github.com/rio9466/easy-admin/server/internal/platform/auth"
	platformpostgres "github.com/rio9466/easy-admin/server/internal/platform/postgres"
	"github.com/rio9466/easy-admin/server/internal/repository/logdb"
	"github.com/rio9466/easy-admin/server/internal/repository/primary"
)

// recordingPasswordHasher records Hash/Compare calls so the dummy-comparison
// path can be asserted deterministically without wall-clock timing.
type recordingPasswordHasher struct {
	hashCalls     int
	hashOut       string
	compareHashes []string
}

func (r *recordingPasswordHasher) Hash(password string) (string, error) {
	r.hashCalls++
	r.hashOut = "rec-hash:" + password
	return r.hashOut, nil
}

func (r *recordingPasswordHasher) Compare(hash, password string) error {
	r.compareHashes = append(r.compareHashes, hash)
	return platformauth.ErrPasswordMismatch
}

type failingAuditStoreInternal struct{}

func (failingAuditStoreInternal) CreatePending(context.Context, *domainadmin.AuditEvent) (int64, error) {
	return 0, errors.New("log postgres unavailable")
}
func (failingAuditStoreInternal) Finalize(context.Context, int64, string, string, map[string]any) error {
	return errors.New("log postgres unavailable")
}
func (failingAuditStoreInternal) List(context.Context, logdb.AuditListFilter) (domainadmin.Page[domainadmin.AuditEvent], error) {
	return domainadmin.Page[domainadmin.AuditEvent]{}, errors.New("log postgres unavailable")
}
func (failingAuditStoreInternal) GetByID(context.Context, int64) (*domainadmin.AuditEvent, error) {
	return nil, errors.New("log postgres unavailable")
}

// TestUnknownUserLoginExecutesDummyBcryptComparison proves a nonexistent
// username performs one configured-cost bcrypt comparison against the
// startup-created dummy hash and returns the exact same result as a wrong
// password for an existing user, with no per-request hash generation.
func TestUnknownUserLoginExecutesDummyBcryptComparison(t *testing.T) {
	if os.Getenv(config.EnvPrimaryPostgresDSN) == "" {
		t.Skipf("integration setup unavailable; skipping without touching configured databases")
	}
	primaryDB := openInternalPrimary(t)
	defer func() { _ = primaryDB.Close() }()
	ctx := context.Background()

	role, err := primary.NewAdminRepository(primaryDB.GORM()).GetRoleByCode(ctx, domainadmin.RoleSuperAdmin)
	if err != nil {
		t.Fatalf("load super role: %v", err)
	}
	// Create an existing enabled administrator whose stored hash never matches.
	existing := &domainadmin.Administrator{
		Username:     "it_timing_existing",
		PasswordHash: "stored-hash-dummy",
		DisplayName:  "Existing",
		Enabled:      true,
	}
	repo := primary.NewAdminRepository(primaryDB.GORM())
	if err := repo.CreateAdministrator(ctx, existing, []int64{role.ID}); err != nil {
		t.Fatalf("create existing admin: %v", err)
	}

	mr, err := miniredis.Run()
	if err != nil {
		t.Fatalf("miniredis: %v", err)
	}
	defer mr.Close()
	rdb := goredis.NewClient(&goredis.Options{Addr: mr.Addr()})
	defer func() { _ = rdb.Close() }()
	sessions, err := platformauth.NewSessionStore(rdb)
	if err != nil {
		t.Fatalf("session store: %v", err)
	}
	tokens, err := platformauth.NewTokenService(platformauth.JWTConfig{
		Secret:   "internal-test-secret-at-least-32-bytes!!",
		Issuer:   "easy-admin",
		Audience: "easy-admin-admin",
	})
	if err != nil {
		t.Fatalf("tokens: %v", err)
	}
	hasher := &recordingPasswordHasher{}

	svc, err := New(repo, failingAuditStoreInternal{}, tokens, sessions, hasher, slog.Default(), AuthOptions{
		Environment: "development",
	})
	if err != nil {
		t.Fatalf("new service: %v", err)
	}
	if hasher.hashCalls != 1 {
		t.Fatalf("dummy hash generated %d times at construction, want exactly 1", hasher.hashCalls)
	}
	if svc.dummyHash == "" || svc.dummyHash != hasher.hashOut {
		t.Fatalf("service dummy hash %q does not match startup hash %q", svc.dummyHash, hasher.hashOut)
	}

	unknownErr := loginCode(t, svc, ctx, "it_timing_ghost", "some-pass-1234")
	wrongPasswordErr := loginCode(t, svc, ctx, "it_timing_existing", "wrong-pass-1234")

	if unknownErr == nil || wrongPasswordErr == nil {
		t.Fatal("both unknown-user and wrong-password logins must fail")
	}
	var unknownApp *apperr.AppError
	var wrongApp *apperr.AppError
	if !errors.As(unknownErr, &unknownApp) || !errors.As(wrongPasswordErr, &wrongApp) {
		t.Fatalf("errors are not AppErrors: %v / %v", unknownErr, wrongPasswordErr)
	}
	if unknownApp.Code != wrongApp.Code {
		t.Fatalf("unknown-user code %d != wrong-password code %d (response must not reveal existence)", unknownApp.Code, wrongApp.Code)
	}

	if len(hasher.compareHashes) != 2 {
		t.Fatalf("compare calls = %d, want 2 (one dummy, one real)", len(hasher.compareHashes))
	}
	// The unknown-user comparison ran against the startup dummy hash with the
	// supplied password, and the wrong-password path compared against the
	// stored hash (not the dummy).
	if hasher.compareHashes[0] != svc.dummyHash {
		t.Fatalf("unknown-user compare hash = %q, want dummy %q", hasher.compareHashes[0], svc.dummyHash)
	}
	if hasher.compareHashes[1] == svc.dummyHash {
		t.Fatal("wrong-password compare reused the dummy hash")
	}
	if hasher.compareHashes[1] != existing.PasswordHash {
		t.Fatalf("wrong-password compare hash = %q, want stored %q", hasher.compareHashes[1], existing.PasswordHash)
	}

	// More unknown-user logins must not generate new hashes per request.
	for i := 0; i < 2; i++ {
		_ = loginCode(t, svc, ctx, "it_timing_ghost_2", "another-pass-123")
	}
	if len(hasher.compareHashes) != 4 {
		t.Fatalf("compare calls after extra logins = %d, want 4", len(hasher.compareHashes))
	}
	if hasher.hashCalls != 1 {
		t.Fatalf("hash generated %d times, want exactly 1 (startup only)", hasher.hashCalls)
	}
}

func loginCode(t *testing.T, svc *Service, ctx context.Context, username, password string) error {
	t.Helper()
	_, err := svc.Login(ctx, Actor{RequestID: "req-timing"}, username, password)
	if err == nil {
		t.Fatalf("login %q unexpectedly succeeded", username)
	}
	return err
}

// openInternalPrimary opens the isolated primary database provisioned by
// TestMain. It is only reachable when disposable DB isolation succeeded.
func openInternalPrimary(t *testing.T) *platformpostgres.DB {
	t.Helper()
	cfgPath := os.Getenv("EASY_ADMIN_CONFIG")
	if cfgPath == "" {
		cfgPath = findInternalConfig()
	}
	cfg, err := config.Load(cfgPath)
	if err != nil {
		t.Skipf("load config: %v", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	db, err := platformpostgres.Open(ctx, cfg.Database.Primary)
	if err != nil {
		t.Skipf("primary postgres unavailable: %v", err)
	}
	return db
}

func findInternalConfig() string {
	wd, err := os.Getwd()
	if err != nil {
		return "configs/config.local.toml"
	}
	dir := wd
	for i := 0; i < 8; i++ {
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
