import { describe, expect, it } from "vitest";
import {
  compactTranslations,
  isEmptyValue,
  listToText,
  textToList
} from "@/views/content/translations";

describe("content translation helpers", () => {
  it("treats blank/null/empty values as empty", () => {
    expect(isEmptyValue(undefined)).toBe(true);
    expect(isEmptyValue(null)).toBe(true);
    expect(isEmptyValue("   ")).toBe(true);
    expect(isEmptyValue([])).toBe(true);
    expect(isEmptyValue({})).toBe(true);
    expect(isEmptyValue("x")).toBe(false);
    expect(isEmptyValue(["x"])).toBe(false);
  });

  it("keeps only locales that actually have content", () => {
    const compacted = compactTranslations({
      en: { title: "About", body_md: "" },
      "zh-CN": { title: "", body_md: "   " },
      fr: { title: "", body_md: "" }
    });
    expect(Object.keys(compacted)).toEqual(["en"]);
    expect(compacted.en.title).toBe("About");
  });

  it("keeps a locale whose only value is a non-empty array", () => {
    const compacted = compactTranslations({
      en: { features: ["one"] },
      "zh-CN": { features: [] }
    });
    expect(Object.keys(compacted)).toEqual(["en"]);
  });

  it("round-trips a one-item-per-line list", () => {
    expect(textToList("a\n\n b \n")).toEqual(["a", "b"]);
    expect(listToText(["a", "b"])).toBe("a\nb");
    expect(listToText(undefined)).toBe("");
  });
});
