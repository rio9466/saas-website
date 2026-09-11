package adminauth_test

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/rio9466/easy-admin/server/internal/config"
)

// This package's integration tests run against unique disposable primary and
// log databases created and migrated in TestMain. Names embed the process PID
// and random bytes so concurrent `go test` invocations never drop or terminate
// each other's databases, and cleanup drops only the databases this invocation
// created. When configuration, the CREATEDB privilege, or either PostgreSQL
// service is unavailable the tests are skipped and the configured application
// databases are never opened for mutation.
var isolatedStoresAvailable bool

func TestMain(m *testing.M) {
	os.Exit(runIntegrationMain(m))
}

func runIntegrationMain(m *testing.M) int {
	cfgPath := os.Getenv("EASY_ADMIN_CONFIG")
	if cfgPath == "" {
		cfgPath = findRepoFile("configs/config.local.toml")
	}
	cfg, err := config.Load(cfgPath)
	if err != nil {
		fmt.Printf("integration tests: load config: %v\n", err)
		return m.Run()
	}

	primaryDBName := uniqueTestDBName("easy_admin_it_svc")
	primaryDSN, primaryCleanup, err := createMigratedDatabase(cfg.Database.Primary.DSN, primaryDBName, "migrations/primary", "000001_admin_rbac.up.sql")
	if err != nil {
		fmt.Printf("integration tests: isolated primary %q unavailable (%v); skipping without touching configured databases\n", primaryDBName, err)
		return m.Run()
	}
	logDBName := uniqueTestDBName("easy_admin_it_svc_log")
	logDSN, logCleanup, err := createMigratedDatabase(cfg.Database.Log.DSN, logDBName, "migrations/log", "000001_audit_events.up.sql")
	if err != nil {
		_ = primaryCleanup()
		fmt.Printf("integration tests: isolated log %q unavailable (%v); skipping without touching configured databases\n", logDBName, err)
		return m.Run()
	}

	if err := os.Setenv(config.EnvPrimaryPostgresDSN, primaryDSN); err != nil {
		_ = logCleanup()
		_ = primaryCleanup()
		return m.Run()
	}
	if err := os.Setenv(config.EnvLogPostgresDSN, logDSN); err != nil {
		_ = logCleanup()
		_ = primaryCleanup()
		return m.Run()
	}
	// Real-Redis session tests use a dedicated logical database so they never
	// touch keys that a locally running development server owns in DB 0.
	if err := os.Setenv(config.EnvRedisDB, "15"); err != nil {
		_ = logCleanup()
		_ = primaryCleanup()
		return m.Run()
	}
	isolatedStoresAvailable = true

	code := m.Run()

	if err := logCleanup(); err != nil {
		fmt.Printf("integration tests: cleanup isolated log %q: %v\n", logDBName, err)
	}
	if err := primaryCleanup(); err != nil {
		fmt.Printf("integration tests: cleanup isolated primary %q: %v\n", primaryDBName, err)
	}
	return code
}

// isolationReason returns a skip message when the disposable databases are not
// available, or an empty string when integration tests may run.
func isolationReason() string {
	if !isolatedStoresAvailable {
		return "integration test skipped: disposable databases unavailable; configured application databases are not used"
	}
	return ""
}

func uniqueTestDBName(prefix string) string {
	var buf [4]byte
	if _, err := rand.Read(buf[:]); err != nil {
		return fmt.Sprintf("%s_%d_%d", prefix, os.Getpid(), time.Now().UnixNano())
	}
	return fmt.Sprintf("%s_%d_%s", prefix, os.Getpid(), hex.EncodeToString(buf[:]))
}

// createMigratedDatabase creates a fresh uniquely named database owned by the
// configured role, applies the migration directory's .up.sql files, and
// returns a DSN pointing at the new database plus a cleanup that drops only
// that database.
func createMigratedDatabase(baseDSN, dbName, migrationDir, migrationProbe string) (string, func() error, error) {
	maintenance, err := sql.Open("pgx", baseDSN)
	if err != nil {
		return "", nil, fmt.Errorf("open maintenance connection: %w", err)
	}
	defer maintenance.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := maintenance.PingContext(ctx); err != nil {
		return "", nil, fmt.Errorf("ping maintenance database: %w", err)
	}
	if err := dropDatabase(ctx, maintenance, dbName); err != nil {
		return "", nil, err
	}
	if _, err := maintenance.ExecContext(ctx, fmt.Sprintf("CREATE DATABASE %s", quoteIdent(dbName))); err != nil {
		return "", nil, fmt.Errorf("create isolated database: %w", err)
	}

	isolatedDSN, err := dsnWithDatabase(baseDSN, dbName)
	if err != nil {
		_ = dropDatabase(context.Background(), maintenance, dbName)
		return "", nil, err
	}

	probePath := findRepoFile(filepath.Join(migrationDir, migrationProbe))
	if probePath == "" {
		_ = dropDatabase(context.Background(), maintenance, dbName)
		return "", nil, fmt.Errorf("migration directory %q not found", migrationDir)
	}
	if err := applyUpMigrations(ctx, isolatedDSN, filepath.Dir(probePath)); err != nil {
		_ = dropDatabase(context.Background(), maintenance, dbName)
		return "", nil, err
	}

	cleanup := func() error {
		main, openErr := sql.Open("pgx", baseDSN)
		if openErr != nil {
			return openErr
		}
		defer main.Close()
		cleanupCtx, cancelCleanup := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancelCleanup()
		return dropDatabase(cleanupCtx, main, dbName)
	}
	return isolatedDSN, cleanup, nil
}

