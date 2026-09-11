package usersvc

import (
	"errors"
	"net/http"

	"github.com/rio9466/easy-admin/server/internal/domain/apperr"
	userdomain "github.com/rio9466/easy-admin/server/internal/domain/user"
	"github.com/rio9466/easy-admin/server/internal/platform/auth"
	"github.com/rio9466/easy-admin/server/internal/platform/ratelimit"
	"github.com/rio9466/easy-admin/server/internal/repository/logdb"
	"github.com/rio9466/easy-admin/server/internal/repository/primary"
)

// mapError converts repository/platform/domain errors into client-safe
// AppErrors. Unknown errors map to dependency-unavailable so no raw internal
// detail ever reaches a response.
func mapError(err error) error {
	if err == nil {
		return nil
	}

	var appErr *apperr.AppError
	if errors.As(err, &appErr) {
		return appErr
	}

	switch {
	case errors.Is(err, apperr.ErrValidation), errors.Is(err, userdomain.ErrDecimalInvalid),
		errors.Is(err, primary.ErrValidationNoop), errors.Is(err, primary.ErrConsumptionNegative):
		return apperr.New(http.StatusBadRequest, apperr.CodeValidation, "validation failed", err)
	case errors.Is(err, apperr.ErrUnauthorized):
		return apperr.New(http.StatusUnauthorized, apperr.CodeUnauthorized, "unauthorized", err)
	case errors.Is(err, apperr.ErrSessionInvalid), errors.Is(err, auth.ErrSessionInvalid),
		errors.Is(err, auth.ErrTokenInvalid), errors.Is(err, auth.ErrTokenTypeMismatch):
		return apperr.New(http.StatusUnauthorized, apperr.CodeSessionInvalid, "session invalid", err)
	case errors.Is(err, apperr.ErrRefreshReplay), errors.Is(err, auth.ErrRefreshReplay):
		return apperr.New(http.StatusUnauthorized, apperr.CodeRefreshReplay, "refresh token replay detected", err)
	case errors.Is(err, apperr.ErrForbidden):
		return apperr.New(http.StatusForbidden, apperr.CodeForbidden, "forbidden", err)
	case errors.Is(err, apperr.ErrRegistrationDisabled):
		return apperr.New(http.StatusForbidden, apperr.CodeRegistrationDisabled, "registration disabled", err)
	case errors.Is(err, apperr.ErrEmailNotVerified):
		return apperr.New(http.StatusForbidden, apperr.CodeEmailNotVerified, "email not verified", err)
	case errors.Is(err, apperr.ErrLoginMethodDisabled):
		return apperr.New(http.StatusBadRequest, apperr.CodeLoginMethodDisabled, "login method disabled", err)
	case errors.Is(err, apperr.ErrConflict), errors.Is(err, primary.ErrConflict), errors.Is(err, logdb.ErrConflict):
		return apperr.New(http.StatusConflict, apperr.CodeConflict, "conflict", err)
	case errors.Is(err, apperr.ErrNotFound), errors.Is(err, primary.ErrNotFound), errors.Is(err, logdb.ErrNotFound):
		return apperr.New(http.StatusNotFound, apperr.CodeNotFound, "not found", err)
	case errors.Is(err, apperr.ErrSettingsConflict), errors.Is(err, primary.ErrSettingsVersionConflict):
		return apperr.New(http.StatusConflict, apperr.CodeSettingsConflict, "settings conflict", err)
	case errors.Is(err, apperr.ErrPointsInsufficient), errors.Is(err, primary.ErrBalanceBelowZero):
		return apperr.New(http.StatusConflict, apperr.CodePointsInsufficient, "insufficient points", err)
	case errors.Is(err, apperr.ErrIdempotencyConflict), errors.Is(err, primary.ErrIdempotencyReplay):
		return apperr.New(http.StatusConflict, apperr.CodeIdempotencyConflict, "idempotency conflict", err)
	case errors.Is(err, apperr.ErrInvalidPassword), errors.Is(err, auth.ErrPasswordTooShort),
		errors.Is(err, auth.ErrPasswordTooLong), errors.Is(err, auth.ErrPasswordMismatch):
		return apperr.New(http.StatusBadRequest, apperr.CodeInvalidPassword, "invalid password", err)
	case errors.Is(err, apperr.ErrAccountDisabled):
		return apperr.New(http.StatusForbidden, apperr.CodeAccountDisabled, "account disabled", err)
	case errors.Is(err, apperr.ErrVerificationInvalid):
		return apperr.New(http.StatusBadRequest, apperr.CodeVerificationInvalid, "verification token invalid", err)
	case errors.Is(err, apperr.ErrRateLimited), errors.Is(err, ratelimit.ErrStorageUnavailable):
		return apperr.New(http.StatusTooManyRequests, apperr.CodeRateLimited, "rate limited", err)
	case errors.Is(err, apperr.ErrAuditUnavailable):
		return apperr.New(http.StatusServiceUnavailable, apperr.CodeAuditUnavailable, "audit unavailable", err)
	case errors.Is(err, apperr.ErrSmtpUnavailable):
		return apperr.New(http.StatusServiceUnavailable, apperr.CodeSmtpUnavailable, "smtp unavailable", err)
	case errors.Is(err, apperr.ErrDependencyUnavailable), errors.Is(err, auth.ErrUntrustedOrigin):
		return apperr.New(http.StatusServiceUnavailable, apperr.CodeDependencyUnavailable, "dependency unavailable", err)
	default:
		return apperr.New(http.StatusServiceUnavailable, apperr.CodeDependencyUnavailable, "dependency unavailable", err)
	}
}

func unauthorizedCredentials() error {
	return apperr.New(http.StatusUnauthorized, apperr.CodeUnauthorized, "unauthorized", apperr.ErrUnauthorized)
}

func validationError(message string) error {
	if message == "" {
		message = "validation failed"
	}
	return apperr.New(http.StatusBadRequest, apperr.CodeValidation, message, apperr.ErrValidation)
}
