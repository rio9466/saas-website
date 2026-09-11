package auth

import (
	"errors"
	"fmt"

	"golang.org/x/crypto/bcrypt"
)

const (
	// MinBcryptCost is the lowest accepted bcrypt cost.
	MinBcryptCost = 12
	// MinPasswordBytes is the minimum password length in bytes (not runes).
	MinPasswordBytes = 8
	// MaxPasswordBytes is the maximum password length in bytes; bcrypt never
	// truncates input longer than 72 bytes (it errors instead).
	MaxPasswordBytes = 72
)

var (
	ErrPasswordTooShort = errors.New("password must be at least 8 bytes")
	ErrPasswordTooLong  = errors.New("password must be at most 72 bytes")
	ErrPasswordMismatch = errors.New("password mismatch")
)

// PasswordHasher hashes and verifies passwords with bcrypt.
type PasswordHasher struct {
	cost int
}

// NewPasswordHasher builds a hasher. Cost must be at least MinBcryptCost.
func NewPasswordHasher(cost int) (*PasswordHasher, error) {
	if cost < MinBcryptCost {
		return nil, fmt.Errorf("bcrypt cost must be at least %d", MinBcryptCost)
	}
	if cost > bcrypt.MaxCost {
		return nil, fmt.Errorf("bcrypt cost must be at most %d", bcrypt.MaxCost)
	}
	return &PasswordHasher{cost: cost}, nil
}

// ValidatePasswordLength rejects passwords outside the 8-72 byte range.
// Passwords are never truncated; bcrypt errors above 72 bytes.
func ValidatePasswordLength(password string) error {
	n := len(password)
	if n < MinPasswordBytes {
		return ErrPasswordTooShort
	}
	if n > MaxPasswordBytes {
		return ErrPasswordTooLong
	}
	return nil
}

// Hash validates length then returns a bcrypt hash of password.
func (h *PasswordHasher) Hash(password string) (string, error) {
	if h == nil {
		return "", errors.New("password hasher is not initialized")
	}
	if err := ValidatePasswordLength(password); err != nil {
		return "", err
	}
	hashed, err := bcrypt.GenerateFromPassword([]byte(password), h.cost)
	if err != nil {
		return "", fmt.Errorf("hash password: %w", err)
	}
	return string(hashed), nil
}

// Compare reports whether password matches a bcrypt hash.
func (h *PasswordHasher) Compare(hash, password string) error {
	if h == nil {
		return errors.New("password hasher is not initialized")
	}
	if err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)); err != nil {
		if errors.Is(err, bcrypt.ErrMismatchedHashAndPassword) {
			return ErrPasswordMismatch
		}
		return fmt.Errorf("compare password: %w", err)
	}
	return nil
}
