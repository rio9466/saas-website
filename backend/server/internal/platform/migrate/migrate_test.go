package migrate_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/rio9466/easy-admin/server/internal/platform/migrate"
)

func writeMigration(t *testing.T, dir, name, sql string) {
	t.Helper()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, name), []byte(sql), 0o600); err != nil {
		t.Fatalf("write migration: %v", err)
	}
}

func TestRunMissingMigrationDirFails(t *testing.T) {
	root := t.TempDir()
	_, err := migrate.Run(context.Background(), "irrelevant", root, "primary")
	if err == nil {
		t.Fatal("expected error for missing migration directory")
	}
}

func TestRunInvalidFilenameFailsBeforeConnecting(t *testing.T) {
	root := t.TempDir()
	writeMigration(t, filepath.Join(root, "primary"), ".up.sql", "SELECT 1;")
	// DSN is intentionally invalid; the filename check must fail first.
	_, err := migrate.Run(context.Background(), "irrelevant-dsn", root, "primary")
	if err == nil {
		t.Fatal("expected error for invalid migration filename")
	}
	if got := err.Error(); got != `invalid migration filename ".up.sql"` {
		t.Fatalf("error = %q, want invalid migration filename", got)
	}
}

func TestRunIgnoresNonUpFiles(t *testing.T) {
	// Only a .down.sql file exists: nothing to apply, but Run still needs a
	// database. The directory listing path is validated via the missing-DB
	// error arriving after filename validation, proving non-up files were
	// skipped during validation.
	root := t.TempDir()
	writeMigration(t, filepath.Join(root, "log"), "000001_old.down.sql", "SELECT 1;")
	_, err := migrate.Run(context.Background(), "irrelevant-dsn", root, "log")
	if err == nil {
		t.Fatal("expected database error even with only down files")
	}
	if got := err.Error(); got != "migration database unreachable" {
		t.Fatalf("error = %q, want migration database unreachable", got)
	}
}
