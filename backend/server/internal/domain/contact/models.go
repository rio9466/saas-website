package contact

import (
	"errors"
	"fmt"
	"net/mail"
	"strings"
	"time"
	"unicode/utf8"
)

// Submission status values. New submissions start as StatusNew; administrators
// may only move them to read or handled.
const (
	StatusNew     = "new"
	StatusRead    = "read"
	StatusHandled = "handled"
)

// Field length limits. message is the documented product cap; the smaller
// limits bound name/email/company so a hostile payload cannot bloat storage.
const (
	MaxNameLength    = 200
	MaxEmailLength   = 254
	MaxCompanyLength = 200
	MaxMessageLength = 5000
	MaxLocaleLength  = 35
)

// ErrInvalid reports a contact submission that failed validation.
var ErrInvalid = errors.New("invalid contact submission")

// SubmissionInput is a raw public contact-form submission as bound from JSON.
type SubmissionInput struct {
	Name    string `json:"name"`
	Email   string `json:"email"`
	Company string `json:"company"`
	Message string `json:"message"`
	Locale  string `json:"locale"`
	Consent bool   `json:"consent"`
	// Website is the hidden honeypot field: bots fill it in, humans never
	// see it. A non-empty value short-circuits to a silent success.
	Website string `json:"website"`
}

// Normalized returns a copy with surrounding whitespace removed from the
// string fields.
func (in SubmissionInput) Normalized() SubmissionInput {
	return SubmissionInput{
		Name:    strings.TrimSpace(in.Name),
		Email:   strings.TrimSpace(in.Email),
		Company: strings.TrimSpace(in.Company),
		Message: strings.TrimSpace(in.Message),
		Locale:  strings.TrimSpace(in.Locale),
		Consent: in.Consent,
		Website: in.Website,
	}
}

// HoneypotTripped reports whether the hidden anti-spam field was filled in.
func (in SubmissionInput) HoneypotTripped() bool {
	return strings.TrimSpace(in.Website) != ""
}

// Submission is a persisted contact-form submission.
type Submission struct {
	ID        int64
	Name      string
	Email     string
	Company   string
	Message   string
	Locale    string
	Status    string
	SourceIP  string
	UserAgent string
	CreatedAt time.Time
	UpdatedAt time.Time
}

// Receipt is the public acknowledgement returned after a submission.
type Receipt struct {
	ID          int64
	SubmittedAt time.Time
}

// ValidStatus reports whether status is one of the inbox statuses.
func ValidStatus(status string) bool {
	switch strings.TrimSpace(status) {
	case StatusNew, StatusRead, StatusHandled:
		return true
	default:
		return false
	}
}

// Validate checks the required fields, consent, email shape, and length caps.
// The returned error message is client-safe.
func Validate(in SubmissionInput) error {
	if in.Name == "" {
		return fmt.Errorf("%w: name is required", ErrInvalid)
	}
	if utf8.RuneCountInString(in.Name) > MaxNameLength {
		return fmt.Errorf("%w: name is too long", ErrInvalid)
	}
	if in.Email == "" {
		return fmt.Errorf("%w: email is required", ErrInvalid)
	}
	if utf8.RuneCountInString(in.Email) > MaxEmailLength || !validEmail(in.Email) {
		return fmt.Errorf("%w: email is not a valid address", ErrInvalid)
	}
	if in.Message == "" {
		return fmt.Errorf("%w: message is required", ErrInvalid)
	}
	if utf8.RuneCountInString(in.Message) > MaxMessageLength {
		return fmt.Errorf("%w: message is too long", ErrInvalid)
	}
	if utf8.RuneCountInString(in.Company) > MaxCompanyLength {
		return fmt.Errorf("%w: company is too long", ErrInvalid)
	}
	if utf8.RuneCountInString(in.Locale) > MaxLocaleLength {
		return fmt.Errorf("%w: locale is too long", ErrInvalid)
	}
	if !in.Consent {
		return fmt.Errorf("%w: consent is required", ErrInvalid)
	}
	return nil
}

func validEmail(raw string) bool {
	// A bare address only: display names and lists are rejected.
	if strings.ContainsAny(raw, " \t\r\n") {
		return false
	}
	addr, err := mail.ParseAddress(raw)
	if err != nil {
		return false
	}
	return addr.Address == raw && strings.Contains(raw, "@")
}
