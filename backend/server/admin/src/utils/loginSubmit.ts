/**
 * 登录提交协调逻辑（纯函数，便于单测）。
 *
 * 约束：
 * - 在第一次异步等待之前同步加锁，同一时刻只允许一次提交；
 * - form 校验失败不产生请求，且锁必须释放；
 * - 成功/失败分支各自只触发一次回调（一次消息）；
 * - 失败后锁释放，允许用户修改后再次提交；
 * - 不做消息合并或“吞请求”等掩盖处理。
 */
export interface LoginSubmitDeps {
  /** 表单校验（返回是否通过） */
  validate: () => Promise<boolean>;
  /** 提交动作：登录请求 + /me + 路由初始化 + 跳转 */
  doLogin: () => Promise<void>;
  /** 成功分支（只调用一次） */
  onSuccess: () => void;
  /** 失败分支（只调用一次） */
  onError: (error: unknown) => void;
}

export type LoginSubmitResult = {
  /** 是否真正执行了一次提交 */
  submitted: boolean;
};

export function createLoginSubmit(deps: LoginSubmitDeps) {
  let submitting = false;

  return {
    get isSubmitting() {
      return submitting;
    },

    /**
     * 执行一次提交。pending 期间的重复调用被直接忽略（不重复请求）。
     * 校验失败 / 登录失败都会在 finally 中释放锁。
     */
    async submit(): Promise<LoginSubmitResult> {
      if (submitting) return { submitted: false };

      // 第一次异步等待之前同步加锁
      submitting = true;
      try {
        let valid = false;
        try {
          valid = await deps.validate();
        } catch {
          valid = false;
        }
        if (!valid) return { submitted: false };

        try {
          await deps.doLogin();
          deps.onSuccess();
        } catch (error) {
          deps.onError(error);
        }
        return { submitted: true };
      } finally {
        submitting = false;
      }
    }
  };
}
