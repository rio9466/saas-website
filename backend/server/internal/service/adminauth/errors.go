package adminauth

import (
	"errors"
	"net/http"

	"github.com/rio9466/easy-admin/server/internal/domain/apperr"
	"github.com/rio9466/easy-admin/server/internal/platform/auth"
	"github.com/rio9466/easy-admin/server/internal/repository/logdb"
	"github.com/rio9466/easy-admin/server/internal/repository/primary"
)

// mapError converts repository and platform errors into client-safe AppErrors.
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
	case errors.Is(err, apperr.ErrSessionInvalid), errors.Is(err, auth.ErrSessionInvalid), errors.Is(err, auth.ErrTokenInvalid), errors.Is(err, auth.ErrTokenTypeMismatch):
		return apperr.New(http.StatusUnauthorized, apperr.CodeSessionInvalid, "session invalid", err)
	case errors.Is(err, apperr.ErrRefreshReplay), errors.Is(err, auth.ErrRefreshReplay):
		return apperr.New(http.StatusUnauthorized, apperr.CodeRefreshReplay, "refresh token replay detected", err)
	case errors.Is(err, apperr.ErrForbidden):
		return apperr.New(http.StatusForbidden, apperr.CodeForbidden, "forbidden", err)
	case errors.Is(err, apperr.ErrConflict), errors.Is(err, primary.ErrConflict), errors.Is(err, logdb.ErrConflict):
		return apperr.New(http.StatusConflict, apperr.CodeConflict, "conflict", err)
	case errors.Is(err, apperr.ErrNotFound), errors.Is(err, primary.ErrNotFound), errors.Is(err, logdb.ErrNotFound):
		return apperr.New(http.StatusNotFound, apperr.CodeNotFound, "not found", err)
	case errors.Is(err, apperr.ErrLastSuperAdmin), errors.Is(err, primary.ErrLastSuperAdmin):
		return apperr.New(http.StatusConflict, apperr.CodeLastSuperAdmin, "cannot modify the last enabled super administrator", err)
	case errors.Is(err, apperr.ErrSelfProtection):
		return apperr.New(http.StatusConflict, apperr.CodeSelfProtection, "operation not allowed on the current administrator", err)
	case errors.Is(err, apperr.ErrBootstrapConflict), errors.Is(err, primary.ErrBootstrapAlreadyCompleted):
		return apperr.New(http.StatusConflict, apperr.CodeBootstrapConflict, "bootstrap already completed", err)
	case errors.Is(err, apperr.ErrInvalidPassword), errors.Is(err, auth.ErrPasswordTooShort), errors.Is(err, auth.ErrPasswordTooLong), errors.Is(err, auth.ErrPasswordMismatch):
		return apperr.New(http.StatusBadRequest, apperr.CodeInvalidPassword, "invalid password", err)
	case errors.Is(err, apperr.ErrAccountDisabled):
		return apperr.New(http.StatusForbidden, apperr.CodeAccountDisabled, "account disabled", err)
	case errors.Is(err, apperr.ErrAuditUnavailable):
		return apperr.New(http.StatusServiceUnavailable, apperr.CodeAuditUnavailable, "audit unavailable", err)
	case errors.Is(err, apperr.ErrDependencyUnavailable), errors.Is(err, auth.ErrUntrustedOrigin):
		return apperr.New(http.StatusServiceUnavailable, apperr.CodeDependencyUnavailable, "dependency unavailable", err)
	default:
		return apperr.New(http.StatusServiceUnavailable, apperr.CodeDependencyUnavailable, "dependency unavailable", err)
	}
}

func unauthorizedCredentials() error {
	return apperr.New(http.StatusUnauthorized, apperr.CodeUnauthorized, "unauthorized", apperr.ErrUnauthorized)
}

func forbidden() error {
	return apperr.New(http.StatusForbidden, apperr.CodeForbidden, "forbidden", apperr.ErrForbidden)
}

func validationError(message string) error {
	if message == "" {
		message = "validation failed"
	}
	return apperr.New(http.StatusBadRequest, apperr.CodeValidation, message, apperr.ErrValidation)
}
