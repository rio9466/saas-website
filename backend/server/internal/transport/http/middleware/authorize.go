package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/rio9466/easy-admin/server/internal/domain/adminauth"
	"github.com/rio9466/easy-admin/server/internal/domain/apperr"
	"github.com/rio9466/easy-admin/server/internal/transport/http/response"
)

// RequirePermission rejects authenticated callers that lack the permission code.
// Administrators holding the built-in super_admin role bypass the check.
func RequirePermission(code string) gin.HandlerFunc {
	return func(c *gin.Context) {
		if AdminIDFromContext(c) <= 0 {
			response.Error(c, http.StatusUnauthorized, apperr.CodeUnauthorized, "unauthorized")
			c.Abort()
			return
		}
		if hasRole(RolesFromContext(c), adminauth.RoleSuperAdmin) {
			c.Next()
			return
		}
		if hasPermission(PermissionsFromContext(c), code) {
			c.Next()
			return
		}
		response.Error(c, http.StatusForbidden, apperr.CodeForbidden, "forbidden")
		c.Abort()
	}
}

func hasRole(roles []string, want string) bool {
	for _, r := range roles {
		if r == want {
			return true
		}
	}
	return false
}

func hasPermission(perms []string, want string) bool {
	for _, p := range perms {
		if p == want {
			return true
		}
	}
	return false
}
