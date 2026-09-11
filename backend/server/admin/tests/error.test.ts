import { describe, it, expect } from "vitest";
import { getErrorMessage, getErrorCode } from "@/utils/error";

function makeError(code: number, message: string, status = 400) {
  return {
    response: {
      status,
      data: { code, message, data: {}, request_id: "x" }
    },
    isAxiosError: true
  } as unknown as Error;
}

describe("getErrorMessage 中文安全文案", () => {
  it("登录失败 20001 → 中文文案（不再直接显示 unauthorized）", () => {
    expect(getErrorMessage(makeError(20001, "unauthorized", 401))).toBe(
      "登录已失效，请重新登录"
    );
  });

  it("token 过期 20002 → 中文文案", () => {
    expect(getErrorMessage(makeError(20002, "token expired", 401))).toBe(
      "登录已过期，请重新登录"
    );
  });

  it("403 30001 → 中文文案", () => {
    expect(getErrorMessage(makeError(30001, "forbidden", 403))).toBe(
      "您没有权限执行此操作"
    );
  });

  it("账号禁用 40006 → 中文文案", () => {
    expect(getErrorMessage(makeError(40006, "account disabled", 403))).toBe(
      "账号已被禁用"
    );
  });

  it("最后超级管理员保护 40003 → 中文文案", () => {
    expect(
      getErrorMessage(
        makeError(
          40003,
          "cannot modify the last enabled super administrator",
          409
        )
      )
    ).toBe("不能修改最后一个启用的超级管理员");
  });

  it("自保护 40004 → 中文文案", () => {
    expect(
      getErrorMessage(
        makeError(
          40004,
          "operation not allowed on the current administrator",
          409
        )
      )
    ).toBe("不能对当前登录账号执行此操作");
  });

  it("审计不可用 50002 → 中文文案", () => {
    expect(getErrorMessage(makeError(50002, "audit unavailable", 503))).toBe(
      "审计服务暂不可用，本次操作未执行，请稍后重试"
    );
  });

  it("未映射的业务 code 按 HTTP 状态给通用中文文案（不直接显示后端英文）", () => {
    expect(
      getErrorMessage(makeError(99999, "some business message", 400))
    ).toBe("请求参数有误，请检查输入");
    expect(getErrorMessage(makeError(99999, "whatever", 500))).toBe(
      "服务器开小差了，请稍后重试"
    );
  });

  it("网络层错误：超时/断连中文提示", () => {
    const timeout = {
      code: "ECONNABORTED",
      message: "timeout of 15000ms exceeded",
      isAxiosError: true
    } as unknown as Error;
    expect(getErrorMessage(timeout)).toBe("请求超时，请稍后重试");

    const offline = {
      message: "Network Error",
      isAxiosError: true
    } as unknown as Error;
    expect(getErrorMessage(offline)).toBe("网络异常，请检查网络连接");
  });
});

describe("getErrorCode", () => {
  it("从信封提取应用错误码", () => {
    expect(getErrorCode(makeError(40006, "account disabled"))).toBe(40006);
  });
  it("无响应时返回 null", () => {
    expect(getErrorCode({ message: "Network Error" })).toBeNull();
  });
});
