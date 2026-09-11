package auth

import (
	"errors"
	"net/http"
	"net/url"
	"strings"
)

// ErrUntrustedOrigin is returned when Origin/Referer is missing or not trusted.
var ErrUntrustedOrigin = errors.New("untrusted origin")

// ValidateTrustedOrigin checks login/refresh/logout browser origin against an
// allowlist. Prefer the Origin header; otherwise derive an origin from Referer.
// When trustedOrigins is non-empty, requests without Origin and Referer are rejected.
// Exact string match against configured origins is required.
func ValidateTrustedOrigin(r *http.Request, trustedOrigins []string) error {
	if len(trustedOrigins) == 0 {
		return nil
	}
	if r == nil {
		return ErrUntrustedOrigin
	}

	if origin := strings.TrimSpace(r.Header.Get("Origin")); origin != "" {
		if originAllowed(origin, trustedOrigins) {
			return nil
		}
		return ErrUntrustedOrigin
	}

	referer := strings.TrimSpace(r.Header.Get("Referer"))
	if referer == "" {
		return ErrUntrustedOrigin
	}

	u, err := url.Parse(referer)
	if err != nil || u.Scheme == "" || u.Host == "" {
		return ErrUntrustedOrigin
	}
	refOrigin := u.Scheme + "://" + u.Host
	if originAllowed(refOrigin, trustedOrigins) {
		return nil
	}
	return ErrUntrustedOrigin
}

func originAllowed(origin string, trusted []string) bool {
	for _, allowed := range trusted {
		if origin == allowed {
			return true
		}
	}
	return false
}
