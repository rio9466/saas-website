package secrets

import (
	"bytes"
	"strings"
	"testing"
)

func testKey() []byte {
	return []byte("0123456789abcdef0123456789abcdef") // exactly 32 bytes
}

func TestBoxRoundTripEncryptDecrypt(t *testing.T) {
	box, err := NewBox(testKey())
	if err != nil {
		t.Fatalf("NewBox: %v", err)
	}
	plaintext := "smtp-password-with-@-symbol-and-ümlaut"
	enc, err := box.Encrypt(plaintext)
	if err != nil {
		t.Fatalf("Encrypt: %v", err)
	}
	if enc == "" || enc == plaintext || strings.Contains(enc, plaintext) {
		t.Fatalf("ciphertext must never contain plaintext, got %q", enc)
	}
	dec, err := box.Decrypt(enc)
	if err != nil {
		t.Fatalf("Decrypt: %v", err)
	}
	if dec != plaintext {
		t.Fatalf("roundtrip = %q, want %q", dec, plaintext)
	}
}

func TestBoxNonceUniqueness(t *testing.T) {
	box, _ := NewBox(testKey())
	a, err := box.Encrypt("same value")
	if err != nil {
		t.Fatal(err)
	}
	b, err := box.Encrypt("same value")
	if err != nil {
		t.Fatal(err)
	}
	if a == b {
		t.Fatal("two encryptions of the same value must differ (random nonces)")
	}
}

func TestBoxWrongKeyAndTamperFail(t *testing.T) {
	box, _ := NewBox(testKey())
	enc, err := box.Encrypt("secret")
	if err != nil {
		t.Fatal(err)
	}

	wrong, _ := NewBox([]byte("fedcba9876543210fedcba9876543210"))
	if _, err := wrong.Decrypt(enc); err == nil {
		t.Fatal("wrong key must fail to decrypt")
	}

	tampered := enc[:len(enc)-2] + flipBit(enc[len(enc)-2:])
	if _, err := box.Decrypt(tampered); err == nil {
		t.Fatal("tampered ciphertext must fail authentication")
	}

	if _, err := box.Decrypt("not-base64!!"); err == nil {
		t.Fatal("malformed ciphertext must fail")
	}
	if _, err := box.Decrypt(""); err == nil {
		t.Fatal("empty ciphertext must fail")
	}
}

func TestBoxRequiresExactKeySize(t *testing.T) {
	for _, k := range [][]byte{nil, {}, []byte("short"), bytes.Repeat([]byte("x"), 31), bytes.Repeat([]byte("x"), 33)} {
		if _, err := NewBox(k); err == nil {
			t.Fatalf("NewBox with %d-byte key must fail", len(k))
		}
	}
}

func TestBoxUnavailableWithoutConstruction(t *testing.T) {
	var box *Box
	if _, err := box.Encrypt("x"); err == nil {
		t.Fatal("nil box must fail closed")
	}
	if _, err := box.Decrypt("x"); err == nil {
		t.Fatal("nil box must fail closed")
	}
}

func flipBit(s string) string {
	b := []byte(s)
	b[len(b)-1] ^= 0x01
	return string(b)
}
