package analyticssvc

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/rio9466/easy-admin/server/internal/domain/apperr"
)

// mapError converts repository/domain errors into client-safe AppErrors.
// Unknown errors map to dependency-unavailable so no internal detail leaks.
func mapError(err error) error {
	if err == nil {
		return nil
	}
	var appErr *apperr.AppError
	if errors.As(err, &appErr) {
		return appErr
	}
	switch {
	case errors.Is(err, apperr.ErrValidation):
		return apperr.New(http.StatusBadRequest, apperr.CodeValidation, "validation failed", err)
	case errors.Is(err, apperr.ErrUnauthorized):
		return apperr.New(http.StatusUnauthorized, apperr.CodeUnauthorized, "unauthorized", err)
	case errors.Is(err, apperr.ErrForbidden):
		return apperr.New(http.StatusForbidden, apperr.CodeForbidden, "forbidden", err)
	default:
		return apperr.New(http.StatusServiceUnavailable, apperr.CodeDependencyUnavailable, "dependency unavailable", err)
	}
}

func validationError(message string) error {
	return apperr.New(http.StatusBadRequest, apperr.CodeValidation, "validation failed", fmt.Errorf("%w: %s", apperr.ErrValidation, message))
}
