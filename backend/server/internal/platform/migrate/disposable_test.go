package migrate_test

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"fmt"
	"net/url"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	_ "github.com/jackc/pgx/v5/stdlib"
)

// disposableDSN points at the current test's unique disposable database after
// newDisposableDB created it.
var disposableDSN string

// newDisposableDB creates a uniquely named disposable database (pid + random
// bytes) and returns a cleanup that drops it. It skips the test when the
// PostgreSQL instance is unavailable, guaranteeing configured application
// databases are never touched by integration tests.
func newDisposableDB(t *testing.T) func() {
	t.Helper()

	base := os.Getenv(maintenanceBaseDSNEnv)
	if base == "" {
		base = "postgres://easy_admin:easy_admin_dev_only@127.0.0.1:55432/postgres?sslmode=disable"
	}

	maintenance, err := sql.Open("pgx", base)
	if err != nil {
		t.Skipf("migrate integration skipped: maintenance connection unavailable (%v)", err)
	}
	defer maintenance.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err := maintenance.PingContext(ctx); err != nil {
		t.Skipf("migrate integration skipped: postgres unavailable (%v)", err)
	}

	var buf [4]byte
	if _, err := rand.Read(buf[:]); err != nil {
		t.Fatalf("random: %v", err)
	}
	dbName := fmt.Sprintf("easy_admin_it_migrate_%d_%s", os.Getpid(), hex.EncodeToString(buf[:]))

	dropStmt := func(db *sql.DB) error {
		if _, err := db.ExecContext(ctx,
			`SELECT pg_terminate_backend(pid) FROM pg_stat_activity WHERE datname = $1 AND pid <> pg_backend_pid()`,
			dbName); err != nil {
			return err
		}
		_, err := db.ExecContext(ctx, fmt.Sprintf(`DROP DATABASE IF EXISTS %s`, quoteIdent(dbName)))
		return err
	}

	parsed, err := pgx.ParseConfig(base)
	if err != nil {
		t.Fatalf("parse base dsn: %v", err)
	}
	sslmode := "disable"
	if parsed.TLSConfig != nil {
		sslmode = "require"
	}
	userinfo := url.UserPassword(parsed.User, parsed.Password).String()
	dsn := fmt.Sprintf("postgres://%s@%s:%d/%s?sslmode=%s", userinfo, parsed.Host, parsed.Port, dbName, sslmode)

	if err := dropStmt(maintenance); err != nil {
		t.Fatalf("drop stale database: %v", err)
	}
	if _, err := maintenance.ExecContext(ctx, fmt.Sprintf("CREATE DATABASE %s", quoteIdent(dbName))); err != nil {
		t.Fatalf("create disposable database: %v", err)
	}
	disposableDSN = dsn

	return func() {
		main, openErr := sql.Open("pgx", base)
		if openErr != nil {
			return
		}
		defer main.Close()
		cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cleanupCancel()
		_ = dropOn(cleanupCtx, main, dbName)
	}
}

func dropOn(ctx context.Context, db *sql.DB, dbName string) error {
	if _, err := db.ExecContext(ctx,
		`SELECT pg_terminate_backend(pid) FROM pg_stat_activity WHERE datname = $1 AND pid <> pg_backend_pid()`,
		dbName); err != nil {
		return err
	}
	_, err := db.ExecContext(ctx, fmt.Sprintf(`DROP DATABASE IF EXISTS %s`, quoteIdent(dbName)))
	return err
}

func quoteIdent(name string) string {
	return `"` + name + `"`
}

func queryCount(ctx context.Context, query string, dest *int) error {
	db, err := sql.Open("pgx", disposableDSN)
	if err != nil {
		return err
	}
	defer db.Close()
	return db.QueryRowContext(ctx, query).Scan(dest)
}

func queryString(ctx context.Context, query string, dest *string) error {
	db, err := sql.Open("pgx", disposableDSN)
	if err != nil {
		return err
	}
	defer db.Close()
	return db.QueryRowContext(ctx, query).Scan(dest)
}

func queryStringList(ctx context.Context, query string, dest *[]string) error {
	db, err := sql.Open("pgx", disposableDSN)
	if err != nil {
		return err
	}
	defer db.Close()
	rows, err := db.QueryContext(ctx, query)
	if err != nil {
		return err
	}
	defer rows.Close()
	out := []string{}
	for rows.Next() {
		var value string
		if err := rows.Scan(&value); err != nil {
			return err
		}
		out = append(out, value)
	}
	*dest = out
	return rows.Err()
}
