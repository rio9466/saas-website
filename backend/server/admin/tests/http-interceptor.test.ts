import { describe, it, expect, vi, beforeAll, afterAll } from "vitest";
import http from "node:http";
import type { AddressInfo } from "node:net";
import { createHttpClient, type HttpTokenApi } from "@/utils/http";

/**
 * 真实 axios + 本地 fake server 的拦截器级联测：
 * 覆盖 401 → 单飞 refresh → 原请求重试一次的完整链路，
 * 以及重试上限（再 401 清认证）、白名单不刷新、非 401 直返。
 */
const BASE = "/api/v1/admin";

function startFakeServer() {
  const server = http.createServer((req, res) => {
    const url = req.url ?? "";
    req.resume();
    req.on("end", () => {
      const send = (status: number, body: unknown) => {
        res.writeHead(status, { "Content-Type": "application/json" });
        res.end(JSON.stringify(body));
      };
      if (url === `${BASE}/me`) {
        const auth = req.headers["authorization"] ?? "";
        if (auth === "Bearer new-token") {
          send(200, {
            code: 0,
            message: "success",
            data: { ok: true },
            request_id: "r3"
          });
        } else if (auth === "Bearer expired") {
          send(401, {
            code: 20002,
            message: "token expired",
            data: {},
            request_id: "r4"
          });
        } else {
          send(401, {
            code: 20001,
            message: "unauthorized",
            data: {},
            request_id: "r5"
          });
        }
        return;
      }
      if (url === `${BASE}/server-error`) {
        send(500, {
          code: 50001,
          message: "internal",
          data: {},
          request_id: "r6"
        });
        return;
      }
      if (url === `${BASE}/flaky-get`) {
        // 首次（Bearer expired）401，刷新后的重试（Bearer new-token）返回 500
        const auth = req.headers["authorization"] ?? "";
        if (auth === "Bearer new-token") {
          send(500, {
            code: 50001,
            message: "internal",
            data: {},
            request_id: "r8"
          });
        } else {
          send(401, {
            code: 20002,
            message: "token expired",
            data: {},
            request_id: "r9"
          });
        }
        return;
      }
      if (url === `${BASE}/flake-forbidden`) {
        // 首次 401，刷新后重试返回 403
        const auth = req.headers["authorization"] ?? "";
        if (auth === "Bearer new-token") {
          send(403, {
            code: 30001,
            message: "forbidden",
            data: {},
            request_id: "r10"
          });
        } else {
          send(401, {
            code: 20002,
            message: "token expired",
            data: {},
            request_id: "r11"
          });
        }
        return;
      }
      send(404, {
        code: 40002,
        message: "not found",
        data: {},
        request_id: "r7"
      });
    });
  });
  return new Promise<{ server: http.Server; port: number }>(resolve => {
    server.listen(0, "127.0.0.1", () => {
      const port = (address => address.port)(server.address() as AddressInfo);
      resolve({ server, port });
    });
  });
}

