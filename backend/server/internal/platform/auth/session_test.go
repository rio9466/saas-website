package auth

import (
	"context"
	"errors"
	"strconv"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	goredis "github.com/redis/go-redis/v9"
)

func newTestSessionStore(t *testing.T) (*SessionStore, *miniredis.Miniredis) {
	t.Helper()
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatalf("miniredis.Run: %v", err)
	}
	t.Cleanup(mr.Close)

	rdb := goredis.NewClient(&goredis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = rdb.Close() })

	store, err := NewSessionStore(rdb)
	if err != nil {
		t.Fatalf("NewSessionStore: %v", err)
	}
	return store, mr
}

func TestSessionRotateRefreshReplayRevokes(t *testing.T) {
	t.Parallel()

	store, _ := newTestSessionStore(t)
	ctx := context.Background()
	sid := "sid-replay"
	expiresAt := time.Now().UTC().Add(RefreshTokenTTL)
	oldHash := HashRefreshVerifier("old-jti")
	newHash := HashRefreshVerifier("new-jti")

	if err := store.Create(ctx, sid, 7, oldHash, 0, expiresAt); err != nil {
		t.Fatalf("Create: %v", err)
	}

	if err := store.RotateRefresh(ctx, sid, oldHash, newHash); err != nil {
		t.Fatalf("RotateRefresh success: %v", err)
	}

	sess, err := store.Get(ctx, sid)
	if err != nil {
		t.Fatalf("Get after rotate: %v", err)
	}
	if sess.RefreshJTIHash != newHash {
		t.Fatalf("hash = %q, want %q", sess.RefreshJTIHash, newHash)
	}
	if sess.AuthEpoch != 0 {
		t.Fatalf("auth epoch = %d, want 0 preserved through rotation", sess.AuthEpoch)
	}
	if !sess.ExpiresAt.Equal(expiresAt.UTC()) {
		t.Fatalf("absolute expiry = %v, want %v preserved through rotation", sess.ExpiresAt, expiresAt.UTC())
	}

	// Replay old refresh verifier.
	err = store.RotateRefresh(ctx, sid, oldHash, HashRefreshVerifier("attacker"))
	if !errors.Is(err, ErrRefreshReplay) {
		t.Fatalf("replay = %v, want %v", err, ErrRefreshReplay)
	}

	if _, err := store.Get(ctx, sid); !errors.Is(err, ErrSessionInvalid) {
		t.Fatalf("Get after replay = %v, want %v", err, ErrSessionInvalid)
	}
}

func TestSessionRevokeFailsClosed(t *testing.T) {
	t.Parallel()

	store, _ := newTestSessionStore(t)
	ctx := context.Background()
	sid := "sid-revoke"
	expiresAt := time.Now().UTC().Add(time.Hour)
	hash := HashRefreshVerifier("jti")

	if err := store.Create(ctx, sid, 1, hash, 3, expiresAt); err != nil {
		t.Fatalf("Create: %v", err)
	}
	sess, err := store.Get(ctx, sid)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if sess.AuthEpoch != 3 {
		t.Fatalf("auth epoch = %d, want 3", sess.AuthEpoch)
	}
	if err := store.Revoke(ctx, sid); err != nil {
		t.Fatalf("Revoke: %v", err)
	}
	if _, err := store.Get(ctx, sid); !errors.Is(err, ErrSessionInvalid) {
		t.Fatalf("Get revoked = %v, want %v", err, ErrSessionInvalid)
	}
}

func TestSessionRevokeAllForAdmin(t *testing.T) {
	t.Parallel()

	store, _ := newTestSessionStore(t)
	ctx := context.Background()
	expiresAt := time.Now().UTC().Add(time.Hour)

	if err := store.Create(ctx, "sid-a1", 42, HashRefreshVerifier("a1"), 0, expiresAt); err != nil {
		t.Fatalf("Create a1: %v", err)
	}
	if err := store.Create(ctx, "sid-a2", 42, HashRefreshVerifier("a2"), 0, expiresAt); err != nil {
		t.Fatalf("Create a2: %v", err)
	}
	if err := store.Create(ctx, "sid-b1", 7, HashRefreshVerifier("b1"), 0, expiresAt); err != nil {
		t.Fatalf("Create b1: %v", err)
	}

	if err := store.RevokeAllForAdmin(ctx, 42); err != nil {
		t.Fatalf("RevokeAllForAdmin: %v", err)
	}
	if _, err := store.Get(ctx, "sid-a1"); !errors.Is(err, ErrSessionInvalid) {
		t.Fatalf("sid-a1 after revoke-all = %v, want invalid", err)
	}
	if _, err := store.Get(ctx, "sid-a2"); !errors.Is(err, ErrSessionInvalid) {
		t.Fatalf("sid-a2 after revoke-all = %v, want invalid", err)
	}
	if _, err := store.Get(ctx, "sid-b1"); err != nil {
		t.Fatalf("other admin session should remain active: %v", err)
	}
}

