package ratelimit

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	goredis "github.com/redis/go-redis/v9"
)

func newTestLimiter(t *testing.T) (*Limiter, *miniredis.Miniredis) {
	t.Helper()
	mr := miniredis.RunT(t)
	rdb := goredis.NewClient(&goredis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = rdb.Close() })
	l, err := New(rdb)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	return l, mr
}

func TestLimiterAllowsUpToLimitThenBlocks(t *testing.T) {
	l, _ := newTestLimiter(t)
	ctx := context.Background()
	key := "easy-admin:rl:test:ip"
	for i := 0; i < 5; i++ {
		ok, err := l.Allow(ctx, key, 5, time.Minute)
		if err != nil || !ok {
			t.Fatalf("allow %d: ok=%v err=%v", i, ok, err)
		}
	}
	ok, err := l.Allow(ctx, key, 5, time.Minute)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if ok {
		t.Fatal("sixth request must be blocked")
	}
}

func TestLimiterWindowExpires(t *testing.T) {
	l, mr := newTestLimiter(t)
	ctx := context.Background()
	key := "easy-admin:rl:test:expire"
	if _, err := l.Allow(ctx, key, 1, time.Minute); err != nil {
		t.Fatal(err)
	}
	mr.FastForward(61 * time.Second)
	ok, err := l.Allow(ctx, key, 1, time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	if !ok {
		t.Fatal("window must reset after expiry")
	}
}

func TestLimiterFailClosedWithoutRedis(t *testing.T) {
	l, err := New(nil)
	if err == nil {
		t.Fatal("nil client must fail construction")
	}
	_ = l

	// A limiter whose Redis is unreachable must fail closed with an error
	// instead of silently allowing traffic.
	mr := miniredis.RunT(t)
	rdb := goredis.NewClient(&goredis.Options{Addr: mr.Addr()})
	limiter, err := New(rdb)
	if err != nil {
		t.Fatal(err)
	}
	mr.Close()
	ok, err := limiter.Allow(context.Background(), "k", 5, time.Minute)
	if err == nil {
		t.Fatal("unreachable redis must return an error (fail closed)")
	}
	if ok {
		t.Fatal("unreachable redis must not allow")
	}
	if !errors.Is(err, ErrStorageUnavailable) {
		t.Fatalf("err = %v, want ErrStorageUnavailable", err)
	}
	_ = rdb.Close()
}
