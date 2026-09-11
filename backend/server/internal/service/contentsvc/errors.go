package contentsvc

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/rio9466/easy-admin/server/internal/domain/apperr"
	"github.com/rio9466/easy-admin/server/internal/domain/content"
	"github.com/rio9466/easy-admin/server/internal/repository/logdb"
	"github.com/rio9466/easy-admin/server/internal/repository/primary"
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
	case errors.Is(err, apperr.ErrValidation),
		errors.Is(err, content.ErrInvalidURL),
		errors.Is(err, content.ErrInvalidMoney):
		return apperr.New(http.StatusBadRequest, apperr.CodeValidation, "validation failed", err)
	case errors.Is(err, apperr.ErrUnauthorized):
		return apperr.New(http.StatusUnauthorized, apperr.CodeUnauthorized, "unauthorized", err)
	case errors.Is(err, apperr.ErrForbidden):
		return apperr.New(http.StatusForbidden, apperr.CodeForbidden, "forbidden", err)
	case errors.Is(err, apperr.ErrConflict), errors.Is(err, primary.ErrConflict), errors.Is(err, logdb.ErrConflict):
		return apperr.New(http.StatusConflict, apperr.CodeConflict, "conflict", err)
	case errors.Is(err, apperr.ErrNotFound), errors.Is(err, primary.ErrNotFound), errors.Is(err, logdb.ErrNotFound):
		return apperr.New(http.StatusNotFound, apperr.CodeNotFound, "not found", err)
	case errors.Is(err, apperr.ErrSettingsConflict), errors.Is(err, primary.ErrSettingsVersionConflict):
		return apperr.New(http.StatusConflict, apperr.CodeSettingsConflict, "settings conflict", err)
	case errors.Is(err, apperr.ErrAuditUnavailable):
		return apperr.New(http.StatusServiceUnavailable, apperr.CodeAuditUnavailable, "audit unavailable", err)
	case errors.Is(err, apperr.ErrDependencyUnavailable):
		return apperr.New(http.StatusServiceUnavailable, apperr.CodeDependencyUnavailable, "dependency unavailable", err)
	default:
		return apperr.New(http.StatusServiceUnavailable, apperr.CodeDependencyUnavailable, "dependency unavailable", err)
	}
}

func validationError(message string) error {
	return apperr.New(http.StatusBadRequest, apperr.CodeValidation, "validation failed", fmt.Errorf("%w: %s", apperr.ErrValidation, message))
}
