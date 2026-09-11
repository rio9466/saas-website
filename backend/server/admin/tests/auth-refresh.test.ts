import { describe, it, expect, vi, beforeEach } from "vitest";
import { RefreshGate, decideAfter401 } from "@/utils/refreshGate";

describe("RefreshGate 单飞刷新", () => {
  let refreshFn: ReturnType<typeof vi.fn>;

  beforeEach(() => {
    refreshFn = vi.fn(() => Promise.resolve("new-token"));
  });

  it("并发 401 只触发一次刷新，其余复用同一结果", async () => {
    const gate = new RefreshGate(refreshFn);
    const results = await Promise.all([
      gate.acquire(),
      gate.acquire(),
      gate.acquire()
    ]);
    expect(results).toEqual(["new-token", "new-token", "new-token"]);
    expect(refreshFn).toHaveBeenCalledTimes(1);
  });

  it("刷新完成后，下一次 acquire 会发起新的刷新（不串号）", async () => {
    const gate = new RefreshGate(refreshFn);
    await gate.acquire();
    await gate.acquire();
    expect(refreshFn).toHaveBeenCalledTimes(2);
  });

  it("刷新失败时所有等待者都收到拒绝，且在途状态被清理", async () => {
    refreshFn = vi.fn(() => Promise.reject(new Error("refresh failed")));
    const gate = new RefreshGate(refreshFn);
    const first = gate.acquire();
    const second = gate.acquire();
    await expect(first).rejects.toThrow("refresh failed");
    await expect(second).rejects.toThrow("refresh failed");
    expect(gate.isInflight).toBe(false);
    // 失败后再次触发会重试刷新（不陷入死锁）
    refreshFn.mockImplementationOnce(() => Promise.resolve("recovered"));
    await expect(gate.acquire()).resolves.toBe("recovered");
  });
});

describe("decideAfter401 重试上限与白名单", () => {
  it("白名单（login/refresh）直接跳过，不触发刷新", () => {
    expect(decideAfter401(true, false)).toBe("skip");
    expect(decideAfter401(true, true)).toBe("skip");
  });

  it("首次 401 允许刷新重试", () => {
    expect(decideAfter401(false, false)).toBe("refresh");
  });

  it("已重试过一次仍然 401 时清理认证状态，不再重试", () => {
    expect(decideAfter401(false, true)).toBe("clear");
  });
});
