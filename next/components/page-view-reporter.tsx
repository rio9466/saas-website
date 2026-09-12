"use client";

import { useEffect, useRef } from "react";
import { usePathname } from "next/navigation";
import { useLocale } from "next-intl";

/**
 * Reports one page view per client-side route change to
 * `POST /api/v1/public/page-view` (contract §4.10).
 *
 * Fire-and-forget: `navigator.sendBeacon` is preferred (it survives an
 * immediate navigation) with a `keepalive` fetch fallback. Failures are always
 * silent — statistics must never break navigation. Private areas (`/account`)
 * are skipped.
 */

const ENDPOINT = "/api/v1/public/page-view";
const IGNORED_PREFIXES = ["/account"];

export function PageViewReporter() {
  const pathname = usePathname();
  const locale = useLocale();
  const lastReported = useRef<string | null>(null);

  useEffect(() => {
    if (!pathname || lastReported.current === pathname) {
      return;
    }
    lastReported.current = pathname;

    // Strip the non-default locale prefix before matching private areas.
    const prefix = `/${locale}`;
    const basePath =
      pathname === prefix
        ? "/"
        : pathname.startsWith(`${prefix}/`)
          ? pathname.slice(prefix.length)
          : pathname;
    if (
      IGNORED_PREFIXES.some(
        (path) => basePath === path || basePath.startsWith(`${path}/`),
      )
    ) {
      return;
    }

    const body = JSON.stringify({
      path: pathname,
      referrer: document.referrer,
      locale,
    });

    try {
      if (typeof navigator.sendBeacon === "function") {
        const blob = new Blob([body], { type: "application/json" });
        if (navigator.sendBeacon(ENDPOINT, blob)) {
          return;
        }
      }
    } catch {
      // Beacon failed synchronously; fall through to the fetch fallback.
    }

    fetch(ENDPOINT, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body,
      keepalive: true,
    }).catch(() => {
      // Statistics only: failures are silent.
    });
  }, [pathname, locale]);

  return null;
}
