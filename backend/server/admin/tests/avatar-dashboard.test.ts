import { describe, expect, it } from "vitest";
import { DEFAULT_ADMIN_AVATAR_URL, resolveAdminAvatar } from "@/utils/avatar";

describe("admin avatar resolution", () => {
  it("uses the designated fallback URL when avatar is empty", () => {
    expect(resolveAdminAvatar("")).toBe(DEFAULT_ADMIN_AVATAR_URL);
    expect(resolveAdminAvatar("   ")).toBe(DEFAULT_ADMIN_AVATAR_URL);
    expect(resolveAdminAvatar(undefined)).toBe(DEFAULT_ADMIN_AVATAR_URL);
    expect(resolveAdminAvatar(null)).toBe(DEFAULT_ADMIN_AVATAR_URL);
  });

  it("prefers the existing avatar when non-empty", () => {
    expect(resolveAdminAvatar("https://example.com/me.png")).toBe(
      "https://example.com/me.png"
    );
    expect(resolveAdminAvatar("data:image/png;base64,abc")).toBe(
      "data:image/png;base64,abc"
    );
  });

  it("exposes the exact designated default URL", () => {
    expect(DEFAULT_ADMIN_AVATAR_URL).toBe(
      "https://avatars.githubusercontent.com/u/310554857?v=4"
    );
  });
});
