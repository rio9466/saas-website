package main

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

// Startup fail-closed tests exercise real PostgreSQL but never the configured
// application databases: each test provisions its own uniquely named temporary
// database (pid + random bytes) from the maintenance DSN and drops it again.
// When PostgreSQL is unavailable the tests skip instead of falling back.
var isolatedPostgresDSN string

func TestMain(m *testing.M) {
	base := os.Getenv("EASY_ADMIN_TEST_POSTGRES_DSN")
	if base == "" {
		base = "postgres://easy_admin:easy_admin_dev_only@127.0.0.1:55432/postgres?sslmode=disable"
	}
	dsn, cleanup, err := createDisposableDatabase(base)
	if err != nil {
		fmt.Printf("startup tests: disposable postgres unavailable (%v); skipping database-backed cases\n", err)
		os.Exit(m.Run())
	}
	isolatedPostgresDSN = dsn
	code := m.Run()
	if err := cleanup(); err != nil {
		fmt.Printf("startup tests: cleanup disposable database: %v\n", err)
	}
	os.Exit(code)
}

// requireIsolatedPostgres skips when no disposable database could be created,
// guaranteeing configured application databases are never touched.
func requireIsolatedPostgres(t *testing.T) string {
	t.Helper()
	if isolatedPostgresDSN == "" {
		t.Skip("startup test skipped: disposable postgres unavailable; configured application databases are not used")
	}
	return isolatedPostgresDSN
}

func createDisposableDatabase(baseDSN string) (string, func() error, error) {
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

	var buf [4]byte
	if _, err := rand.Read(buf[:]); err != nil {
		return "", nil, fmt.Errorf("random: %w", err)
	}
	dbName := fmt.Sprintf("easy_admin_it_server_%d_%s", os.Getpid(), hex.EncodeToString(buf[:]))

	if _, err := maintenance.ExecContext(ctx,
		`SELECT pg_terminate_backend(pid) FROM pg_stat_activity WHERE datname = $1 AND pid <> pg_backend_pid()`,
		dbName); err != nil {
		return "", nil, fmt.Errorf("terminate stale connections: %w", err)
	}
	if _, err := maintenance.ExecContext(ctx, fmt.Sprintf(`DROP DATABASE IF EXISTS %s`, quoteIdent(dbName))); err != nil {
		return "", nil, fmt.Errorf("drop stale database: %w", err)
	}
	if _, err := maintenance.ExecContext(ctx, fmt.Sprintf("CREATE DATABASE %s", quoteIdent(dbName))); err != nil {
		return "", nil, fmt.Errorf("create disposable database: %w", err)
	}

	parsed, err := pgx.ParseConfig(baseDSN)
	if err != nil {
		_ = dropDatabase(ctx, maintenance, dbName)
		return "", nil, fmt.Errorf("parse base dsn: %w", err)
	}
	sslmode := "disable"
	if parsed.TLSConfig != nil {
		sslmode = "require"
	}
	userinfo := url.UserPassword(parsed.User, parsed.Password).String()
	dsn := fmt.Sprintf("postgres://%s@%s:%d/%s?sslmode=%s", userinfo, parsed.Host, parsed.Port, dbName, sslmode)

	cleanup := func() error {
		main, openErr := sql.Open("pgx", baseDSN)
		if openErr != nil {
			return openErr
		}
		defer main.Close()
		return dropDatabase(context.Background(), main, dbName)
	}
	return dsn, cleanup, nil
}

func dropDatabase(ctx context.Context, db *sql.DB, dbName string) error {
	if _, err := db.ExecContext(ctx,
		`SELECT pg_terminate_backend(pid) FROM pg_stat_activity WHERE datname = $1 AND pid <> pg_backend_pid()`,
		dbName); err != nil {
		return fmt.Errorf("terminate disposable database connections: %w", err)
	}
	if _, err := db.ExecContext(ctx, fmt.Sprintf(`DROP DATABASE IF EXISTS %s`, quoteIdent(dbName))); err != nil {
		return fmt.Errorf("drop disposable database: %w", err)
	}
	return nil
}

func quoteIdent(name string) string {
	return `"` + name + `"`
}
