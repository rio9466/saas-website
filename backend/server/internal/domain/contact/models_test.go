package contact

import (
	"errors"
	"strings"
	"testing"
)

func validInput() SubmissionInput {
	return SubmissionInput{
		Name:    "Alice",
		Email:   "alice@example.com",
		Company: "ACME",
		Message: "Hello there",
		Locale:  "en",
		Consent: true,
	}
}

func TestValidateAcceptsValidSubmission(t *testing.T) {
	t.Parallel()
	if err := Validate(validInput()); err != nil {
		t.Fatalf("Validate(valid) = %v, want nil", err)
	}
}

func TestValidateAcceptsMaximumLengthMessage(t *testing.T) {
	t.Parallel()
	in := validInput()
	in.Message = strings.Repeat("a", MaxMessageLength)
	if err := Validate(in); err != nil {
		t.Fatalf("Validate(max message) = %v, want nil", err)
	}
}

func TestValidateRejectsInvalidSubmissions(t *testing.T) {
	t.Parallel()
	cases := map[string]func(in *SubmissionInput){
		"missing name":     func(in *SubmissionInput) { in.Name = "" },
		"missing email":    func(in *SubmissionInput) { in.Email = "" },
		"invalid email":    func(in *SubmissionInput) { in.Email = "not-an-email" },
		"email list":       func(in *SubmissionInput) { in.Email = "a@b.com, c@d.com" },
		"missing message":  func(in *SubmissionInput) { in.Message = "" },
		"consent false":    func(in *SubmissionInput) { in.Consent = false },
		"message too long": func(in *SubmissionInput) { in.Message = strings.Repeat("a", MaxMessageLength+1) },
		"name too long":    func(in *SubmissionInput) { in.Name = strings.Repeat("a", MaxNameLength+1) },
	}
	for name, mutate := range cases {
		mutate := mutate
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			in := validInput()
			mutate(&in)
			err := Validate(in)
			if !errors.Is(err, ErrInvalid) {
				t.Fatalf("Validate() = %v, want ErrInvalid", err)
			}
		})
	}
}

func TestNormalizedTrimsWhitespace(t *testing.T) {
	t.Parallel()
	in := SubmissionInput{Name: "  Alice \n", Email: " a@b.com ", Message: " hi ", Locale: " en "}
	got := in.Normalized()
	if got.Name != "Alice" || got.Email != "a@b.com" || got.Message != "hi" || got.Locale != "en" {
		t.Fatalf("Normalized() = %+v", got)
	}
}

func TestHoneypotTripped(t *testing.T) {
	t.Parallel()
	if (SubmissionInput{Website: ""}).HoneypotTripped() {
		t.Fatal("empty website must not trip the honeypot")
	}
	if (SubmissionInput{Website: "   "}).HoneypotTripped() {
		t.Fatal("whitespace website must not trip the honeypot")
	}
	if !(SubmissionInput{Website: "http://spam"}).HoneypotTripped() {
		t.Fatal("non-empty website must trip the honeypot")
	}
}

func TestValidStatus(t *testing.T) {
	t.Parallel()
	for _, ok := range []string{StatusNew, StatusRead, StatusHandled, " read "} {
		if !ValidStatus(ok) {
			t.Fatalf("ValidStatus(%q) = false, want true", ok)
		}
	}
	for _, bad := range []string{"", "deleted", "NEW "} {
		if ValidStatus(bad) {
			t.Fatalf("ValidStatus(%q) = true, want false", bad)
		}
	}
}