describe("http 拦截器（真实 axios + fake server）", () => {
  let server: http.Server | undefined;
  let port = 0;

  beforeAll(async () => {
    ({ server, port } = await startFakeServer());
  });
  afterAll(() => {
    server?.close();
  });

  function makeClient(opts?: {
    token?: string;
    refreshFail?: boolean;
    refreshDelayMs?: number;
  }) {
    let token = opts?.token ?? "expired";
    const onAuthExpired = vi.fn();
    const refreshMock = vi.fn(async () => {
      if (opts?.refreshDelayMs) {
        await new Promise(resolve => setTimeout(resolve, opts.refreshDelayMs));
      }
      token = "new-token";
      return "new-token";
    });
    const effectiveRefresh = () =>
      opts?.refreshFail
        ? Promise.reject(new Error("refresh backend down"))
        : refreshMock();

    const tokenApi: HttpTokenApi = {
      getToken: () => token,
      clearToken: vi.fn(() => {
        token = "";
      }),
      refresh: effectiveRefresh,
      isWhiteList: url =>
        url.endsWith("/auth/login") || url.endsWith("/auth/refresh"),
      onAuthExpired
    };
    const client = createHttpClient(tokenApi, {
      baseURL: `http://127.0.0.1:${port}/api/v1/admin`
    });
    return { client, tokenApi, onAuthExpired, refreshMock };
  }

  type Envelope = {
    code: number;
    message: string;
    data: unknown;
    request_id: string;
  };
  function unwrap<T>(res: Envelope): T {
    return res.data as T;
  }

  it("正常请求：返回信封且 data 只解包一次", async () => {
    const { client } = makeClient({ token: "new-token" });
    const res = await client.request<Envelope>("get", "/me");
    expect(unwrap<{ ok: boolean }>(res)).toEqual({ ok: true });
  });

  it("401 → 触发一次 refresh → 用新 token 重试一次并成功", async () => {
    const { client, refreshMock, onAuthExpired } = makeClient({
      token: "expired"
    });
    const res = await client.request<Envelope>("get", "/me");
    expect(unwrap<{ ok: boolean }>(res)).toEqual({ ok: true });
    expect(refreshMock).toHaveBeenCalledTimes(1);
    expect(onAuthExpired).not.toHaveBeenCalled();
  });

  it("并发多个 401：单飞刷新只调用一次，全部重试成功", async () => {
    const { client, refreshMock } = makeClient({
      token: "expired",
      refreshDelayMs: 60
    });
    const results = await Promise.all([
      client.request<Envelope>("get", "/me"),
      client.request<Envelope>("get", "/me"),
      client.request<Envelope>("get", "/me"),
      client.request<Envelope>("get", "/me"),
      client.request<Envelope>("get", "/me")
    ]);
    expect(results).toHaveLength(5);
    results.forEach(r =>
      expect(unwrap<{ ok: boolean }>(r)).toEqual({ ok: true })
    );
    expect(refreshMock).toHaveBeenCalledTimes(1);
  });

  it("重试后仍然 401：清理认证状态并不再重试（不刷新循环）", async () => {
    let token = "expired";
    const onAuthExpired = vi.fn();
    const clearToken = vi.fn(() => {
      token = "";
    });
    const tokenApi: HttpTokenApi = {
      getToken: () => token,
      clearToken,
      refresh: async () => {
        token = "still-bad"; // 模拟刷新成功但新 token 依然无权限
        return token;
      },
      isWhiteList: url =>
        url.endsWith("/auth/login") || url.endsWith("/auth/refresh"),
      onAuthExpired
    };
    const client = createHttpClient(tokenApi, {
      baseURL: `http://127.0.0.1:${port}/api/v1/admin`
    });
    await expect(client.request("get", "/me")).rejects.toBeTruthy();
    expect(clearToken).toHaveBeenCalledTimes(1);
    expect(onAuthExpired).toHaveBeenCalledTimes(1);
  });

  it("刷新成功后重试返回 500：原样拒绝，保留 token，不清理认证、不触发过期回调", async () => {
    const { client, tokenApi, onAuthExpired, refreshMock } = makeClient({
      token: "expired"
    });
    const err = (await client
      .request("get", "/flaky-get")
      .then(() => undefined)
      .catch(e => e as { response?: { status?: number } })) as
      | { response?: { status?: number } }
      | undefined;
    expect(err?.response?.status).toBe(500);
    expect(refreshMock).toHaveBeenCalledTimes(1); // 只刷新一次，不再次刷新
    expect(tokenApi.clearToken).not.toHaveBeenCalled();
    expect(onAuthExpired).not.toHaveBeenCalled();
    // token 仍为刷新后的新 token（未被清空）
    expect(tokenApi.getToken()).toBe("new-token");
  });

  it("刷新成功后重试返回 403：原样拒绝，不清理认证、不触发过期回调", async () => {
    const { client, tokenApi, onAuthExpired, refreshMock } = makeClient({
      token: "expired"
    });
    const err = (await client
      .request("get", "/flake-forbidden")
      .then(() => undefined)
      .catch(e => e as { response?: { status?: number } })) as
      | { response?: { status?: number } }
      | undefined;
    expect(err?.response?.status).toBe(403);
    expect(refreshMock).toHaveBeenCalledTimes(1);
    expect(tokenApi.clearToken).not.toHaveBeenCalled();
    expect(onAuthExpired).not.toHaveBeenCalled();
    expect(tokenApi.getToken()).toBe("new-token");
  });

  it("刷新失败：清理认证状态并拒绝原请求", async () => {
    const { client, onAuthExpired } = makeClient({
      token: "expired",
      refreshFail: true
    });
    await expect(client.request("get", "/me")).rejects.toBeTruthy();
    expect(onAuthExpired).toHaveBeenCalledTimes(1);
  });

  it("白名单接口（login/refresh）401 时不触发刷新", async () => {
    const { client, refreshMock } = makeClient({ token: "expired" });
    await expect(client.request("post", "/auth/login")).rejects.toBeTruthy();
    await expect(client.request("post", "/auth/refresh")).rejects.toBeTruthy();
    expect(refreshMock).not.toHaveBeenCalled();
  });

  it("非 401（500）直接拒绝，不触发刷新", async () => {
    const { client, refreshMock } = makeClient({ token: "expired" });
    const err: { response?: { status?: number } } | undefined = await client
      .request("get", "/server-error")
      .then(() => undefined)
      .catch(e => e as { response?: { status?: number } });
    expect(err?.response?.status).toBe(500);
    expect(refreshMock).not.toHaveBeenCalled();
  });
});
