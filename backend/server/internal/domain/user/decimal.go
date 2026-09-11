package user

import (
	"errors"
	"fmt"
	"math/big"
	"strings"
)

// decimalScale is the fixed scale for business-user points: exactly four
// fractional digits. Floating-point arithmetic is prohibited for points, so
// every value is carried as an unscaled big integer of 1/10000 units.
const decimalScale = 10000

// maxUnscaled bounds the unscaled value for NUMERIC(20,4): 16 integer digits
// plus 4 fractional digits, i.e. |unscaled| <= 10^20 - 1.
var maxUnscaled = new(big.Int).Exp(big.NewInt(10), big.NewInt(20), nil)

var (
	// ErrDecimalInvalid reports a malformed or out-of-range decimal string.
	ErrDecimalInvalid = errors.New("invalid decimal value")
	// ErrDecimalOverflow reports an arithmetic result outside NUMERIC(20,4).
	ErrDecimalOverflow = errors.New("decimal overflow")
)

// Decimal4 is an immutable exact decimal with four fixed fractional digits.
// The zero value represents 0.0000.
type Decimal4 struct {
	unscaled *big.Int
}

// ParseDecimal4 parses a fixed four-decimal string such as "123.4567".
func ParseDecimal4(raw string) (Decimal4, error) {
	s := strings.TrimSpace(raw)
	if s == "" {
		return Decimal4{}, ErrDecimalInvalid
	}
	neg := false
	if strings.HasPrefix(s, "-") || strings.HasPrefix(s, "+") {
		neg = strings.HasPrefix(s, "-")
		s = s[1:]
	}
	if s == "" {
		return Decimal4{}, ErrDecimalInvalid
	}
	if strings.Count(s, ".") > 1 {
		return Decimal4{}, ErrDecimalInvalid
	}
	parts := strings.SplitN(s, ".", 2)
	intPart := parts[0]
	fracPart := ""
	if len(parts) == 2 {
		fracPart = parts[1]
	}
	if intPart == "" {
		intPart = "0"
	}
	if !isAllDigits(intPart) || (fracPart != "" && !isAllDigits(fracPart)) {
		return Decimal4{}, ErrDecimalInvalid
	}
	if len(fracPart) > 4 {
		return Decimal4{}, ErrDecimalInvalid
	}
	// Normalize to four fractional digits.
	for len(fracPart) < 4 {
		fracPart += "0"
	}

	trimmed := strings.TrimLeft(intPart, "0")
	if trimmed == "" {
		trimmed = "0"
	}
	unscaled, ok := new(big.Int).SetString(trimmed+fracPart, 10)
	if !ok {
		return Decimal4{}, ErrDecimalInvalid
	}
	if neg {
		unscaled.Neg(unscaled)
	}
	if unscaled.Cmp(maxUnscaled) >= 0 || unscaled.Cmp(new(big.Int).Neg(maxUnscaled)) <= 0 {
		return Decimal4{}, ErrDecimalInvalid
	}
	return Decimal4{unscaled: unscaled}, nil
}

// Decimal4FromUnscaled builds a Decimal4 from an unscaled 1/10000 integer.
func Decimal4FromUnscaled(unscaled *big.Int) (Decimal4, error) {
	if unscaled == nil {
		return Decimal4{}, ErrDecimalInvalid
	}
	if unscaled.Cmp(maxUnscaled) >= 0 || unscaled.Cmp(new(big.Int).Neg(maxUnscaled)) <= 0 {
		return Decimal4{}, ErrDecimalInvalid
	}
	return Decimal4{unscaled: new(big.Int).Set(unscaled)}, nil
}

// Zero returns the 0.0000 value.
func Zero() Decimal4 {
	return Decimal4{unscaled: big.NewInt(0)}
}

// Unscaled returns a copy of the integer value in 1/10000 units.
func (d Decimal4) Unscaled() *big.Int {
	if d.unscaled == nil {
		return big.NewInt(0)
	}
	return new(big.Int).Set(d.unscaled)
}

// String renders the exact fixed four-decimal representation.
func (d Decimal4) String() string {
	if d.unscaled == nil {
		return "0.0000"
	}
	n := new(big.Int).Set(d.unscaled)
	neg := n.Sign() < 0
	if neg {
		n.Neg(n)
	}
	s := n.String()
	for len(s) < 5 {
		s = "0" + s
	}
	intPart := s[:len(s)-4]
	fracPart := s[len(s)-4:]
	if neg {
		return "-" + intPart + "." + fracPart
	}
	return intPart + "." + fracPart
}

// Cmp returns -1, 0, or 1 comparing d to other.
func (d Decimal4) Cmp(other Decimal4) int {
	return d.Unscaled().Cmp(other.Unscaled())
}

// Equal reports whether d and other have the same exact value.
func (d Decimal4) Equal(other Decimal4) bool {
	return d.Cmp(other) == 0
}

// IsNegative reports whether d < 0.
func (d Decimal4) IsNegative() bool {
	return d.Unscaled().Sign() < 0
}

// IsZero reports whether d == 0.0000.
func (d Decimal4) IsZero() bool {
	return d.Unscaled().Sign() == 0
}

// Add returns d + other, or ErrDecimalOverflow outside NUMERIC(20,4).
func (d Decimal4) Add(other Decimal4) (Decimal4, error) {
	sum := new(big.Int).Add(d.Unscaled(), other.Unscaled())
	out, err := Decimal4FromUnscaled(sum)
	if err != nil {
		return Decimal4{}, fmt.Errorf("%w: %s + %s", ErrDecimalOverflow, d.String(), other.String())
	}
	return out, nil
}

// Sub returns d - other, or ErrDecimalOverflow outside NUMERIC(20,4).
func (d Decimal4) Sub(other Decimal4) (Decimal4, error) {
	diff := new(big.Int).Sub(d.Unscaled(), other.Unscaled())
	out, err := Decimal4FromUnscaled(diff)
	if err != nil {
		return Decimal4{}, fmt.Errorf("%w: %s - %s", ErrDecimalOverflow, d.String(), other.String())
	}
	return out, nil
}

func isAllDigits(s string) bool {
	if s == "" {
		return false
	}
	for _, r := range s {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}
