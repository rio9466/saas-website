/**
 * 密码强度规则：UTF-8 字节长度 8-72（ADM-003）。
 *
 * 必须按字节计算（TextEncoder），不能依赖 Element Plus 的 min/max 字符计数，
 * 也不能依赖输入框 maxlength（按 UTF-16 code unit 计数）。后端以字节为权威。
 */
export const MIN_PASSWORD_BYTES = 8;
export const MAX_PASSWORD_BYTES = 72;

/** UTF-8 字节长度 */
export function utf8ByteLength(value: string): number {
  return new TextEncoder().encode(value ?? "").length;
}

export const PASSWORD_PLACEHOLDER = "8-72 字节";

export function passwordByteError(value: string): string | null {
  const n = utf8ByteLength(value);
  if (n < MIN_PASSWORD_BYTES) return `密码至少 ${MIN_PASSWORD_BYTES} 字节`;
  if (n > MAX_PASSWORD_BYTES) return `密码最多 ${MAX_PASSWORD_BYTES} 字节`;
  return null;
}

/** Element Plus 表单校验器（按 UTF-8 字节数检查长度） */
export function createPasswordByteValidator(
  trigger: "blur" | "change" = "blur"
) {
  return {
    validator: (
      _rule: unknown,
      value: string,
      callback: (error?: Error) => void
    ) => {
      const err = passwordByteError(value ?? "");
      callback(err ? new Error(err) : undefined);
    },
    trigger
  };
}
