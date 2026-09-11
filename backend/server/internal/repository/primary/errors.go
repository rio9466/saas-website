package primary

import (
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5/pgconn"
	"gorm.io/gorm"
)

// Sentinel errors detectable with errors.Is by the service layer.
var (
	ErrConflict                  = errors.New("conflict")
	ErrNotFound                  = errors.New("not found")
	ErrLastSuperAdmin            = errors.New("last super admin")
	ErrBootstrapAlreadyCompleted = errors.New("bootstrap already completed")
	// Business-user domain invariants.
	ErrValidationNoop      = errors.New("points adjustment must change at least one balance")
	ErrConsumptionNegative = errors.New("consumption points cannot be reduced")
	ErrBalanceBelowZero    = errors.New("available points cannot become negative")
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
