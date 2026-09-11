package main

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/rio9466/easy-admin/server/internal/config"
)

func TestOpenPostgresRejectsInvalidDSNWithoutLeakingMarker(t *testing.T) {
	t.Parallel()

	const marker = "definitely-secret-invalid-dsn-SRV001R1-unit"

	cfg := config.PostgresConfig{
		DSN:             marker,
		MaxOpenConns:    1,
		MaxIdleConns:    1,
		ConnMaxLifetime: time.Minute,
		ConnMaxIdleTime: time.Minute,
		StartupTimeout:  500 * time.Millisecond,
		PingTimeout:     200 * time.Millisecond,
	}

	db, err := openPostgres(context.Background(), cfg, msgConnectPrimaryPostgresFailed)
	if db != nil {
		t.Fatal("expected nil db on invalid DSN")
	}
	if err == nil {
		t.Fatal("expected error on invalid DSN")
	}
	if err.Error() != msgConnectPrimaryPostgresFailed {
		t.Fatalf("error = %q, want %q", err.Error(), msgConnectPrimaryPostgresFailed)
	}
	if strings.Contains(err.Error(), marker) {
		t.Fatalf("process-boundary error leaked marker: %v", err)
	}
}

// TestRunFailsClosedWhenPrimaryDatabaseUnavailable proves the startup path
// aborts before any HTTP listener exists when PostgreSQL is unreachable, and
// that the error carries no DSN material.
func TestRunFailsClosedWhenPrimaryDatabaseUnavailable(t *testing.T) {
	const marker = "definitely-secret-invalid-dsn-SRV003-run"

	path := writeTempValidConfig(t)
	t.Setenv("PRIMARY_POSTGRES_DSN", marker)

	runErr := run(path)
	if runErr == nil {
		t.Fatal("expected run to fail with invalid primary DSN")
	}
	if !strings.Contains(runErr.Error(), msgConnectPrimaryPostgresFailed) {
		t.Fatalf("run error = %q, want fixed message containing %q", runErr.Error(), msgConnectPrimaryPostgresFailed)
	}
	if strings.Contains(runErr.Error(), marker) {
		t.Fatalf("startup error leaked DSN marker: %v", runErr)
	}
}

// TestRunFailsClosedWhenMigrationRootMissing proves a broken migration setup
// aborts startup instead of starting HTTP without the required schema. The
// database in this test is a real, unique, disposable PostgreSQL database, so
// the failure is exercised at the migration step, not the connection step.
func TestRunFailsClosedWhenMigrationRootMissing(t *testing.T) {
	dsn := requireIsolatedPostgres(t)

	path := writeTempValidConfigWithDSNs(t, dsn, dsn)

	workdir := t.TempDir() // has no migrations/ directory
	t.Chdir(workdir)

	runErr := run(path)
	if runErr == nil {
		t.Fatal("expected run to fail when migrations are missing")
	}
	if !strings.Contains(runErr.Error(), "primary migrations") {
		t.Fatalf("run error = %q, want primary migrations failure", runErr.Error())
	}
}

// TestRunFailsClosedWhenMigrationSQLBroken proves a corrupt migration aborts
// startup with a non-zero failure path (HTTP never starts).
func TestRunFailsClosedWhenMigrationSQLBroken(t *testing.T) {
	dsn := requireIsolatedPostgres(t)

	workdir := t.TempDir()
	migrationDir := filepath.Join(workdir, "migrations", "primary")
	if err := os.MkdirAll(migrationDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	broken := "THIS IS NOT VALID SQL;"
	if err := os.WriteFile(filepath.Join(migrationDir, "000001_broken.up.sql"), []byte(broken), 0o600); err != nil {
		t.Fatalf("write migration: %v", err)
	}
	t.Chdir(workdir)

	path := writeTempValidConfigWithDSNs(t, dsn, dsn)

	runErr := run(path)
	if runErr == nil {
		t.Fatal("expected run to fail on broken migration SQL")
	}
	if !strings.Contains(runErr.Error(), "000001_broken") {
		t.Fatalf("run error = %q, want broken migration version in message", runErr.Error())
	}
	if strings.Contains(runErr.Error(), dsn) {
		t.Fatalf("migration error leaked DSN: %v", runErr)
	}
}

func TestApplyMigrationsSurfacesTargetName(t *testing.T) {
	err := func() error {
		_, err := applyMigrations(context.Background(), "irrelevant-dsn", "primary")
		return err
	}()
	if err == nil {
		t.Fatal("expected error for unreachable migration database")
	}
	if !strings.Contains(err.Error(), "primary migrations") {
		t.Fatalf("error = %q, want target name in message", err.Error())
	}
}

func writeTempValidConfig(t *testing.T) string {
	t.Helper()
	return writeTempValidConfigWithDSNs(t,
		"postgres://easy_admin:change-me@127.0.0.1:55432/easy_admin?sslmode=disable",
		"postgres://easy_admin_logs:change-me@127.0.0.1:55433/easy_admin_logs?sslmode=disable",
	)
}

func writeTempValidConfigWithDSNs(t *testing.T, primaryDSN, logDSN string) string {
	t.Helper()

	contents := `
[server]
addr = "127.0.0.1:0"
read_header_timeout = "5s"
read_timeout = "15s"
write_timeout = "15s"
idle_timeout = "60s"
shutdown_timeout = "2s"

[database.primary]
dsn = "` + primaryDSN + `"
max_open_conns = 2
max_idle_conns = 1
conn_max_lifetime = "30m"
conn_max_idle_time = "5m"
startup_timeout = "1s"
ping_timeout = "500ms"

[database.log]
dsn = "` + logDSN + `"
max_open_conns = 2
max_idle_conns = 1
conn_max_lifetime = "30m"
conn_max_idle_time = "5m"
startup_timeout = "1s"
ping_timeout = "500ms"

[redis]
addr = "127.0.0.1:56379"
password = ""
db = 0
pool_size = 4
dial_timeout = "1s"
read_timeout = "1s"
write_timeout = "1s"
startup_timeout = "1s"
ping_timeout = "500ms"

[auth]
environment = "development"
jwt_secret = "change-me-to-a-long-random-secret-at-least-32"
jwt_issuer = "easy-admin"
jwt_audience = "easy-admin-admin"
refresh_cookie_name = "ea_admin_refresh"
refresh_cookie_path = "/api/v1/admin/auth"
refresh_cookie_secure = false
trusted_origins = ["http://127.0.0.1:8848"]
bcrypt_cost = 12

[log]
level = "error"
`
	path := filepath.Join(t.TempDir(), "config.toml")
	if err := os.WriteFile(path, []byte(contents), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}
	return path
}
