import { describe, it, expect } from "vitest";
import { loginRules } from "@/views/login/utils/rule";

function validateField(prop: "username" | "password", value: string) {
  const rule = loginRules[prop];
  const validators = Array.isArray(rule) ? rule : [rule];
  return validators.map(
    v =>
      new Promise<string | null>((resolve, reject) => {
        try {
          const cb = (error?: Error) => resolve(error ? error.message : null);
          // 兼容 required / 自定义 validator 两种形式
          if (typeof v.validator === "function") {
            const validator = v.validator as (
              rule: unknown,
              value: unknown,
              callback: (error?: Error) => void,
              source?: unknown,
              options?: unknown
            ) => void;
            validator({} as never, value, cb, {}, {});
          } else if (v.required && !value) {
            cb(new Error(v.message as string));
          } else {
            cb();
          }
        } catch (e) {
          reject(e);
        }
      })
  );
}

describe("登录校验规则", () => {
  it("账号为空时校验失败", async () => {
    const results = await Promise.all(validateField("username", ""));
    expect(results.some(msg => msg !== null)).toBe(true);
  });

  it("账号非空时校验通过", async () => {
    const results = await Promise.all(validateField("username", "admin"));
    expect(results.every(msg => msg === null)).toBe(true);
  });

  it("密码为空时校验失败", async () => {
    const results = await Promise.all(validateField("password", ""));
    expect(results.some(msg => msg !== null)).toBe(true);
  });

  it("密码非空时校验通过（强度由后端约束，前端不做多余限制）", async () => {
    const results = await Promise.all(
      validateField("password", "long-enough-password-123")
    );
    expect(results.every(msg => msg === null)).toBe(true);
  });
});
