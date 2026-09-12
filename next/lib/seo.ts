import type { Metadata } from "next";
import { routing } from "@/i18n/routing";
import type { SiteSettings } from "./site-settings";

/** Localized path helper (default locale has no prefix, others do). */
export function localePath(locale: string, path: string): string {
  const clean = path === "/" ? "" : path.startsWith("/") ? path : `/${path}`;
  if (locale === routing.defaultLocale) {
    return clean === "" ? "/" : clean;
  }
  return `/${locale}${clean}`;
}

interface BuildMetadataInput {
  locale: string;
  settings: SiteSettings;
  path?: string;
}

/**
 * Base metadata for a page: title/description, canonical and hreflang
 * alternates (contract §7.3). The `<html lang>` attribute is set in the layout.
 */
export function buildMetadata({
  locale,
  settings,
  path = "/",
}: BuildMetadataInput): Metadata {
  const title = settings.seo_default_title || settings.site_name;
  const description =
    settings.seo_default_description || settings.tagline || undefined;

  // Prefer an explicit public origin; in production fall back to the backend's
  // configured `public_frontend_url`, and in dev use the local template port.
  const origin =
    process.env.NEXT_PUBLIC_SITE_URL ||
    (process.env.NODE_ENV === "production"
      ? settings.public_frontend_url
      : "") ||
    "http://localhost:3200";

  const languages = Object.fromEntries(
    routing.locales.map((code) => [code, localePath(code, path)]),
  );

  return {
    metadataBase: new URL(origin),
    title,
    description,
    alternates: {
      canonical: localePath(locale, path),
      languages,
    },
    openGraph: {
      title,
      description,
      url: localePath(locale, path),
      siteName: settings.site_name,
      images: settings.seo_default_og_image_url
        ? [settings.seo_default_og_image_url]
        : undefined,
    },
    icons: settings.favicon_url ? { icon: settings.favicon_url } : undefined,
  };
}
