package middleware

import (
	"testing"
)

func TestValidRequestID(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		id    string
		valid bool
	}{
		{name: "simple alphanumeric", id: "abc123", valid: true},
		{name: "allowed punctuation", id: "req-fixed_001:v2.0", valid: true},
		{name: "single char", id: "a", valid: true},
		{name: "max length", id: repeatASCII('a', MaxRequestIDLength), valid: true},
		{name: "empty", id: "", valid: false},
		{name: "oversized", id: repeatASCII('a', MaxRequestIDLength+1), valid: false},
		{name: "whitespace space", id: "req 001", valid: false},
		{name: "whitespace tab", id: "req\t001", valid: false},
		{name: "unicode", id: "req-请求-001", valid: false},
		{name: "slash punctuation", id: "req/001", valid: false},
		{name: "at sign", id: "req@001", valid: false},
		{name: "newline", id: "req\n001", valid: false},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := ValidRequestID(tt.id); got != tt.valid {
				t.Fatalf("ValidRequestID(%q) = %v, want %v", tt.id, got, tt.valid)
			}
		})
	}
}

func TestNewRequestIDIsValid(t *testing.T) {
	t.Parallel()

	for i := 0; i < 20; i++ {
		id := newRequestID()
		if !ValidRequestID(id) {
			t.Fatalf("generated request ID %q is invalid", id)
		}
	}
}

func repeatASCII(ch byte, n int) string {
	b := make([]byte, n)
	for i := range b {
		b[i] = ch
	}
	return string(b)
}
