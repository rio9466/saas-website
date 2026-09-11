package middleware

import (
	"strconv"

	"github.com/gin-gonic/gin"
)

const (
	contextAdminID      = "admin_id"
	contextAdminRoles   = "admin_roles"
	contextAdminPerms   = "admin_permissions"
	contextAdminSession = "admin_session_id"
	contextAdminUser    = "admin_username"
	contextAdminDisplay = "admin_display_name"
)

// SetActor stores authenticated administrator identity and authorization on the request.
func SetActor(c *gin.Context, adminID int64, sessionID, username, displayName string, roles, permissions []string) {
	if c == nil {
		return
	}
	c.Set(contextAdminID, adminID)
	c.Set(contextAdminSession, sessionID)
	c.Set(contextAdminUser, username)
	c.Set(contextAdminDisplay, displayName)
	if roles == nil {
		roles = []string{}
	}
	if permissions == nil {
		permissions = []string{}
	}
	c.Set(contextAdminRoles, roles)
	c.Set(contextAdminPerms, permissions)
}

// AdminIDFromContext returns the authenticated administrator ID, or 0.
func AdminIDFromContext(c *gin.Context) int64 {
	if c == nil {
		return 0
	}
	if v, ok := c.Get(contextAdminID); ok {
		switch id := v.(type) {
		case int64:
			return id
		case int:
			return int64(id)
		case string:
			n, err := strconv.ParseInt(id, 10, 64)
			if err == nil {
				return n
			}
		}
	}
	return 0
}

// SessionIDFromContext returns the Redis session ID for the current request.
func SessionIDFromContext(c *gin.Context) string {
	return stringFromContext(c, contextAdminSession)
}

// UsernameFromContext returns the authenticated administrator username.
func UsernameFromContext(c *gin.Context) string {
	return stringFromContext(c, contextAdminUser)
}

// DisplayNameFromContext returns the authenticated administrator display name.
func DisplayNameFromContext(c *gin.Context) string {
	return stringFromContext(c, contextAdminDisplay)
}

// RolesFromContext returns role codes loaded for the authenticated administrator.
func RolesFromContext(c *gin.Context) []string {
	return stringSliceFromContext(c, contextAdminRoles)
}

// PermissionsFromContext returns effective permission codes for the request.
func PermissionsFromContext(c *gin.Context) []string {
	return stringSliceFromContext(c, contextAdminPerms)
}

func stringFromContext(c *gin.Context, key string) string {
	if c == nil {
		return ""
	}
	if v, ok := c.Get(key); ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

func stringSliceFromContext(c *gin.Context, key string) []string {
	if c == nil {
		return nil
	}
	if v, ok := c.Get(key); ok {
		if s, ok := v.([]string); ok {
			out := make([]string, len(s))
			copy(out, s)
			return out
		}
	}
	return nil
}