func dropDatabase(ctx context.Context, db *sql.DB, dbName string) error {
	if _, err := db.ExecContext(ctx,
		`SELECT pg_terminate_backend(pid) FROM pg_stat_activity WHERE datname = $1 AND pid <> pg_backend_pid()`,
		dbName); err != nil {
		return fmt.Errorf("terminate isolated database connections: %w", err)
	}
	if _, err := db.ExecContext(ctx, fmt.Sprintf("DROP DATABASE IF EXISTS %s", quoteIdent(dbName))); err != nil {
		return fmt.Errorf("drop isolated database: %w", err)
	}
	return nil
}

func applyUpMigrations(ctx context.Context, dsn, migrationRoot string) error {
	files, err := filepath.Glob(filepath.Join(migrationRoot, "*.up.sql"))
	if err != nil {
		return fmt.Errorf("list migrations: %w", err)
	}
	sort.Strings(files)

	db, err := sql.Open("pgx", dsn)
	if err != nil {
		return fmt.Errorf("open isolated database: %w", err)
	}
	defer db.Close()

	for _, file := range files {
		raw, err := os.ReadFile(file)
		if err != nil {
			return fmt.Errorf("read migration %s: %w", filepath.Base(file), err)
		}
		tx, err := db.BeginTx(ctx, nil)
		if err != nil {
			return fmt.Errorf("begin migration transaction: %w", err)
		}
		if _, err := tx.ExecContext(ctx, string(raw)); err != nil {
			_ = tx.Rollback()
			return fmt.Errorf("apply migration %s: %w", filepath.Base(file), err)
		}
		if err := tx.Commit(); err != nil {
			return fmt.Errorf("commit migration %s: %w", filepath.Base(file), err)
		}
	}
	return nil
}

func quoteIdent(name string) string {
	return `"` + name + `"`
}

// TestIsolationGuardSkipsWhenUnavailable proves losing CREATEDB or database
// availability disables integration tests instead of falling back to the
// configured application databases.
func TestIsolationGuardSkipsWhenUnavailable(t *testing.T) {
	if msg := isolationReason(); msg != "" {
		t.Skipf("isolation already unavailable: %s", msg)
	}
	isolatedStoresAvailable = false
	defer func() { isolatedStoresAvailable = true }()
	if msg := isolationReason(); msg == "" {
		t.Fatal("guard allowed integration tests after disposable databases became unavailable")
	}
}

// TestUniqueIsolatedDBNamePerInvocation proves generated database names embed
// the process and random bytes so concurrent invocations cannot collide.
func TestUniqueIsolatedDBNamePerInvocation(t *testing.T) {
	a := uniqueTestDBName("easy_admin_it_svc")
	b := uniqueTestDBName("easy_admin_it_svc")
	if a == b {
		t.Fatalf("generated names must differ, got %q twice", a)
	}
	if len(a) > 63 || len(b) > 63 {
		t.Fatalf("database names exceed PostgreSQL identifier limit: %q %q", a, b)
	}
	if !strings.HasPrefix(a, "easy_admin_it_svc_") || !strings.Contains(a, fmt.Sprintf("_%d_", os.Getpid())) {
		t.Fatalf("name %q must embed prefix and pid", a)
	}
}

func dsnWithDatabase(baseDSN, dbName string) (string, error) {
	cfg, err := pgx.ParseConfig(baseDSN)
	if err != nil {
		return "", fmt.Errorf("parse base dsn: %w", err)
	}
	sslmode := "disable"
	if cfg.TLSConfig != nil {
		sslmode = "require"
	}
	userinfo := url.UserPassword(cfg.User, cfg.Password).String()
	return fmt.Sprintf("postgres://%s@%s:%d/%s?sslmode=%s", userinfo, cfg.Host, cfg.Port, dbName, sslmode), nil
}

// findRepoFile walks up from the package working directory to locate a path
// relative to the repository root.
func findRepoFile(rel string) string {
	wd, err := os.Getwd()
	if err != nil {
		return ""
	}
	dir := wd
	for i := 0; i < 8; i++ {
		candidate := filepath.Join(dir, rel)
		if _, err := os.Stat(candidate); err == nil {
			return candidate
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return ""
}
