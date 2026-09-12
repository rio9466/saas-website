/**
 * Contract §3 business codes that trigger a single refresh + replay (contract §2).
 */
export const RETRYABLE_UNAUTHORIZED_CODES = new Set([20001, 20002, 20003, 40006]);

const KNOWN_ERROR_CODES = new Set<number>([
  10001, 20001, 20002, 20003, 30001, 40001, 40002, 40005, 40006, 40010, 40011,
  40012, 40013, 40016, 42901, 50001, 50002, 50003,
]);

/**
 * Envelope-aware API failure. `message` is the backend's developer-facing
 * English text and must never be shown to users; map `code` to i18n instead.
 */
export class ApiError extends Error {
  readonly status: number;
  readonly code: number;
  readonly requestId: string;

  constructor(status: number, code: number, message: string, requestId = "") {
    super(message);
    this.name = "ApiError";
    this.status = status;
    this.code = code;
    this.requestId = requestId;
  }
}

/** i18n key for a backend code; falls back to a generic message (contract §3). */
export function apiErrorKey(code?: number | null): string {
  if (typeof code === "number" && KNOWN_ERROR_CODES.has(code)) {
    return `errorCodes.${code}`;
  }
  return "errorCodes.unknown";
}
