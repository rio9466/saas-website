package auth

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	goredis "github.com/redis/go-redis/v9"
)

func userRedisTest(t *testing.T) (*SessionStore, *SessionStore, *miniredis.Miniredis) {
	t.Helper()
	mr := miniredis.RunT(t)
	rdb := goredis.NewClient(&goredis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = rdb.Close() })
	user, err := NewUserSessionStore(rdb)
	if err != nil {
		t.Fatal(err)
	}
	admin, err := NewSessionStore(rdb)
	if err != nil {
		t.Fatal(err)
	}
	return user, admin, mr
}

func TestUserSessionNamespaceSeparateFromAdmin(t *testing.T) {
	userStore, adminStore, _ := userRedisTest(t)
	ctx := context.Background()
	exp := time.Now().UTC().Add(RefreshTokenTTL)

	if err := userStore.Create(ctx, "u-sid-1", 42, "hash-a", 1, exp); err != nil {
		t.Fatalf("user store create: %v", err)
	}
	if err := adminStore.Create(ctx, "a-sid-1", 7, "hash-a", 1, exp); err != nil {
		t.Fatalf("admin store create: %v", err)
	}

	// Same sid strings in different namespaces must not collide.
	sess, err := userStore.Get(ctx, "u-sid-1")
	if err != nil {
		t.Fatalf("user get: %v", err)
	}
	if sess.AdminID != 42 {
		t.Fatalf("user session subject = %d, want 42", sess.AdminID)
	}

	// The admin namespace must not see the user session, and vice versa.
	if _, err := adminStore.Get(ctx, "u-sid-1"); err == nil {
		t.Fatal("admin store must not see a user session")
	}
	if _, err := userStore.Get(ctx, "a-sid-1"); err == nil {
		t.Fatal("user store must not see an admin session")
	}

	if !strings.HasPrefix(UserSessionKey("u-sid-1"), "easy-admin:user-session:") {
		t.Fatalf("user key prefix mismatch: %s", UserSessionKey("u-sid-1"))
	}
	if !strings.HasPrefix(SessionKey("a-sid-1"), "easy-admin:admin-session:") {
		t.Fatalf("admin key prefix mismatch: %s", SessionKey("a-sid-1"))
	}
}

func TestUserSessionRotateReplayAndRevoke(t *testing.T) {
	store, _, _ := userRedisTest(t)
	ctx := context.Background()
	exp := time.Now().UTC().Add(RefreshTokenTTL)

	if err := store.Create(ctx, "u-rot-1", 42, HashRefreshVerifier("old-jti"), 1, exp); err != nil {
		t.Fatal(err)
	}
	// Rotate old -> new succeeds.
	if err := store.RotateRefresh(ctx, "u-rot-1", HashRefreshVerifier("old-jti"), HashRefreshVerifier("new-jti")); err != nil {
		t.Fatalf("rotate: %v", err)
	}
	// Replay of the old verifier must revoke and report replay.
	if err := store.RotateRefresh(ctx, "u-rot-1", HashRefreshVerifier("old-jti"), HashRefreshVerifier("x")); err != ErrRefreshReplay {
		t.Fatalf("replay err = %v, want ErrRefreshReplay", err)
	}
	// After replay the session is revoked.
	if _, err := store.Get(ctx, "u-rot-1"); err == nil {
		t.Fatal("session must be revoked after replay")
	}

	// RevokeAllForAdmin removes every session of a subject.
	if err := store.Create(ctx, "u-x1", 42, "h1", 1, exp); err != nil {
		t.Fatal(err)
	}
	if err := store.Create(ctx, "u-x2", 42, "h2", 1, exp); err != nil {
		t.Fatal(err)
	}
	if err := store.Create(ctx, "u-y1", 99, "h3", 1, exp); err != nil {
		t.Fatal(err)
	}
	if err := store.RevokeAllForAdmin(ctx, 42); err != nil {
		t.Fatal(err)
	}
	if _, err := store.Get(ctx, "u-x1"); err == nil {
		t.Fatal("u-x1 must be revoked")
	}
	if _, err := store.Get(ctx, "u-x2"); err == nil {
		t.Fatal("u-x2 must be revoked")
	}
	if _, err := store.Get(ctx, "u-y1"); err != nil {
		t.Fatalf("u-y1 must survive: %v", err)
	}
}

func TestUserSessionRotationNeverExtendsExpiry(t *testing.T) {
	store, _, mr := userRedisTest(t)
	ctx := context.Background()
	exp := time.Now().UTC().Add(RefreshTokenTTL)

	if err := store.Create(ctx, "u-abs-1", 42, HashRefreshVerifier("a"), 1, exp); err != nil {
		t.Fatal(err)
	}
	// Fast-forward past expiry: rotation must fail closed because the session's
	// remaining lifetime is gone and renewal would extend the absolute expiry.
	mr.FastForward(RefreshTokenTTL + 2*time.Second)
	if err := store.RotateRefresh(ctx, "u-abs-1", HashRefreshVerifier("a"), HashRefreshVerifier("b")); err != ErrSessionInvalid {
		t.Fatalf("rotating an expired session must fail closed, got %v", err)
	}
}

func TestUserSessionMissingFailsClosed(t *testing.T) {
	store, _, _ := userRedisTest(t)
	ctx := context.Background()
	if _, err := store.Get(ctx, "does-not-exist"); err == nil {
		t.Fatal("missing session must fail closed")
	}
}
