package auth

import (
	"net/http"
	"testing"
)

func TestValidateTrustedOrigin(t *testing.T) {
	t.Parallel()

	trusted := []string{"https://admin.example.com", "http://127.0.0.1:5173"}

	t.Run("empty allowlist", func(t *testing.T) {
		t.Parallel()
		req, _ := http.NewRequest(http.MethodPost, "http://localhost/login", nil)
		if err := ValidateTrustedOrigin(req, nil); err != nil {
			t.Fatalf("empty allowlist: %v", err)
		}
	})

	t.Run("origin match", func(t *testing.T) {
		t.Parallel()
		req, _ := http.NewRequest(http.MethodPost, "http://localhost/login", nil)
		req.Header.Set("Origin", "https://admin.example.com")
		if err := ValidateTrustedOrigin(req, trusted); err != nil {
			t.Fatalf("origin match: %v", err)
		}
	})

	t.Run("referer fallback", func(t *testing.T) {
		t.Parallel()
		req, _ := http.NewRequest(http.MethodPost, "http://localhost/login", nil)
		req.Header.Set("Referer", "http://127.0.0.1:5173/login?x=1")
		if err := ValidateTrustedOrigin(req, trusted); err != nil {
			t.Fatalf("referer fallback: %v", err)
		}
	})

	t.Run("missing both", func(t *testing.T) {
		t.Parallel()
		req, _ := http.NewRequest(http.MethodPost, "http://localhost/login", nil)
		if err := ValidateTrustedOrigin(req, trusted); err != ErrUntrustedOrigin {
			t.Fatalf("missing both = %v, want %v", err, ErrUntrustedOrigin)
		}
	})

	t.Run("origin wins over referer", func(t *testing.T) {
		t.Parallel()
		req, _ := http.NewRequest(http.MethodPost, "http://localhost/login", nil)
		req.Header.Set("Origin", "https://evil.example")
		req.Header.Set("Referer", "https://admin.example.com/")
		if err := ValidateTrustedOrigin(req, trusted); err != ErrUntrustedOrigin {
			t.Fatalf("bad origin with good referer = %v, want %v", err, ErrUntrustedOrigin)
		}
	})
}
