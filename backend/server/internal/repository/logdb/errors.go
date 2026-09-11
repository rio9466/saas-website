package logdb

import (
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5/pgconn"
	"gorm.io/gorm"
)

// Sentinel errors detectable with errors.Is by the service layer.
var (
	ErrConflict = errors.New("conflict")
	ErrNotFound = errors.New("not found")
)

func wrapDBError(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return fmt.Errorf("%w: %w", ErrNotFound, err)
	}
	if isUniqueViolation(err) {
		return fmt.Errorf("%w: %w", ErrConflict, err)
	}
	return err
}

func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		return pgErr.Code == "23505"
	}
	return false
}