func sessionTTL(t *testing.T, store *SessionStore, key string) time.Duration {
	t.Helper()
	ttl, err := store.rdb.TTL(context.Background(), key).Result()
	if err != nil {
		t.Fatalf("TTL(%s): %v", key, err)
	}
	return ttl
}

// TestSessionRotateKeepsAbsoluteExpiryAndNeverExtendsTTL proves rotation keeps
// the login-time absolute expiry unchanged and never grows the Redis TTL, so
// repeated refreshes cannot slide a session forward indefinitely.
func TestSessionRotateKeepsAbsoluteExpiryAndNeverExtendsTTL(t *testing.T) {
	t.Parallel()

	store, _ := newTestSessionStore(t)
	ctx := context.Background()
	sid := "sid-absolute"
	absolute := time.Now().UTC().Add(2 * time.Hour)
	adminID := int64(9)

	if err := store.Create(ctx, sid, adminID, HashRefreshVerifier("v0"), 0, absolute); err != nil {
		t.Fatalf("Create: %v", err)
	}
	sessionKey := SessionKey(sid)
	indexKey := SessionIndexKey(adminID)
	originalTTL := sessionTTL(t, store, sessionKey)

	currentHash := HashRefreshVerifier("v0")
	for i := 1; i <= 5; i++ {
		nextHash := HashRefreshVerifier("v" + strconv.Itoa(i))
		if err := store.RotateRefresh(ctx, sid, currentHash, nextHash); err != nil {
			t.Fatalf("rotate %d: %v", i, err)
		}
		currentHash = nextHash

		sess, err := store.Get(ctx, sid)
		if err != nil {
			t.Fatalf("Get after rotate %d: %v", i, err)
		}
		if !sess.ExpiresAt.Equal(absolute.UTC()) {
			t.Fatalf("rotate %d changed absolute expiry to %v, want %v", i, sess.ExpiresAt, absolute.UTC())
		}
		// The session and index TTLs must never exceed the original lifetime.
		if got := sessionTTL(t, store, sessionKey); got > originalTTL+2*time.Second {
			t.Fatalf("rotate %d extended session TTL to %v (> original %v)", i, got, originalTTL)
		}
		if got := sessionTTL(t, store, indexKey); got > originalTTL+2*time.Second {
			t.Fatalf("rotate %d extended index TTL to %v (> original %v)", i, got, originalTTL)
		}
	}
}

// TestSessionCannotRefreshPastAbsoluteExpiry proves a session whose remaining
// lifetime is gone is dropped instead of being renewed by a rotation.
func TestSessionCannotRefreshPastAbsoluteExpiry(t *testing.T) {
	t.Parallel()

	store, mr := newTestSessionStore(t)
	ctx := context.Background()
	sid := "sid-expired-rotate"
	absolute := time.Now().UTC().Add(90 * time.Minute)
	if err := store.Create(ctx, sid, 11, HashRefreshVerifier("exp-v0"), 0, absolute); err != nil {
		t.Fatalf("Create: %v", err)
	}

	// Advance the Redis clock past the absolute expiry.
	mr.FastForward(2 * time.Hour)

	if _, err := store.Get(ctx, sid); !errors.Is(err, ErrSessionInvalid) {
		t.Fatalf("Get after absolute expiry = %v, want %v", err, ErrSessionInvalid)
	}
	if err := store.RotateRefresh(ctx, sid, HashRefreshVerifier("exp-v0"), HashRefreshVerifier("exp-v1")); !errors.Is(err, ErrSessionInvalid) {
		t.Fatalf("rotate past absolute expiry = %v, want %v", err, ErrSessionInvalid)
	}
	if _, err := store.Get(ctx, sid); !errors.Is(err, ErrSessionInvalid) {
		t.Fatalf("session must be gone after absolute expiry: %v", err)
	}
}
