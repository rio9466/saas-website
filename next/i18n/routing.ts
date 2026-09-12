import { defineRouting } from "next-intl/routing";

/**
 * URL strategy (contract §7.3): the default language has no prefix, every
 * other language lives under `/<code>/...`. The set of routable locales has to
 * be known at build time, so it is fixed here; `/api/v1/public/settings` may
 * filter which ones are actually offered at runtime.
 *
 * The choice is persisted in the `NEXT_LOCALE` cookie, which next-intl reads
 * during middleware detection, so a reload or a new visit keeps the language.
 */
export const routing = defineRouting({
  locales: ["en", "zh-CN"],
  defaultLocale: "en",
  localePrefix: "as-needed",
  localeCookie: {
    name: "NEXT_LOCALE",
    sameSite: "lax",
    maxAge: 60 * 60 * 24 * 365,
  },
});

export type AppLocale = (typeof routing.locales)[number];

export function isAppLocale(value: unknown): value is AppLocale {
  return (
    typeof value === "string" &&
    (routing.locales as readonly string[]).includes(value)
  );
}
