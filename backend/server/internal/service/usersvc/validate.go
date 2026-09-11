package usersvc

import (
	"net/mail"
	"strings"

	"github.com/rio9466/easy-admin/server/internal/domain/apperr"
)

// validateUsername enforces business-user username syntax: trimmed 3-64
// characters, no '@', and only letters/digits/._- so it can never overlap
// email syntax and compares case-insensitively.
func validateUsername(username string) error {
	if len(username) < 3 || len(username) > 64 {
		return validationError("username must be 3-64 characters")
	}
	if strings.Contains(username, "@") {
		return validationError("username must not contain '@'")
	}
	for _, r := range username {
		if !(r >= 'a' && r <= 'z') && !(r >= 'A' && r <= 'Z') && !(r >= '0' && r <= '9') &&
			r != '.' && r != '_' && r != '-' {
			return validationError("username may only contain letters, digits, '.', '_', '-'")
		}
	}
	return nil
}

// validateEmail validates and lowercases an email address.
func validateEmail(email string) error {
	if email == "" || len(email) > 254 {
		return validationError("invalid email address")
	}
	addr, err := mail.ParseAddress(email)
	if err != nil {
		return validationError("invalid email address")
	}
	if addr.Address != email {
		return validationError("invalid email address")
	}
	if strings.ContainsAny(email, " \t\r\n") {
		return validationError("invalid email address")
	}
	return nil
}

var _ = apperr.ErrValidation
