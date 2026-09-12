import type { Metadata } from "next";
import { getTranslations } from "next-intl/server";
import { routing } from "@/i18n/routing";
import { getSiteSettings, type SiteSettings } from "./site-settings";

/** Localized path helper (default locale has no prefix, others do). */
export function localePath(locale: string, path: string): string {
  const clean = path === "/" ? "" : path.startsWith("/") ? path : `/${path}`;
  if (locale === routing.defaultLocale) {
    return clean === "" ? "/" : clean;
  }
  return `/${locale}${clean}`;
}

/**
 * Public site origin for canonical URLs, Open Graph and the sitemap. Prefers an
 * explicit `NEXT_PUBLIC_SITE_URL`; in production falls back to the backend's
 * configured `public_frontend_url`, and in dev uses the local template port.
 */
export function siteOrigin(settings?: SiteSettings | null): string {
  return (
    process.env.NEXT_PUBLIC_SITE_URL ||
    (process.env.NODE_ENV === "production"
      ? settings?.public_frontend_url || ""
      : "") ||
    "http://localhost:3200"
  );
}

interface BuildMetadataInput {
  locale: string;
  settings: SiteSettings;
  path?: string;
  /** Page-specific title; falls back to the site default when omitted. */
  title?: string;
  /** Page-specific description; falls back to the site default. */
  description?: string;
}

/**
 * Metadata for a page: title/description overrides with site defaults, plus
 * canonical and hreflang alternates (contract §7.3). The `<html lang>`
 * attribute is set in the layout.
 */
export function buildMetadata({
  locale,
  settings,
  path = "/",
  title,
  description,
}: BuildMetadataInput): Metadata {
  const siteTitle = settings.seo_default_title || settings.site_name;
  const resolvedTitle = title ? `${title} · ${settings.site_name}` : siteTitle;
  const resolvedDescription =
    description ||
    settings.seo_default_description ||
    settings.tagline ||
    undefined;

  const origin = siteOrigin(settings);

  const languages: Record<string, string> = Object.fromEntries(
    routing.locales.map((code) => [code, localePath(code, path)]),
  );
  languages["x-default"] = localePath(routing.defaultLocale, path);

  return {
    metadataBase: new URL(origin),
    title: resolvedTitle,
    description: resolvedDescription,
    alternates: {
      canonical: localePath(locale, path),
      languages,
    },
    openGraph: {
      title: resolvedTitle,
      description: resolvedDescription,
      url: localePath(locale, path),
      siteName: settings.site_name,
      images: settings.seo_default_og_image_url
        ? [settings.seo_default_og_image_url]
        : undefined,
    },
    icons: settings.favicon_url ? { icon: settings.favicon_url } : undefined,
  };
}

/**
 * Convenience wrapper for a page's `generateMetadata`: loads the site settings
 * (with translated fallbacks) and builds the metadata for one localized path.
 */
export async function pageMetadata({
  locale,
  path,
  title,
  description,
}: {
  locale: string;
  path: string;
  title?: string;
  description?: string;
}): Promise<Metadata> {
  const t = await getTranslations({ locale });
  const settings = await getSiteSettings(locale, t);
  return buildMetadata({ locale, settings, path, title, description });
}
