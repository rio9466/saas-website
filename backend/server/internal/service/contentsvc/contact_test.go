package contentsvc

import (
	"context"
	"errors"
	"testing"

	"github.com/alicebob/miniredis/v2"
	goredis "github.com/redis/go-redis/v9"
	"github.com/rio9466/easy-admin/server/internal/domain/apperr"
	"github.com/rio9466/easy-admin/server/internal/domain/contact"
	"github.com/rio9466/easy-admin/server/internal/platform/ratelimit"
)

func newTestLimiter(t *testing.T) (*ratelimit.Limiter, *miniredis.Miniredis) {
	t.Helper()
	mr := miniredis.RunT(t)
	rdb := goredis.NewClient(&goredis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = rdb.Close() })
	limiter, err := ratelimit.New(rdb)
	if err != nil {
		t.Fatalf("ratelimit.New: %v", err)
	}
	return limiter, mr
}

func validContactInput() contact.SubmissionInput {
	return contact.SubmissionInput{
		Name:    "Alice",
		Email:   "alice@example.com",
		Message: "Hello",
		Consent: true,
	}
}

func TestSubmitContactHoneypotSkipsStorageAndLimit(t *testing.T) {
	t.Parallel()
	// A service with no repositories/limiter: the honeypot path must return
	// before touching any collaborator.
	svc := &Service{}
	receipt, err := svc.SubmitContact(context.Background(), contact.SubmissionInput{Website: "spam"}, "203.0.113.7", "ua")
	if err != nil {
		t.Fatalf("SubmitContact(honeypot) error = %v", err)
	}
	if receipt == nil || receipt.ID != 0 {
		t.Fatalf("SubmitContact(honeypot) = %+v, want zero receipt", receipt)
	}
}

func TestSubmitContactValidationRunsBeforeRateLimit(t *testing.T) {
	t.Parallel()
	limiter, mr := newTestLimiter(t)
	svc := &Service{limiter: limiter}
	in := validContactInput()
	in.Consent = false
	_, err := svc.SubmitContact(context.Background(), in, "203.0.113.9", "ua")
	var appErr *apperr.AppError
	if !errors.As(err, &appErr) || appErr.Code != apperr.CodeValidation {
		t.Fatalf("SubmitContact(invalid) = %v, want validation error", err)
	}
	if mr.Exists(contactRatePrefix + "203.0.113.9") {
		t.Fatal("invalid submission must not consume the rate-limit budget")
	}
}

func TestSubmitContactFailsClosedWhenRedisUnavailable(t *testing.T) {
	t.Parallel()
	limiter, mr := newTestLimiter(t)
	mr.Close()
	svc := &Service{limiter: limiter}
	_, err := svc.SubmitContact(context.Background(), validContactInput(), "203.0.113.10", "ua")
	var appErr *apperr.AppError
	if !errors.As(err, &appErr) {
		t.Fatalf("SubmitContact(redis down) = %v, want AppError", err)
	}
	if appErr.Code != apperr.CodeRateLimited || appErr.HTTPStatus != 429 {
		t.Fatalf("SubmitContact(redis down) = code %d status %d, want 42901/429", appErr.Code, appErr.HTTPStatus)
	}
}

func TestAllowContactRateBlocksSixthSubmission(t *testing.T) {
	t.Parallel()
	limiter, _ := newTestLimiter(t)
	svc := &Service{limiter: limiter}
	ctx := context.Background()
	const ip = "198.51.100.4"
	for i := 0; i < contactRateLimit; i++ {
		if err := svc.allowContactRate(ctx, ip); err != nil {
			t.Fatalf("allowContactRate #%d = %v, want nil", i+1, err)
		}
	}
	err := svc.allowContactRate(ctx, ip)
	var appErr *apperr.AppError
	if !errors.As(err, &appErr) || appErr.Code != apperr.CodeRateLimited {
		t.Fatalf("sixth allowContactRate = %v, want rate limited", err)
	}
}
