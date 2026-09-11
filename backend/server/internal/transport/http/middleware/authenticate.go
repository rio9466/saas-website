package middleware

import (
	"context"
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/rio9466/easy-admin/server/internal/domain/apperr"
	platformauth "github.com/rio9466/easy-admin/server/internal/platform/auth"
	"github.com/rio9466/easy-admin/server/internal/transport/http/response"
)

// TokenParser validates access JWTs.
type TokenParser interface {
	ParseAndValidate(tokenString string, expectedType platformauth.TokenType) (*platformauth.Claims, error)
}

// SessionGetter loads Redis sessions and fails closed on errors.
type SessionGetter interface {
	Get(ctx context.Context, sid string) (*platformauth.Session, error)
}

// AuthzPrincipal is live authorization state loaded from primary storage.
type AuthzPrincipal struct {
	AdminID         int64
	Username        string
	DisplayName     string
	Enabled         bool
	AuthEpoch       int64
	RoleCodes       []string
	PermissionCodes []string
}

// AuthzLoader loads current administrator enablement, roles, and permissions.
type AuthzLoader interface {
	LoadAuthz(ctx context.Context, adminID int64) (*AuthzPrincipal, error)
}

// Authenticate validates the Bearer access JWT and Redis session, then loads
// live RBAC state into the Gin context. Redis and lookup failures fail closed.
func Authenticate(tokens TokenParser, sessions SessionGetter, authz AuthzLoader) gin.HandlerFunc {
	return func(c *gin.Context) {
		if tokens == nil || sessions == nil || authz == nil {
			response.Error(c, http.StatusUnauthorized, apperr.CodeUnauthorized, "unauthorized")
			c.Abort()
			return
		}

		raw := bearerToken(c.GetHeader("Authorization"))
		if raw == "" {
			response.Error(c, http.StatusUnauthorized, apperr.CodeUnauthorized, "unauthorized")
			c.Abort()
			return
		}

		claims, err := tokens.ParseAndValidate(raw, platformauth.TokenTypeAccess)
		if err != nil {
			response.Error(c, http.StatusUnauthorized, apperr.CodeUnauthorized, "unauthorized")
			c.Abort()
			return
		}

		sess, err := sessions.Get(c.Request.Context(), claims.SessionID)
		if err != nil {
			code := apperr.CodeSessionInvalid
			msg := "session invalid"
			if !errors.Is(err, platformauth.ErrSessionInvalid) {
				code = apperr.CodeUnauthorized
				msg = "unauthorized"
			}
			response.Error(c, http.StatusUnauthorized, code, msg)
			c.Abort()
			return
		}

		adminID, err := strconv.ParseInt(claims.Subject, 10, 64)
		if err != nil || adminID <= 0 || sess.AdminID != adminID {
			response.Error(c, http.StatusUnauthorized, apperr.CodeUnauthorized, "unauthorized")
			c.Abort()
			return
		}

		principal, err := authz.LoadAuthz(c.Request.Context(), adminID)
		if err != nil {
			WriteAppError(c, err)
			c.Abort()
			return
		}
		if principal == nil || !principal.Enabled {
			response.Error(c, http.StatusUnauthorized, apperr.CodeAccountDisabled, "account disabled")
			c.Abort()
			return
		}
		if principal.AdminID != 0 && principal.AdminID != adminID {
			response.Error(c, http.StatusUnauthorized, apperr.CodeUnauthorized, "unauthorized")
			c.Abort()
			return
		}
		// Fail closed when a password change, password reset, or account
		// enable/disable bumped the administrator auth epoch after this
		// session was issued. This is the primary-database guarantee that
		// credentials/account transitions invalidate every old session even
		// when a Redis revocation scan missed a concurrently created one.
		if principal.AuthEpoch != sess.AuthEpoch {
			response.Error(c, http.StatusUnauthorized, apperr.CodeSessionInvalid, "session invalid")
			c.Abort()
			return
		}

		SetActor(c, adminID, claims.SessionID, principal.Username, principal.DisplayName, principal.RoleCodes, principal.PermissionCodes)
		c.Next()
	}
}

func bearerToken(header string) string {
	header = strings.TrimSpace(header)
	if header == "" {
		return ""
	}
	const prefix = "Bearer "
	if len(header) < len(prefix) || !strings.EqualFold(header[:len(prefix)], prefix) {
		return ""
	}
	return strings.TrimSpace(header[len(prefix):])
}
