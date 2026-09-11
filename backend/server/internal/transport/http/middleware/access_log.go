package middleware

import (
	"log/slog"
	"time"

	"github.com/gin-gonic/gin"
)

// AccessLog writes a structured JSON access line after each request.
// It never logs bodies, query strings, cookies, Authorization, or tokens.
func AccessLog(logger *slog.Logger) gin.HandlerFunc {
	if logger == nil {
		logger = slog.Default()
	}
	return func(c *gin.Context) {
		start := time.Now()
		c.Next()

		route := c.FullPath()
		if route == "" {
			route = c.Request.URL.Path
		}

		attrs := []any{
			"request_id", RequestIDFromContext(c),
			"method", c.Request.Method,
			"route", route,
			"status", c.Writer.Status(),
			"latency_ms", time.Since(start).Milliseconds(),
		}
		if adminID := AdminIDFromContext(c); adminID > 0 {
			attrs = append(attrs, "actor_id", adminID)
		}
		logger.Info("http_access", attrs...)
	}
}
