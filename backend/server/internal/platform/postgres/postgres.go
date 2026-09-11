package postgres

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/rio9466/easy-admin/server/internal/config"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// DB wraps a GORM connection with pool settings and cleanup.
type DB struct {
	gormDB *gorm.DB
}

// Open creates a PostgreSQL connection, configures the pool, and verifies
// connectivity within the configured startup timeout.
func Open(ctx context.Context, cfg config.PostgresConfig) (*DB, error) {
	gormDB, err := gorm.Open(postgres.Open(cfg.DSN), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		return nil, fmt.Errorf("open postgres: %w", err)
	}

	sqlDB, err := gormDB.DB()
	if err != nil {
		return nil, fmt.Errorf("unwrap postgres sql db: %w", err)
	}

	sqlDB.SetMaxOpenConns(cfg.MaxOpenConns)
	sqlDB.SetMaxIdleConns(cfg.MaxIdleConns)
	sqlDB.SetConnMaxLifetime(cfg.ConnMaxLifetime)
	sqlDB.SetConnMaxIdleTime(cfg.ConnMaxIdleTime)

	db := &DB{gormDB: gormDB}

	startupCtx, cancel := context.WithTimeout(ctx, cfg.StartupTimeout)
	defer cancel()

	if err := db.Ping(startupCtx); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("postgres startup ping: %w", err)
	}

	return db, nil
}

// GORM returns the underlying GORM handle for repository use.
func (db *DB) GORM() *gorm.DB {
	if db == nil {
		return nil
	}
	return db.gormDB
}

// SQL returns the underlying database/sql pool.
func (db *DB) SQL() (*sql.DB, error) {
	if db == nil || db.gormDB == nil {
		return nil, fmt.Errorf("postgres is not initialized")
	}
	sqlDB, err := db.gormDB.DB()
	if err != nil {
		return nil, fmt.Errorf("unwrap postgres sql db: %w", err)
	}
	return sqlDB, nil
}

// Ping verifies database connectivity using the provided context deadline.
func (db *DB) Ping(ctx context.Context) error {
	if db == nil || db.gormDB == nil {
		return fmt.Errorf("postgres is not initialized")
	}

	sqlDB, err := db.gormDB.DB()
	if err != nil {
		return fmt.Errorf("unwrap postgres sql db: %w", err)
	}

	if err := sqlDB.PingContext(ctx); err != nil {
		return fmt.Errorf("ping postgres: %w", err)
	}
	return nil
}

// Close releases the underlying connection pool.
func (db *DB) Close() error {
	if db == nil || db.gormDB == nil {
		return nil
	}

	sqlDB, err := db.gormDB.DB()
	if err != nil {
		return fmt.Errorf("unwrap postgres sql db: %w", err)
	}
	if err := sqlDB.Close(); err != nil {
		return fmt.Errorf("close postgres: %w", err)
	}
	return nil
}
