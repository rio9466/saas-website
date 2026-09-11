package adminauth

import (
	"errors"
	"testing"

	platformauth "github.com/rio9466/easy-admin/server/internal/platform/auth"
)

// stubPasswordCrypto lets the constructor's dummy-hash derivation be exercised
// without a database or wall-clock bcrypt work.
type stubPasswordCrypto struct {
	hashOut string
	hashErr error
}

func (s *stubPasswordCrypto) Hash(string) (string, error) {
	return s.hashOut, s.hashErr
}

func (s *stubPasswordCrypto) Compare(string, string) error {
	return platformauth.ErrPasswordMismatch
}

// TestNewRejectsDummyHashError proves a dummy-hash derivation error fails
// construction closed: no Service is returned and the sanitized sentinel is
// exposed instead of any raw cause.
func TestNewRejectsDummyHashError(t *testing.T) {
	hasher := &stubPasswordCrypto{hashErr: errors.New("raw hasher cause that must not leak")}
	svc, err := New(nil, nil, nil, nil, hasher, nil, AuthOptions{})
	if err == nil {
		t.Fatal("New succeeded despite dummy hash error; unknown-user logins could skip the comparison")
	}
	if !errors.Is(err, ErrDummyCredentialUnavailable) {
		t.Fatalf("err = %v, want ErrDummyCredentialUnavailable", err)
	}
	if svc != nil {
		t.Fatalf("New returned a usable Service (%p) on dummy hash error; must fail closed", svc)
	}
}

// TestNewRejectsEmptyDummyHash proves an empty hash returned without an error
// is also rejected: compareDummyPassword must never silently skip because the
// startup hash came back empty.
func TestNewRejectsEmptyDummyHash(t *testing.T) {
	hasher := &stubPasswordCrypto{hashOut: ""}
	svc, err := New(nil, nil, nil, nil, hasher, nil, AuthOptions{})
	if err == nil {
		t.Fatal("New succeeded despite empty dummy hash; unknown-user logins could skip the comparison")
	}
	if !errors.Is(err, ErrDummyCredentialUnavailable) {
		t.Fatalf("err = %v, want ErrDummyCredentialUnavailable", err)
	}
	if svc != nil {
		t.Fatalf("New returned a usable Service (%p) on empty dummy hash; must fail closed", svc)
	}
}

// TestNewRejectsNilPasswordHasher proves a nil hasher cannot produce the
// startup dummy hash and therefore cannot construct a Service.
func TestNewRejectsNilPasswordHasher(t *testing.T) {
	svc, err := New(nil, nil, nil, nil, nil, nil, AuthOptions{})
	if err == nil {
		t.Fatal("New succeeded with a nil password hasher")
	}
	if svc != nil {
		t.Fatalf("New returned a usable Service (%p) with a nil hasher", svc)
	}
}

// TestNewSucceedsStoresStartupDummyHash locks the healthy path: a successful
// configured-cost hash is stored once and a Service is returned.
func TestNewSucceedsStoresStartupDummyHash(t *testing.T) {
	hasher := &stubPasswordCrypto{hashOut: "startup-dummy-bcrypt-hash"}
	svc, err := New(nil, nil, nil, nil, hasher, nil, AuthOptions{})
	if err != nil {
		t.Fatalf("New failed on a healthy dummy hash: %v", err)
	}
	if svc == nil {
		t.Fatal("New returned a nil Service")
	}
	if svc.dummyHash != "startup-dummy-bcrypt-hash" {
		t.Fatalf("dummyHash = %q, want the startup hash", svc.dummyHash)
	}
}
