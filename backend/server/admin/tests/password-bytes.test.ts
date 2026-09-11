import { describe, it, expect } from "vitest";
import {
  utf8ByteLength,
  passwordByteError,
  MIN_PASSWORD_BYTES,
  MAX_PASSWORD_BYTES
} from "@/utils/password";

describe("密码 UTF-8 字节规则（ADM-003，8-72）", () => {
  it("边界：7 字节拒绝、8 字节接受、72 字节接受、73 字节拒绝", () => {
    expect(passwordByteError("a".repeat(7))).toContain("至少 8");
    expect(passwordByteError("a".repeat(8))).toBeNull();
    expect(passwordByteError("a".repeat(72))).toBeNull();
    expect(passwordByteError("a".repeat(73))).toContain("最多 72");
  });

  it("多字节 Unicode 按 UTF-8 字节数处理（你=3 字节）", () => {
    expect(utf8ByteLength("你你")).toBe(6); // < 8 -> 拒绝
    expect(passwordByteError("你你")).toContain("至少 8");
    expect(passwordByteError("你你你")).toBeNull(); // 9 字节
    expect(utf8ByteLength("你你aa")).toBe(8); // 恰好 8 字节 -> 接受
    expect(passwordByteError("你你aa")).toBeNull();
    // 25 个“你” = 75 字节 -> 拒绝（不能用字符数 25 误判为合法）
    expect(utf8ByteLength("你".repeat(25))).toBe(75);
    expect(passwordByteError("你".repeat(25))).toContain("最多 72");
  });

  it("admin123（8 字节）仍可通过规则", () => {
    expect(utf8ByteLength("admin123")).toBe(8);
    expect(passwordByteError("admin123")).toBeNull();
  });

  it("常量边界正确", () => {
    expect(MIN_PASSWORD_BYTES).toBe(8);
    expect(MAX_PASSWORD_BYTES).toBe(72);
  });
});
