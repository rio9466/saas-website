package middleware

import (
	"crypto/rand"
	"encoding/hex"

	"github.com/gin-gonic/gin"
)

const (
	// HeaderRequestID is the inbound/outbound request ID header.
	HeaderRequestID = "X-Request-ID"

	// MaxRequestIDLength is the maximum accepted inbound request ID length.
	MaxRequestIDLength = 128

	requestIDContextKey = "request_id"
)

// RequestID ensures every request has a non-empty, format-compliant request ID
// that is echoed in the response header and available to handlers via context.
//
// Valid inbound IDs are 1-128 characters of ASCII alphanumeric plus `.`, `_`,
// `:`, or `-`. Empty, oversized, whitespace-containing, Unicode, or otherwise
// invalid values are replaced with a newly generated compliant ID.
func RequestID() gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.GetHeader(HeaderRequestID)
		if !ValidRequestID(id) {
			id = newRequestID()
		}

		c.Set(requestIDContextKey, id)
		c.Writer.Header().Set(HeaderRequestID, id)
		c.Next()
	}
}

// ValidRequestID reports whether id meets the request-ID contract.
func ValidRequestID(id string) bool {
	if len(id) < 1 || len(id) > MaxRequestIDLength {
		return false
	}
	for i := 0; i < len(id); i++ {
		c := id[i]
		switch {
		case c >= 'a' && c <= 'z':
		case c >= 'A' && c <= 'Z':
		case c >= '0' && c <= '9':
		case c == '.' || c == '_' || c == ':' || c == '-':
		default:
			return false
		}
	}
	return true
}

// RequestIDFromContext returns the request ID stored by RequestID middleware.
func RequestIDFromContext(c *gin.Context) string {
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

func newRequestID() string {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "00000000000000000000000000000000"
	}
	id := hex.EncodeToString(b[:])
	if !ValidRequestID(id) {
		return "00000000000000000000000000000000"
	}
	return id
}
