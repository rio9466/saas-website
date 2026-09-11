package user

import (
	"strings"
	"testing"
)

func TestParseDecimal4FormatsAndCompares(t *testing.T) {
	cases := []struct {
		raw  string
		want string
	}{
		{"0", "0.0000"},
		{"0.0000", "0.0000"},
		{"123.4567", "123.4567"},
		{"1.5", "1.5000"},
		{"-5.0000", "-5.0000"},
		{"-0.0001", "-0.0001"},
		{"00000000000000000000.0000", "0.0000"},
		{"9999999999999999.9999", "9999999999999999.9999"},
		{" 12.3400 ", "12.3400"},
	}
	for _, c := range cases {
		d, err := ParseDecimal4(c.raw)
		if err != nil {
			t.Fatalf("ParseDecimal4(%q): %v", c.raw, err)
		}
		if got := d.String(); got != c.want {
			t.Fatalf("ParseDecimal4(%q).String() = %q, want %q", c.raw, got, c.want)
		}
	}

	a, _ := ParseDecimal4("1.0000")
	b, _ := ParseDecimal4("1.0000")
	c1, _ := ParseDecimal4("1.0001")
	if !a.Equal(b) {
		t.Fatal("1.0000 must equal 1.0000")
	}
	if a.Cmp(c1) >= 0 {
		t.Fatal("1.0000 must be less than 1.0001")
	}
}

func TestParseDecimal4RejectsInvalid(t *testing.T) {
	bad := []string{
		"", "  ", "abc", "1.2.3", "1..2", "-", "--1", "1.00000", "1e3", "NaN",
		"10000000000000000.0000", // NUMERIC(20,4) overflow: 17 integer digits
		"9999999999999999.99999", // too many fractions
		"1,5",
	}
	for _, raw := range bad {
		if _, err := ParseDecimal4(raw); err == nil {
			t.Fatalf("ParseDecimal4(%q) must fail", raw)
		}
	}
}

func TestDecimal4ArithmeticExactness(t *testing.T) {
	a, _ := ParseDecimal4("0.0001")
	b, _ := ParseDecimal4("0.0009")
	sum, err := a.Add(b)
	if err != nil {
		t.Fatalf("add: %v", err)
	}
	if sum.String() != "0.0010" {
		t.Fatalf("0.0001+0.0009 = %q, want 0.0010", sum.String())
	}

	// Classic floating-point trap (0.1 + 0.2) is exact in decimal4.
	x, _ := ParseDecimal4("0.1000")
	y, _ := ParseDecimal4("0.2000")
	z, err := x.Add(y)
	if err != nil {
		t.Fatalf("add: %v", err)
	}
	if z.String() != "0.3000" {
		t.Fatalf("0.1000+0.2000 = %q, want 0.3000", z.String())
	}

	diff, err := x.Sub(y)
	if err != nil {
		t.Fatalf("sub: %v", err)
	}
	if diff.String() != "-0.1000" {
		t.Fatalf("0.1000-0.2000 = %q, want -0.1000", diff.String())
	}
	if !diff.IsNegative() {
		t.Fatal("result must be negative")
	}
	if diff.IsZero() {
		t.Fatal("result must not be zero")
	}
}

func TestDecimal4OverflowFailsClosed(t *testing.T) {
	max, err := ParseDecimal4("9999999999999999.9999")
	if err != nil {
		t.Fatalf("parse max: %v", err)
	}
	if _, err := max.Add(max); !strings.Contains(err.Error(), "overflow") {
		t.Fatalf("overflow must fail with a bounded error, got %v", err)
	}
}

func TestDecimal4Zero(t *testing.T) {
	z := Zero()
	if !z.IsZero() {
		t.Fatal("zero must be zero")
	}
	if z.String() != "0.0000" {
		t.Fatalf("zero string = %q", z.String())
	}
}
