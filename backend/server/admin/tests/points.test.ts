import { describe, expect, it } from "vitest";
import { roundHalfUpTwo, toFourDecimal } from "@/utils/points";

describe("roundHalfUpTwo", () => {
  it("renders every visible points value rounded to exactly two decimals", () => {
    expect(roundHalfUpTwo("0.0000")).toBe("0.00");
    expect(roundHalfUpTwo("123.4567")).toBe("123.46");
    expect(roundHalfUpTwo("123.4549")).toBe("123.45");
    expect(roundHalfUpTwo("123.4550")).toBe("123.46"); // half-up
    expect(roundHalfUpTwo("9999999999999999.9999")).toBe(
      "10000000000000000.00"
    );
    expect(roundHalfUpTwo("-5.0001")).toBe("-5.00");
    expect(roundHalfUpTwo("-5.0050")).toBe("-5.01"); // half-away-from-zero for magnitude
    expect(roundHalfUpTwo("0.0004")).toBe("0.00");
    expect(roundHalfUpTwo("0.0005")).toBe("0.00"); // 4th digit never rounds a hundredth
    expect(roundHalfUpTwo("0.0050")).toBe("0.01");
  });

  it("carries with exact decimal strings, never binary floating point", () => {
    // 2^53 boundary: Number-based increment would silently lose this digit.
    expect(roundHalfUpTwo("9007199254740992.9999")).toBe("9007199254740993.00");
    expect(roundHalfUpTwo("9007199254740993.4999")).toBe("9007199254740993.50");
    expect(roundHalfUpTwo("1234567890123456.9950")).toBe("1234567890123457.00");
    expect(roundHalfUpTwo("9999999999999999.9999")).toBe(
      "10000000000000000.00"
    );
    // tens-digit carry inside the hundredths group
    expect(roundHalfUpTwo("7.0990")).toBe("7.10");
    expect(roundHalfUpTwo("7.1990")).toBe("7.20");
    expect(roundHalfUpTwo("7.9990")).toBe("8.00");
  });

  it("never changes the stored four-decimal value merely by rendering", () => {
    // Rendering is a projection; the source value must remain untouched.
    const stored = "12.3456";
    const shown = roundHalfUpTwo(stored);
    expect(shown).toBe("12.35");
    expect(stored).toBe("12.3456");
  });

  it("handles empty/undefined input", () => {
    expect(roundHalfUpTwo(undefined)).toBe("0.00");
    expect(roundHalfUpTwo("")).toBe("0.00");
  });
});

describe("toFourDecimal", () => {
  it("normalizes user input to fixed four decimals", () => {
    expect(toFourDecimal("100")).toBe("100.0000");
    expect(toFourDecimal("100.5")).toBe("100.5000");
    expect(toFourDecimal("0.0001")).toBe("0.0001");
    expect(toFourDecimal("-5.25")).toBe("-5.2500");
    expect(toFourDecimal("")).toBe("");
  });
});
