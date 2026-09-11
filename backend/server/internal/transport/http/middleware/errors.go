package middleware

import (
	"github.com/gin-gonic/gin"
	"github.com/rio9466/easy-admin/server/internal/transport/http/response"
)

// WriteAppError maps an error to the shared JSON envelope.
// Kept as a thin alias so existing call sites remain stable.
func WriteAppError(c *gin.Context, err error) {
	response.WriteAppError(c, err)
}
