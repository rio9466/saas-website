package usersvc

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"strings"
	"time"

	"github.com/rio9466/easy-admin/server/internal/domain/apperr"
)

// Rate-limit windows for public business-user endpoints. Limits are per fixed
// window, keyed by IP and (for login) hashed identifier, so abusive bursts are
// contained without revealing whether an account exists. Redis failures fail
// closed through ratelimit.ErrStorageUnavailable.
const (
	registerWindow = time.Hour
	registerLimit  = 5
	loginWindow    = 15 * time.Minute
	loginLimit     = 10
	resendWindow   = time.Hour
	resendLimit    = 3
	verifyWindow   = time.Hour
	verifyLimit    = 10
	rateKeyPrefix  = "easy-admin:rl:"
)

func rateKey(name string) string {
	return rateKeyPrefix + name
}

func rateKeyRegister(ip string) string  { return rateKey("register:" + ip) }
func rateKeyLoginByIP(ip string) string { return rateKey("login-ip:" + ip) }
func rateKeyLoginByID(identifier string) string {
	sum := sha256.Sum256([]byte(strings.ToLower(strings.TrimSpace(identifier))))
	return rateKey("login-id:" + hex.EncodeToString(sum[:]))
}
func rateKeyResend(ip string) string     { return rateKey("resend:" + ip) }
func rateKeyVerifyByIP(ip string) string { return rateKey("verify:" + ip) }

// allowRate wraps the limiter and maps storage failures to a fail-closed
// "rate limited" AppError so protected endpoints never bypass throttling.
func (s *Service) allowRate(ctx context.Context, key string, limit int, window time.Duration) error {
	ok, err := s.limiter.Allow(ctx, key, limit, window)
	if err != nil {
		// Redis outage: fail closed rather than allow unbounded traffic.
		return apperr.New(429, apperr.CodeRateLimited, "rate limited", apperr.ErrRateLimited)
	}
	if !ok {
		return apperr.New(429, apperr.CodeRateLimited, "rate limited", apperr.ErrRateLimited)
	}
	return nil
}
