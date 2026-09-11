import { describe, expect, it } from "vitest";
import { DEFAULT_ADMIN_AVATAR_URL, resolveAdminAvatar } from "@/utils/avatar";
import {
  greetingForHour,
  greetingNow,
  todayText
} from "@/views/dashboard/greeting";

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

describe("dashboard greeting", () => {
  it("maps hours to Chinese greetings", () => {
    expect(greetingForHour(5)).toBe("凌晨好");
    expect(greetingForHour(6)).toBe("早上好");
    expect(greetingForHour(8)).toBe("早上好");
    expect(greetingForHour(9)).toBe("上午好");
    expect(greetingForHour(10)).toBe("上午好");
    expect(greetingForHour(13)).toBe("中午好");
    expect(greetingForHour(16)).toBe("下午好");
    expect(greetingForHour(20)).toBe("晚上好");
    expect(greetingForHour(0)).toBe("凌晨好");
    expect(greetingForHour(24)).toBe("凌晨好");
    expect(greetingForHour(-1)).toBe("晚上好");
  });

  it("derives the greeting from the current time", () => {
    expect(greetingNow(new Date(2026, 8, 7, 7, 30))).toBe("早上好");
    expect(todayText(new Date(2026, 8, 7, 9, 30))).toBe("2026年9月7日 星期一");
    expect(todayText(new Date(2026, 11, 31, 23, 0))).toBe(
      "2026年12月31日 星期四"
    );
  });
});
