package apperr

import "errors"

// Application error codes registered for the admin auth/RBAC/audit and
// business-user surfaces.
const (
	CodeValidation            = 10001
	CodeUnauthorized          = 20001
	CodeSessionInvalid        = 20002
	CodeRefreshReplay         = 20003
	CodeForbidden             = 30001
	CodeConflict              = 40001
	CodeNotFound              = 40002
	CodeLastSuperAdmin        = 40003
	CodeSelfProtection        = 40004
	CodeInvalidPassword       = 40005
	CodeAccountDisabled       = 40006
	CodeBootstrapConflict     = 40007
	CodeRegistrationDisabled  = 40010
	CodeEmailNotVerified      = 40011
	CodeLoginMethodDisabled   = 40012
	CodeSettingsConflict      = 40013
	CodePointsInsufficient    = 40014
	CodeIdempotencyConflict   = 40015
	CodeVerificationInvalid   = 40016
	CodeDependencyUnavailable = 50001
	CodeAuditUnavailable      = 50002
	CodeSmtpUnavailable       = 50003
)

// CodeRateLimited is a dedicated HTTP 429 application code.
const CodeRateLimited = 42901

var (
	ErrValidation            = errors.New("validation failed")
	ErrUnauthorized          = errors.New("unauthorized")
	ErrSessionInvalid        = errors.New("session invalid")
	ErrRefreshReplay         = errors.New("refresh token replay")
	ErrForbidden             = errors.New("forbidden")
	ErrConflict              = errors.New("conflict")
	ErrNotFound              = errors.New("not found")
	ErrLastSuperAdmin        = errors.New("last super admin protected")
	ErrSelfProtection        = errors.New("self protection")
	ErrInvalidPassword       = errors.New("invalid password")
	ErrAccountDisabled       = errors.New("account disabled")
	ErrBootstrapConflict     = errors.New("bootstrap already completed")
	ErrDependencyUnavailable = errors.New("dependency unavailable")
	ErrAuditUnavailable      = errors.New("audit unavailable")
	ErrRegistrationDisabled  = errors.New("registration disabled")
	ErrEmailNotVerified      = errors.New("email not verified")
	ErrLoginMethodDisabled   = errors.New("login method disabled")
	ErrSettingsConflict      = errors.New("settings conflict")
	ErrPointsInsufficient    = errors.New("insufficient points")
	ErrIdempotencyConflict   = errors.New("idempotency conflict")
	ErrVerificationInvalid   = errors.New("verification token invalid")
	ErrRateLimited           = errors.New("rate limited")
	ErrSmtpUnavailable       = errors.New("smtp unavailable")
)

// AppError carries an HTTP status, application code, and client-safe message.
type AppError struct {
	HTTPStatus int
	Code       int
	Message    string
	Err        error
}

func (e *AppError) Error() string {
	if e == nil {
		return ""
	}
	if e.Err != nil {
		return e.Err.Error()
	}
	return e.Message
}

func (e *AppError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Err
}

func New(httpStatus, code int, message string, err error) *AppError {
	return &AppError{HTTPStatus: httpStatus, Code: code, Message: message, Err: err}
}
