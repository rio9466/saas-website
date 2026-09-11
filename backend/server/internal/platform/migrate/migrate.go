// Package migrate applies explicit SQL migrations from the migrations/
// directory with the existing schema_migrations bookkeeping. It is used by the
// single server startup entry point so the process never listens for HTTP
// before both primary and log schemas are current.
//
// Rules:
//   - Explicit SQL files only; AutoMigrate is never used.
//   - "up" migrations are idempotent: applied versions are skipped, and each
//     file runs inside one transaction together with its schema_migrations row.
//   - Only the requested target database is touched; failures return an error
//     and leave the version bookkeeping unchanged (transaction rollback).
package migrate

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	// pgx stdlib registers the "pgx" database/sql driver used below.
	_ "github.com/jackc/pgx/v5/stdlib"
)

// directionUp is the only direction the startup path applies.
const directionUp = "up"

// pingTimeout bounds connectivity checks performed on the migration connection.
const pingTimeout = 30 * time.Second

// Result summarizes one Run call.
type Result struct {
	// Target is the migration target name, e.g. "primary" or "log".
	Target string
	// Applied lists the versions applied by this run (empty when current).
	Applied []string
	// Skipped counts already-applied versions found during this run.
	Skipped int
}

// Run applies all pending "up" migrations for target ("primary" or "log")
// against dsn, using the .up.sql files under migrationRoot/<target>. It is
// safe to call repeatedly; applied versions are skipped without error.
func Run(ctx context.Context, dsn, migrationRoot, target string) (Result, error) {
	dir := filepath.Join(migrationRoot, target)
	entries, err := os.ReadDir(dir)
	if err != nil {
		return Result{}, fmt.Errorf("read migration directory: %w", err)
	}

	// Validate and order all migration files before touching the database so a
	// bad filename fails closed without opening a connection.
	var files []string
	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() || !strings.HasSuffix(name, "."+directionUp+".sql") {
			continue
		}
		if versionOf(name) == "" {
			return Result{}, fmt.Errorf("invalid migration filename %q", name)
		}
		files = append(files, name)
	}
	sort.Strings(files)

	db, err := sql.Open("pgx", dsn)
	if err != nil {
		return Result{}, errors.New("open migration database failed")
	}
	defer db.Close()

	pingCtx, cancel := context.WithTimeout(ctx, pingTimeout)
	defer cancel()
	if err := db.PingContext(pingCtx); err != nil {
		return Result{}, errors.New("migration database unreachable")
	}

	if _, err := db.ExecContext(pingCtx, `
CREATE TABLE IF NOT EXISTS schema_migrations (
    version TEXT PRIMARY KEY,
    applied_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
)`); err != nil {
		return Result{}, errors.New("ensure schema_migrations failed")
	}

	result := Result{Target: target, Applied: []string{}}
	for _, file := range files {
		version := versionOf(file)

		applied, err := versionApplied(ctx, db, version)
		if err != nil {
			return result, err
		}
		if applied {
			result.Skipped++
			continue
		}

		sqlBytes, err := os.ReadFile(filepath.Join(dir, file))
		if err != nil {
			return result, fmt.Errorf("read migration: %w", err)
		}

		tx, err := db.BeginTx(ctx, nil)
		if err != nil {
			return result, errors.New("begin migration transaction failed")
		}
		if _, err := tx.ExecContext(ctx, string(sqlBytes)); err != nil {
			_ = tx.Rollback()
			return result, fmt.Errorf("apply %s failed", version)
		}
		if _, err := tx.ExecContext(ctx,
			`INSERT INTO schema_migrations(version) VALUES ($1) ON CONFLICT DO NOTHING`,
			version); err != nil {
			_ = tx.Rollback()
			return result, fmt.Errorf("record %s failed", version)
		}
		if err := tx.Commit(); err != nil {
			return result, errors.New("commit migration failed")
		}
		result.Applied = append(result.Applied, version)
	}
	return result, nil
}

func versionOf(name string) string {
	name = strings.TrimSuffix(name, ".up.sql")
	return strings.TrimSuffix(name, ".down.sql")
}

func versionApplied(ctx context.Context, db *sql.DB, version string) (bool, error) {
	var one int
	err := db.QueryRowContext(ctx,
		`SELECT 1 FROM schema_migrations WHERE version = $1`, version).Scan(&one)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, errors.New("query schema_migrations failed")
	}
	return true, nil
}
