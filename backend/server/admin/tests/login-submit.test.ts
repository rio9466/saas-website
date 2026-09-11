import { describe, it, expect, vi } from "vitest";
import { createLoginSubmit } from "@/utils/loginSubmit";

function buildDeps(opts?: {
  validate?: () => Promise<boolean>;
  doLogin?: () => Promise<void>;
}) {
  const onSuccess = vi.fn();
  const onError = vi.fn();
  const validate = opts?.validate ?? vi.fn(async () => true);
  const doLogin = opts?.doLogin ?? vi.fn(async () => {});
  const submit = createLoginSubmit({ validate, doLogin, onSuccess, onError });
  return {
    submit,
    onSuccess,
    onError,
    validate: validate as unknown as ReturnType<typeof vi.fn>,
    doLogin: doLogin as unknown as ReturnType<typeof vi.fn>
  };
}

describe("登录提交协调 createLoginSubmit", () => {
  it("pending 期间重复调用不会再次触发登录逻辑", async () => {
    let resolveLogin: () => void = () => {};
    const doLogin = vi.fn(
      () =>
        new Promise<void>(resolve => {
          resolveLogin = resolve;
        })
    );
    const submit = createLoginSubmit({
      validate: async () => true,
      doLogin,
      onSuccess: vi.fn(),
      onError: vi.fn()
    });
    const first = submit.submit();
    // 等待进入 pending（doLogin 已开始）
    await vi.waitFor(() => expect(doLogin).toHaveBeenCalledTimes(1));
    // pending 中的重复提交被忽略
    const second = await submit.submit();
    expect(second.submitted).toBe(false);
    resolveLogin();
    const firstResult = await first;
    expect(firstResult.submitted).toBe(true);
    expect(doLogin).toHaveBeenCalledTimes(1);
  });

  it("失败后锁释放，可以再次提交", async () => {
    const { submit, doLogin, onError, onSuccess } = buildDeps();
    doLogin.mockImplementationOnce(() => Promise.reject(new Error("bad")));
    const first = await submit.submit();
    expect(first.submitted).toBe(true);
    expect(onError).toHaveBeenCalledTimes(1);
    expect(onSuccess).not.toHaveBeenCalled();
    // 再次提交成功
    const second = await submit.submit();
    expect(second.submitted).toBe(true);
    expect(onSuccess).toHaveBeenCalledTimes(1);
    expect(doLogin).toHaveBeenCalledTimes(2);
  });

  it("成功分支只触发一次消息", async () => {
    const { submit, onSuccess, onError } = buildDeps();
    await submit.submit();
    expect(onSuccess).toHaveBeenCalledTimes(1);
    expect(onError).not.toHaveBeenCalled();
  });

  it("失败分支只触发一次消息", async () => {
    const { submit, onError, onSuccess } = buildDeps({
      doLogin: async () => {
        throw new Error("bad");
      }
    });
    await submit.submit();
    expect(onError).toHaveBeenCalledTimes(1);
    expect(onSuccess).not.toHaveBeenCalled();
  });

  it("校验失败不发起登录请求且锁释放", async () => {
    const { submit, doLogin, validate } = buildDeps();
    validate.mockResolvedValueOnce(false);
    const result = await submit.submit();
    expect(result.submitted).toBe(false);
    expect(doLogin).not.toHaveBeenCalled();
    // 锁已释放；下次校验通过后可以正常提交
    const retry = await submit.submit();
    expect(retry.submitted).toBe(true);
    expect(doLogin).toHaveBeenCalledTimes(1);
  });

  it("校验函数抛异常按校验失败处理，不发起登录", async () => {
    const { submit, doLogin } = buildDeps({
      validate: () => Promise.reject(new Error("validate crash"))
    });
    const result = await submit.submit();
    expect(result.submitted).toBe(false);
    expect(doLogin).not.toHaveBeenCalled();
  });
});
