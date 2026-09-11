// Package secrets provides authenticated encryption for the SMTP password and
// other application-secret values that must live in PostgreSQL yet never be
// readable back. Ciphertexts are random-nonce AES-256-GCM, base64-encoded.
// The 32-byte master key is loaded from configuration/environment, never from
// PostgreSQL or Git, and no plaintext or ciphertext is ever written to logs.
package secrets

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
)

const (
	// KeySize is the required master key length in bytes.
	KeySize = 32
	// NonceSize is the AES-GCM nonce length.
	NonceSize = 12
)

var (
	// ErrInvalidKey reports a master key that is not exactly 32 bytes.
	ErrInvalidKey = errors.New("smtp master key must be exactly 32 bytes")
	// ErrBoxUnavailable reports that no master key is configured.
	ErrBoxUnavailable = errors.New("smtp secret box is not configured")
	// ErrCiphertextInvalid reports a tampered or malformed stored ciphertext.
	ErrCiphertextInvalid = errors.New("smtp ciphertext invalid")
)

// Box encrypts and decrypts SMTP passwords with AES-256-GCM.
type Box struct {
	aead cipher.AEAD
}

// NewBox builds a Box from a 32-byte master key.
func NewBox(masterKey []byte) (*Box, error) {
	if len(masterKey) != KeySize {
		return nil, fmt.Errorf("%w: got %d bytes", ErrInvalidKey, len(masterKey))
	}
	block, err := aes.NewCipher(masterKey)
	if err != nil {
		return nil, fmt.Errorf("create aes cipher: %w", err)
	}
	aead, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("create gcm: %w", err)
	}
	return &Box{aead: aead}, nil
}

// Encrypt seals plaintext with a fresh random nonce and returns the base64
// nonce+ciphertext blob.
func (b *Box) Encrypt(plaintext string) (string, error) {
	if b == nil || b.aead == nil {
		return "", ErrBoxUnavailable
	}
	nonce := make([]byte, NonceSize)
	if _, err := rand.Read(nonce); err != nil {
		return "", fmt.Errorf("generate nonce: %w", err)
	}
	sealed := b.aead.Seal(nil, nonce, []byte(plaintext), nil)
	blob := make([]byte, 0, len(nonce)+len(sealed))
	blob = append(blob, nonce...)
	blob = append(blob, sealed...)
	return base64.StdEncoding.EncodeToString(blob), nil
}

// Decrypt opens a base64 nonce+ciphertext blob produced by Encrypt.
func (b *Box) Decrypt(encoded string) (string, error) {
	if b == nil || b.aead == nil {
		return "", ErrBoxUnavailable
	}
	if encoded == "" {
		return "", ErrCiphertextInvalid
	}
	blob, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return "", fmt.Errorf("%w: %v", ErrCiphertextInvalid, err)
	}
	if len(blob) <= NonceSize {
		return "", ErrCiphertextInvalid
	}
	nonce := blob[:NonceSize]
	sealed := blob[NonceSize:]
	plain, err := b.aead.Open(nil, nonce, sealed, nil)
	if err != nil {
		return "", fmt.Errorf("%w: %v", ErrCiphertextInvalid, err)
	}
	return string(plain), nil
}
