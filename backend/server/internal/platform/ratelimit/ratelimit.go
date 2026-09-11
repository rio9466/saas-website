// Package ratelimit provides a small fail-closed fixed-window rate limiter
// backed by Redis. Used for registration, login, resend-verification, and
// verification attempts. A Redis outage rejects requests instead of allowing
// unbounded traffic.
package ratelimit

import (
	"context"
	"errors"
	"fmt"
	"time"

	goredis "github.com/redis/go-redis/v9"
)

var (
	// ErrStorageUnavailable reports that rate-limit state could not be
	// consulted; callers must fail closed.
	ErrStorageUnavailable = errors.New("rate limit storage unavailable")
)

// Limiter is a fixed-window counter limiter.
type Limiter struct {
	rdb *goredis.Client
}

// New wraps a go-redis client. rdb must be non-nil.
func New(rdb *goredis.Client) (*Limiter, error) {
	if rdb == nil {
		return nil, errors.New("redis client is required")
	}
	return &Limiter{rdb: rdb}, nil
}

// Allow reports whether the key may proceed. limit is the maximum number of
// events in one window. On Redis failure it returns ErrStorageUnavailable so
// protected endpoints can fail closed instead of skipping the check.
func (l *Limiter) Allow(ctx context.Context, key string, limit int, window time.Duration) (bool, error) {
	if l == nil || l.rdb == nil {
		return false, ErrStorageUnavailable
	}
	if key == "" || limit <= 0 || window <= 0 {
		return false, ErrStorageUnavailable
	}

	script := goredis.NewScript(`
local current = redis.call('INCR', KEYS[1])
if current == 1 then
  redis.call('PEXPIRE', KEYS[1], ARGV[1])
end
if current > tonumber(ARGV[2]) then
  return 0
end
return 1
`)
	res, err := script.Run(ctx, l.rdb, []string{key}, int64(window/time.Millisecond), limit).Int64()
	if err != nil {
		return false, fmt.Errorf("%w: %v", ErrStorageUnavailable, err)
	}
	return res == 1, nil
}
