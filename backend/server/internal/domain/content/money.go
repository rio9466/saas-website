package content

import (
	"errors"
	"regexp"
	"strings"
)

// ErrInvalidMoney reports a malformed or out-of-range money string.
var ErrInvalidMoney = errors.New("invalid money value")

// moneyRE accepts a non-negative decimal with at most two fractional digits
// and at most 18 integer digits (NUMERIC(20,2)).
var moneyRE = regexp.MustCompile(`^\d{1,18}(\.\d{1,2})?$`)

// NormalizeMoney validates a decimal money string and returns it with exactly
// two fractional digits. An empty value normalizes to "0.00".
func NormalizeMoney(raw string) (string, error) {
	s := strings.TrimSpace(raw)
	if s == "" {
		return "0.00", nil
	}
	if strings.HasPrefix(s, "+") {
		s = s[1:]
	}
	if !moneyRE.MatchString(s) {
		return "", ErrInvalidMoney
	}
	intPart := s
	fracPart := "00"
	if idx := strings.IndexByte(s, '.'); idx >= 0 {
		intPart = s[:idx]
		fracPart = s[idx+1:]
	}
	for len(fracPart) < 2 {
		fracPart += "0"
	}
	trimmed := strings.TrimLeft(intPart, "0")
	if trimmed == "" {
		trimmed = "0"
	}
	return trimmed + "." + fracPart, nil
}
