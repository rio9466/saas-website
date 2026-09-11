package response

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/rio9466/easy-admin/server/internal/domain/apperr"
)

// requestIDContextKey must match middleware.RequestID storage key.
const requestIDContextKey = "request_id"

// Envelope is the shared JSON response shape for all HTTP endpoints.
type Envelope struct {
	Code      int         `json:"code"`
	Message   string      `json:"message"`
	Data      interface{} `json:"data"`
	RequestID string      `json:"request_id"`
}

// JSON writes a successful envelope response.
func JSON(c *gin.Context, status int, data interface{}) {
	write(c, status, 0, "success", data)
}

// Error writes an application error envelope with a client-safe message.
func Error(c *gin.Context, status int, code int, message string) {
	write(c, status, code, message, map[string]interface{}{})
}

func write(c *gin.Context, status int, code int, message string, data interface{}) {
	if data == nil {
		data = map[string]interface{}{}
	}

	c.JSON(status, Envelope{
		Code:      code,
		Message:   message,
		Data:      data,
		RequestID: requestIDFromContext(c),
	})
}

// OK is a convenience helper for HTTP 200 success responses.
func OK(c *gin.Context, data interface{}) {
	JSON(c, http.StatusOK, data)
}

// WriteAppError maps an error to the shared JSON envelope with client-safe messages.
func WriteAppError(c *gin.Context, err error) {
	if err == nil {
		return
	}

	var ae *apperr.AppError
	if errors.As(err, &ae) && ae != nil {
		status := ae.HTTPStatus
		if status == 0 {
			status = http.StatusInternalServerError
		}
		Error(c, status, ae.Code, safeMessage(ae))
		return
	}

	switch {
	case errors.Is(err, apperr.ErrUnauthorized), errors.Is(err, apperr.ErrSessionInvalid):
		Error(c, http.StatusUnauthorized, apperr.CodeUnauthorized, "unauthorized")
	case errors.Is(err, apperr.ErrForbidden):
		Error(c, http.StatusForbidden, apperr.CodeForbidden, "forbidden")
	case errors.Is(err, apperr.ErrNotFound):
		Error(c, http.StatusNotFound, apperr.CodeNotFound, "not found")
	case errors.Is(err, apperr.ErrConflict), errors.Is(err, apperr.ErrBootstrapConflict):
		Error(c, http.StatusConflict, apperr.CodeConflict, "conflict")
	case errors.Is(err, apperr.ErrValidation):
		Error(c, http.StatusBadRequest, apperr.CodeValidation, "validation failed")
	case errors.Is(err, apperr.ErrInvalidPassword):
		Error(c, http.StatusBadRequest, apperr.CodeInvalidPassword, "invalid password")
	case errors.Is(err, apperr.ErrAccountDisabled):
		Error(c, http.StatusUnauthorized, apperr.CodeAccountDisabled, "account disabled")
	case errors.Is(err, apperr.ErrRefreshReplay):
		Error(c, http.StatusUnauthorized, apperr.CodeRefreshReplay, "unauthorized")
	case errors.Is(err, apperr.ErrAuditUnavailable):
		Error(c, http.StatusServiceUnavailable, apperr.CodeAuditUnavailable, "audit unavailable")
	case errors.Is(err, apperr.ErrSmtpUnavailable):
		Error(c, http.StatusServiceUnavailable, apperr.CodeSmtpUnavailable, "smtp unavailable")
	case errors.Is(err, apperr.ErrRegistrationDisabled):
		Error(c, http.StatusForbidden, apperr.CodeRegistrationDisabled, "registration disabled")
	case errors.Is(err, apperr.ErrEmailNotVerified):
		Error(c, http.StatusForbidden, apperr.CodeEmailNotVerified, "email not verified")
	case errors.Is(err, apperr.ErrLoginMethodDisabled):
		Error(c, http.StatusBadRequest, apperr.CodeLoginMethodDisabled, "login method disabled")
	case errors.Is(err, apperr.ErrSettingsConflict):
		Error(c, http.StatusConflict, apperr.CodeSettingsConflict, "settings conflict")
	case errors.Is(err, apperr.ErrPointsInsufficient):
		Error(c, http.StatusConflict, apperr.CodePointsInsufficient, "insufficient points")
	case errors.Is(err, apperr.ErrIdempotencyConflict):
		Error(c, http.StatusConflict, apperr.CodeIdempotencyConflict, "idempotency conflict")
	case errors.Is(err, apperr.ErrVerificationInvalid):
		Error(c, http.StatusBadRequest, apperr.CodeVerificationInvalid, "verification token invalid")
	case errors.Is(err, apperr.ErrRateLimited):
		Error(c, http.StatusTooManyRequests, apperr.CodeRateLimited, "rate limited")
	case errors.Is(err, apperr.ErrDependencyUnavailable):
		Error(c, http.StatusServiceUnavailable, apperr.CodeDependencyUnavailable, "dependency unavailable")
	default:
		Error(c, http.StatusInternalServerError, apperr.CodeDependencyUnavailable, "dependency unavailable")
	}
}

func requestIDFromContext(c *gin.Context) string {
	if c == nil {
		return ""
	}
	if v, ok := c.Get(requestIDContextKey); ok {
		if id, ok := v.(string); ok {
			return id
		}
	}
	return ""
}

func safeMessage(ae *apperr.AppError) string {
	if ae == nil {
		return "dependency unavailable"
	}
	if ae.Message != "" {
		return ae.Message
	}
	switch ae.Code {
	case apperr.CodeUnauthorized, apperr.CodeSessionInvalid, apperr.CodeRefreshReplay:
		return "unauthorized"
	case apperr.CodeAccountDisabled:
		return "account disabled"
	case apperr.CodeForbidden, apperr.CodeSelfProtection, apperr.CodeLastSuperAdmin:
		return "forbidden"
	case apperr.CodeConflict, apperr.CodeBootstrapConflict:
		return "conflict"
	case apperr.CodeNotFound:
		return "not found"
	case apperr.CodeValidation:
		return "validation failed"
	case apperr.CodeInvalidPassword:
		return "invalid password"
	case apperr.CodeAuditUnavailable:
		return "audit unavailable"
	case apperr.CodeSmtpUnavailable:
		return "smtp unavailable"
	case apperr.CodeRegistrationDisabled:
		return "registration disabled"
	case apperr.CodeEmailNotVerified:
		return "email not verified"
	case apperr.CodeLoginMethodDisabled:
		return "login method disabled"
	case apperr.CodeSettingsConflict, apperr.CodePointsInsufficient, apperr.CodeIdempotencyConflict:
		return "conflict"
	case apperr.CodeVerificationInvalid:
		return "verification token invalid"
	case apperr.CodeRateLimited:
		return "rate limited"
	case apperr.CodeDependencyUnavailable:
		return "dependency unavailable"
	default:
		return "dependency unavailable"
	}
}
