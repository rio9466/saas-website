// Maps backend business codes (docs/api/frontend-api-contract.md §3) to i18n
// message keys. Never surface the backend `message` field to users.
export class ApiError extends Error {
  readonly status: number
  readonly code: number
  readonly requestId: string

  constructor(status: number, code: number, message: string, requestId = '') {
    super(message)
    this.name = 'ApiError'
    this.status = status
    this.code = code
    this.requestId = requestId
  }
}

const KNOWN_ERROR_CODES = new Set<number>([
  10001,
  20001,
  20002,
  20003,
  30001,
  40001,
  40002,
  40005,
  40006,
  40010,
  40011,
  40012,
  40013,
  40016,
  42901,
  50001,
  50002,
  50003
])

export function apiErrorKey(code?: number | null): string {
  if (typeof code === 'number' && KNOWN_ERROR_CODES.has(code)) {
    return `errorCodes.${code}`
  }
  return 'errorCodes.unknown'
}
