package migrate_test

import (
	"context"
	"os"
	"testing"

	"github.com/rio9466/easy-admin/server/internal/platform/migrate"
)

// Integration tests apply the real project migrations to a unique disposable
// database and verify idempotency and failure rollback. They skip when
// PostgreSQL is unavailable; configured application databases are never used.
const maintenanceBaseDSNEnv = "EASY_ADMIN_TEST_POSTGRES_DSN"

func maintenanceBaseDSN() string {
	if base := os.Getenv(maintenanceBaseDSNEnv); base != "" {
		return base
	}
	return "postgres://easy_admin:easy_admin_dev_only@127.0.0.1:55432/postgres?sslmode=disable"
}

// findRepoRoot walks up from the test working directory until the migrations
// directory of the server module is found.
func findRepoRoot(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	for i := 0; i < 8; i++ {
		if _, err := os.Stat(dir + "/migrations/primary"); err == nil {
			// migrate.Run joins root + target, so hand back the migrations dir.
			return dir + "/migrations"
		}
		parent := parentDir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	t.Fatal("migrations directory not found")
	return ""
}

func parentDir(dir string) string {
	idx := len(dir) - 1
	for idx > 0 && dir[idx] != '/' {
		idx--
	}
	return dir[:idx]
}

func TestRunRealMigrationsAreIdempotent(t *testing.T) {
	root := findRepoRoot(t)

	cleanup := newDisposableDB(t)
	defer cleanup()

	ctx := context.Background()
	first, err := migrate.Run(ctx, disposableDSN, root, "primary")
	if err != nil {
		t.Fatalf("first run: %v", err)
	}
	if len(first.Applied) != 7 {
		t.Fatalf("first run applied %d migrations, want 7: %+v", len(first.Applied), first)
	}

	second, err := migrate.Run(ctx, disposableDSN, root, "primary")
	if err != nil {
		t.Fatalf("second run: %v", err)
	}
	if len(second.Applied) != 0 || second.Skipped != len(first.Applied) {
		t.Fatalf("second run = %+v, want no-op with all versions skipped", second)
	}

	// A fresh database must end up with exactly one seeded role: super_admin.
	// The permissions catalog stays full; no admin/finance roles are seeded.
	var roles []string
	if err := queryStringList(ctx, `SELECT lower(code) FROM roles ORDER BY id`, &roles); err != nil {
		t.Fatalf("query roles: %v", err)
	}
	if len(roles) != 1 || roles[0] != "super_admin" {
		t.Fatalf("fresh database roles = %v, want exactly [super_admin]", roles)
	}
	var perms int
	if err := queryCount(ctx, `SELECT COUNT(*) FROM permissions`, &perms); err != nil {
		t.Fatalf("count permissions: %v", err)
	}
	if perms == 0 {
		t.Fatal("permissions catalog must stay populated")
	}
}

func TestRunRealLogMigrationsAreIdempotent(t *testing.T) {
	root := findRepoRoot(t)

	cleanup := newDisposableDB(t)
	defer cleanup()

	ctx := context.Background()
	if _, err := migrate.Run(ctx, disposableDSN, root, "log"); err != nil {
		t.Fatalf("first log run: %v", err)
	}
	second, err := migrate.Run(ctx, disposableDSN, root, "log")
	if err != nil {
		t.Fatalf("second log run: %v", err)
	}
	if len(second.Applied) != 0 {
		t.Fatalf("second log run applied %v, want no-op", second.Applied)
	}
}

func TestRunBrokenMigrationLeavesSchemaMigrationsClean(t *testing.T) {
	root := t.TempDir()
	brokenDir := root + "/log"
	if err := os.MkdirAll(brokenDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	good := "CREATE TABLE marker_ok (id INT);"
	if err := os.WriteFile(brokenDir+"/000001_good.up.sql", []byte(good), 0o600); err != nil {
		t.Fatalf("write good migration: %v", err)
	}
	broken := "CREATE TABLE marker_broken ("
	if err := os.WriteFile(brokenDir+"/000002_broken.up.sql", []byte(broken), 0o600); err != nil {
		t.Fatalf("write broken migration: %v", err)
	}

	cleanup := newDisposableDB(t)
	defer cleanup()

	ctx := context.Background()
	if _, err := migrate.Run(ctx, disposableDSN, root, "log"); err == nil {
		t.Fatal("expected broken migration to fail")
	}

	// The good migration committed before the broken one, so schema_migrations
	// must contain exactly 000001_good and no partial state from 000002.
	var count int
	if err := queryCount(ctx, `SELECT COUNT(*) FROM schema_migrations`, &count); err != nil {
		t.Fatalf("query schema_migrations: %v", err)
	}
	if count != 1 {
		t.Fatalf("schema_migrations rows = %d, want exactly the applied 000001_good", count)
	}
	var version string
	if err := queryString(ctx, `SELECT version FROM schema_migrations LIMIT 1`, &version); err != nil {
		t.Fatalf("query version: %v", err)
	}
	if version != "000001_good" {
		t.Fatalf("version = %q, want 000001_good", version)
	}

	// Fixing the broken file and re-running must complete idempotently.
	if err := os.WriteFile(brokenDir+"/000002_broken.up.sql", []byte("CREATE TABLE marker_fixed (id INT);"), 0o600); err != nil {
		t.Fatalf("rewrite migration: %v", err)
	}
	res, err := migrate.Run(ctx, disposableDSN, root, "log")
	if err != nil {
		t.Fatalf("re-run after fix: %v", err)
	}
	if len(res.Applied) != 1 || res.Applied[0] != "000002_broken" {
		t.Fatalf("re-run = %+v, want only 000002_broken applied", res)
	}
}
