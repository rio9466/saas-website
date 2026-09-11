package auth

import (
	"strings"
	"testing"
)

func TestValidatePasswordLength(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		password string
		wantErr  error
	}{
		{name: "7 bytes too short", password: strings.Repeat("a", 7), wantErr: ErrPasswordTooShort},
		{name: "8 bytes min ok", password: strings.Repeat("a", 8), wantErr: nil},
		{name: "12 bytes ok", password: strings.Repeat("a", 12), wantErr: nil},
		{name: "72 bytes max ok", password: strings.Repeat("a", 72), wantErr: nil},
		{name: "73 bytes too long", password: strings.Repeat("a", 73), wantErr: ErrPasswordTooLong},
		// 你 is 3 UTF-8 bytes: 2 runes => 6 bytes (too short), 3 runes => 9
		// bytes (ok), 25 runes => 75 bytes (too long). Byte counting must be
		// applied, not rune/character counting.
		{name: "multibyte 6 bytes too short", password: strings.Repeat("你", 2), wantErr: ErrPasswordTooShort},
		{name: "multibyte 9 bytes ok", password: strings.Repeat("你", 3), wantErr: nil},
		{name: "multibyte 75 bytes too long", password: strings.Repeat("你", 25), wantErr: ErrPasswordTooLong},
		// 8-byte multibyte boundary: 2 runes of 你 (6 bytes) + 2 ascii (2 bytes).
		{name: "multibyte exactly 8 bytes ok", password: strings.Repeat("你", 2) + "aa", wantErr: nil},
		// 72-byte multibyte boundary: 24 runes of 你 (72 bytes).
		{name: "multibyte exactly 72 bytes ok", password: strings.Repeat("你", 24), wantErr: nil},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := ValidatePasswordLength(tt.password)
			if tt.wantErr == nil {
				if err != nil {
					t.Fatalf("ValidatePasswordLength() unexpected error: %v", err)
				}
				return
			}
			if err != tt.wantErr {
				t.Fatalf("ValidatePasswordLength() error = %v, want %v", err, tt.wantErr)
			}
		})
	}
}

func TestPasswordHasherRejectsLowCost(t *testing.T) {
	t.Parallel()
	if _, err := NewPasswordHasher(11); err == nil {
		t.Fatal("NewPasswordHasher(11) expected error")
	}
}

func TestPasswordHasherRoundTrip(t *testing.T) {
	t.Parallel()
	h, err := NewPasswordHasher(MinBcryptCost)
	if err != nil {
		t.Fatalf("NewPasswordHasher: %v", err)
	}
	// 8-byte boundary password (admin123 is the documented dev default).
	const password = "admin123"
	hash, err := h.Hash(password)
	if err != nil {
		t.Fatalf("Hash: %v", err)
	}
	if err := h.Compare(hash, password); err != nil {
		t.Fatalf("Compare match: %v", err)
	}
	if err := h.Compare(hash, "wrong-password"); err != ErrPasswordMismatch {
		t.Fatalf("Compare mismatch = %v, want %v", err, ErrPasswordMismatch)
	}
}

func TestPasswordHasherNeverTruncatesLongPassword(t *testing.T) {
	t.Parallel()
	h, err := NewPasswordHasher(MinBcryptCost)
	if err != nil {
		t.Fatalf("NewPasswordHasher: %v", err)
	}
	long := strings.Repeat("x", 73)
	if _, err := h.Hash(long); err != ErrPasswordTooLong {
		t.Fatalf("Hash(73 bytes) = %v, want %v", err, ErrPasswordTooLong)
	}
	// bcrypt itself rejects input beyond 72 bytes rather than truncating, and
	// our length guard rejects it before bcrypt is ever called.
	short := strings.Repeat("x", 72)
	hash, err := h.Hash(short)
	if err != nil {
		t.Fatalf("Hash(72 bytes) = %v", err)
	}
	if err := h.Compare(hash, short); err != nil {
		t.Fatalf("Compare 72-byte roundtrip: %v", err)
	}
}
