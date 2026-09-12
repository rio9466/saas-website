import { routing } from "@/i18n/routing";

/** UTF-8 byte length, used for the 8–72 byte password rule (contract §5.1). */
export function byteLength(value: string): number {
  return new TextEncoder().encode(value).length;
}

export const EMAIL_PATTERN = /^[^\s@]+@[^\s@]+\.[^\s@]+$/;

/**
 * Format a fixed-point decimal string with exactly 4 fraction digits without
 * ever going through a float (contract §1.2: points are decimal strings).
 */
export function formatPoints(raw: string): string {
  const value = (raw || "").trim();
  const negative = value.startsWith("-");
  const unsigned = negative ? value.slice(1) : value;
  const [intRaw = "0", fracRaw = ""] = unsigned.split(".");
  const intPart = intRaw.replace(/^0+(?=\d)/, "") || "0";
  let frac = fracRaw.replace(/\D.*$/, "");
  frac = frac.length >= 4 ? frac.slice(0, 4) : frac.padEnd(4, "0");
  return `${negative ? "-" : ""}${intPart}.${frac}`;
}

/** True when a fixed-point decimal string is numerically zero. */
export function isZeroPoints(raw: string): boolean {
  return /^[+-]?0*(\.0*)?$/.test((raw || "").trim());
}

/** Localized date/time; `""` (the contract's null time) renders as an em dash. */
export function formatDateTime(value: string, locale: string): string {
  if (!value) {
    return "—";
  }
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) {
    return value;
  }
  return new Intl.DateTimeFormat(locale, {
    dateStyle: "medium",
    timeStyle: "short",
  }).format(date);
}

/**
 * Only allow same-site absolute paths, preventing open redirects. The task
 * requires the login `?redirect=` to be a site-relative path (contract §2).
 */
export function safeRedirect(value: unknown): string | null {
  if (typeof value !== "string" || !value.startsWith("/")) {
    return null;
  }
  if (value.startsWith("//") || value.includes("\\")) {
    return null;
  }
  return value;
}

/**
 * Drop a leading locale prefix so a path can be passed to next-intl's router,
 * which re-adds the prefix for the current locale (avoiding `/zh-CN/zh-CN/...`).
 */
export function stripLocalePrefix(path: string): string {
  for (const locale of routing.locales) {
    if (path === `/${locale}`) {
      return "/";
    }
    if (path.startsWith(`/${locale}/`)) {
      return path.slice(locale.length + 1);
    }
  }
  return path;
}
