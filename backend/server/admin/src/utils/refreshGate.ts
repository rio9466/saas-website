/**
 * 单飞刷新门卫：同一时刻只允许一个刷新请求在途，
 * 其余并发 401 请求复用同一个刷新结果。
 */
export class RefreshGate {
  private inflight: Promise<string> | null = null;

  constructor(private readonly refresh: () => Promise<string>) {}

  /** 获取新 access token。若已有刷新在途则复用其结果（单飞）。 */
  acquire(): Promise<string> {
    if (!this.inflight) {
      this.inflight = this.refresh().finally(() => {
        this.inflight = null;
      });
    }
    return this.inflight;
  }

  /** 是否有刷新在途（用于测试与诊断） */
  get isInflight(): boolean {
    return this.inflight !== null;
  }
}

/**
 * 401 后是否触发刷新 / 清理。
 * - 白名单（login/refresh）：直接跳过，永不触发刷新循环。
 * - 已重试过一次仍然 401：清理认证状态，不再重试。
 * - 其他：单飞刷新后重试原请求一次。
 */
export type RefreshDecision = "skip" | "refresh" | "clear";

export function decideAfter401(
  whitelisted: boolean,
  alreadyRetried: boolean
): RefreshDecision {
  if (whitelisted) return "skip";
  if (alreadyRetried) return "clear";
  return "refresh";
}
