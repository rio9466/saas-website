import type { NextConfig } from "next";
import createNextIntlPlugin from "next-intl/plugin";

// `i18n/request.ts` loads the locale + messages for every server render.
const withNextIntl = createNextIntlPlugin("./i18n/request.ts");

const nextConfig: NextConfig = {
  // The browser only ever talks to the same-origin `/api/**` route handler,
  // which reads API_PROXY_TARGET per request at runtime (contract §7.1).
  // Never add build-time `rewrites` here: a rewrite target is frozen into the
  // build output and could not be changed without rebuilding.
};

export default withNextIntl(nextConfig);
