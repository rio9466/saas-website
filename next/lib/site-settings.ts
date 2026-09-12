export interface SiteLocale {
  code: string;
  label: string;
}

export interface SocialLink {
  platform: string;
  url: string;
}

/** Shape of `GET /api/v1/public/settings` (contract §4.1). */
export interface SiteSettings {
  platform_name: string;
  public_frontend_url: string;
  public_api_url: string;
  registration_enabled: boolean;
  username_login_enabled: boolean;
  email_login_enabled: boolean;
  email_verification_required: boolean;
  default_avatar_url: string;
  site_name: string;
  logo_url: string;
  logo_dark_url: string;
  favicon_url: string;
  tagline: string;
  footer_text: string;
  icp_record: string;
  contact_email: string;
  contact_phone: string;
  contact_address: string;
  social_links: SocialLink[];
  seo_default_title: string;
  seo_default_description: string;
  seo_default_og_image_url: string;
  default_locale: string;
  locales: SiteLocale[];
}

type Translate = (key: string) => string;

/** Built-in defaults so the shell still renders when the API is unavailable. */
export function makeFallbackSettings(t: Translate): SiteSettings {
  const name = t("site.fallbackName");
  const tagline = t("site.fallbackTagline");
  return {
    platform_name: name,
    public_frontend_url: "",
    public_api_url: "/api",
    registration_enabled: true,
    username_login_enabled: true,
    email_login_enabled: true,
    email_verification_required: true,
    default_avatar_url: "",
    site_name: name,
    logo_url: "",
    logo_dark_url: "",
    favicon_url: "",
    tagline,
    footer_text: "",
    icp_record: "",
    contact_email: "",
    contact_phone: "",
    contact_address: "",
    social_links: [],
    seo_default_title: name,
    seo_default_description: tagline,
    seo_default_og_image_url: "",
    default_locale: "en",
    locales: [
      { code: "en", label: t("locale.names.en") },
      { code: "zh-CN", label: t("locale.names.zh-CN") },
    ],
  };
}

function normalizeSettings(data: SiteSettings): SiteSettings {
  return {
    ...data,
    social_links: Array.isArray(data.social_links) ? data.social_links : [],
  };
}

/**
 * Server-side fetch of the public site settings (contract §4.1). SSR talks to
 * the backend directly via `API_INTERNAL_BASE`; returns `null` on any failure
 * so the caller can fall back to built-in defaults.
 */
export async function fetchSiteSettings(
  locale: string,
): Promise<SiteSettings | null> {
  const base = (
    process.env.API_INTERNAL_BASE || "http://127.0.0.1:8080"
  ).replace(/\/+$/, "");
  const url = `${base}/api/v1/public/settings?locale=${encodeURIComponent(locale)}`;

  try {
    const response = await fetch(url, {
      headers: { Accept: "application/json", "Accept-Language": locale },
      next: { revalidate: 60 },
    });
    if (!response.ok) {
      return null;
    }
    const envelope = (await response.json()) as {
      code: number;
      data?: SiteSettings;
    };
    if (envelope.code !== 0 || !envelope.data) {
      return null;
    }
    return normalizeSettings(envelope.data);
  } catch {
    return null;
  }
}

/** Settings merged over translated fallbacks; safe to render at all times. */
export async function getSiteSettings(
  locale: string,
  t: Translate,
): Promise<SiteSettings> {
  const fallback = makeFallbackSettings(t);
  const fetched = await fetchSiteSettings(locale);
  if (!fetched) {
    return fallback;
  }
  return {
    ...fallback,
    ...fetched,
    social_links: fetched.social_links ?? fallback.social_links,
    locales: fetched.locales?.length ? fetched.locales : fallback.locales,
  };
}
