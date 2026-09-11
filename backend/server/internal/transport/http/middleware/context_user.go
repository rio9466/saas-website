package middleware

import (
	"strconv"

	"github.com/gin-gonic/gin"
)

const (
	contextUserID       = "user_id"
	contextUserSession  = "user_session_id"
	contextUserUsername = "user_username"
	contextUserEmail    = "user_email"
	contextUserNickname = "user_nickname"
)

// SetUserActor stores the authenticated business-user identity on the request.
func SetUserActor(c *gin.Context, userID int64, sessionID, username, email, nickname string) {
	if c == nil {
		return
	}
	c.Set(contextUserID, userID)
	c.Set(contextUserSession, sessionID)
	c.Set(contextUserUsername, username)
	c.Set(contextUserEmail, email)
	c.Set(contextUserNickname, nickname)
}

// UserIDFromContext returns the authenticated business-user ID, or 0.
func UserIDFromContext(c *gin.Context) int64 {
	if c == nil {
		return 0
	}
	if v, ok := c.Get(contextUserID); ok {
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

// UserSessionIDFromContext returns the user Redis session id.
func UserSessionIDFromContext(c *gin.Context) string {
	return stringFromContext(c, contextUserSession)
}

// UserUsernameFromContext returns the business-user username.
func UserUsernameFromContext(c *gin.Context) string {
	return stringFromContext(c, contextUserUsername)
}

// UserEmailFromContext returns the business-user email.
func UserEmailFromContext(c *gin.Context) string {
	return stringFromContext(c, contextUserEmail)
}

// UserNicknameFromContext returns the business-user nickname.
func UserNicknameFromContext(c *gin.Context) string {
	return stringFromContext(c, contextUserNickname)
}
