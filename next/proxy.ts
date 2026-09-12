import createMiddleware from "next-intl/middleware";
import { routing } from "@/i18n/routing";

/**
 * Locale routing. Next.js 16 renamed Middleware to Proxy (`proxy.ts`,
 * Node runtime); the handler itself is next-intl's locale middleware.
 *
 * It rewrites/redirects so the default locale has no prefix (`/`) and other
 * locales do (`/zh-CN/...`), detects the language from the `NEXT_LOCALE`
 * cookie, sets `Link` alternate headers, and lets `/api`, `/_next` and files
 * (anything with a dot) pass through untouched — in particular the same-origin
 * API route handler must never be locale-rewritten.
 */
export default createMiddleware(routing);

export const config = {
  matcher: ["/((?!api|_next|_vercel|.*\\..*).*)"],
};
