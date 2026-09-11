package middleware

import (
	"context"
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/rio9466/easy-admin/server/internal/domain/apperr"
	platformauth "github.com/rio9466/easy-admin/server/internal/platform/auth"
	"github.com/rio9466/easy-admin/server/internal/transport/http/response"
)

// UserAuthzPrincipal is live business-user authz state loaded from primary
// storage for protected user endpoints.
type UserAuthzPrincipal struct {
	UserID    int64
	Username  string
	Email     string
	Nickname  string
	Status    string
	AuthEpoch int64
}

// UserAuthzLoader loads current user status and auth epoch.
type UserAuthzLoader interface {
	LoadUserAuthz(ctx context.Context, userID int64) (*UserAuthzPrincipal, error)
}

// AuthenticateUser validates the business-user Bearer access JWT (separate
// audience) and the user Redis session, then loads live user state. Redis,
// database, or session failures fail closed; a user token can never pass an
// administrator endpoint and vice versa because the audiences differ and each
// surface uses its own token service and session namespace.
func AuthenticateUser(tokens TokenParser, sessions SessionGetter, authz UserAuthzLoader) gin.HandlerFunc {
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

		userID, err := strconv.ParseInt(claims.Subject, 10, 64)
		if err != nil || userID <= 0 || sess.AdminID != userID {
			response.Error(c, http.StatusUnauthorized, apperr.CodeUnauthorized, "unauthorized")
			c.Abort()
			return
		}

		principal, err := authz.LoadUserAuthz(c.Request.Context(), userID)
		if err != nil {
			WriteAppError(c, err)
			c.Abort()
			return
		}
		if principal == nil || principal.AuthEpoch != sess.AuthEpoch {
			response.Error(c, http.StatusUnauthorized, apperr.CodeSessionInvalid, "session invalid")
			c.Abort()
			return
		}

		SetUserActor(c, userID, claims.SessionID, principal.Username, principal.Email, principal.Nickname)
		c.Next()
	}
}
