package usersvc_test

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
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	_ "github.com/jackc/pgx/v5/stdlib"
	goredis "github.com/redis/go-redis/v9"
	"github.com/rio9466/easy-admin/server/internal/config"
)

// Integration tests in this package run against unique disposable primary and
// log databases plus a dedicated Redis logical database. When provisioning
// fails, tests skip explicitly and never touch configured application data.
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

	primaryName := uniqueTestDBName("easy_admin_it_usvc")
	primaryDSN, primaryCleanup, err := createMigratedDatabase(cfg.Database.Primary.DSN, primaryName, "migrations/primary", "000001_admin_rbac.up.sql")
	if err != nil {
		fmt.Printf("integration tests: isolated primary %q unavailable (%v); skipping without touching configured databases\n", primaryName, err)
		return m.Run()
	}
	logName := uniqueTestDBName("easy_admin_it_usvc_log")
	logDSN, logCleanup, err := createMigratedDatabase(cfg.Database.Log.DSN, logName, "migrations/log", "000001_audit_events.up.sql")
	if err != nil {
		_ = primaryCleanup()
		fmt.Printf("integration tests: isolated log %q unavailable (%v); skipping without touching configured databases\n", logName, err)
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
	// Dedicated Redis DB: never touches keys a local development server owns.
	if err := os.Setenv(config.EnvRedisDB, "14"); err != nil {
		_ = logCleanup()
		_ = primaryCleanup()
		return m.Run()
	}
	flushDedicatedRedis(cfg.Redis.Addr, cfg.Redis.Password, 14)
	isolatedStoresAvailable = true

	code := m.Run()

	if err := logCleanup(); err != nil {
		fmt.Printf("integration tests: cleanup isolated log %q: %v\n", logName, err)
	}
	if err := primaryCleanup(); err != nil {
		fmt.Printf("integration tests: cleanup isolated primary %q: %v\n", primaryName, err)
	}
	return code
}

// flushDedicatedRedis clears only the dedicated test logical database. It is
// exclusively owned by this test package (never a development server), so the
// flush removes stale rate/session/verification keys between invocations and
// touches nothing else.
func flushDedicatedRedis(addr, password string, db int) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	rdb := goredis.NewClient(&goredis.Options{Addr: addr, Password: password, DB: db})
	defer func() { _ = rdb.Close() }()
	if err := rdb.FlushDB(ctx).Err(); err != nil {
		fmt.Printf("integration tests: flush dedicated redis db failed: %v\n", err)
	}
}

// isolationReason returns a skip message when the disposable stores are not
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
		cleanupCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		return dropDatabase(cleanupCtx, main, dbName)
	}
	return isolatedDSN, cleanup, nil
}

func dropDatabase(ctx context.Context, db *sql.DB, dbName string) error {
	if _, err := db.ExecContext(ctx,
		`SELECT pg_terminate_backend(pid) FROM pg_stat_activity WHERE datname = $1 AND pid <> pg_backend_pid()`, dbName); err != nil {
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

func quoteIdent(name string) string { return `"` + name + `"` }

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

func mustSkipIfUnavailable(t *testing.T) {
	t.Helper()
	if msg := isolationReason(); msg != "" {
		t.Skipf("%s", msg)
	}
}
