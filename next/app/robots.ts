import type { MetadataRoute } from "next";
import { routing } from "@/i18n/routing";
import { siteOrigin } from "@/lib/seo";
import { fetchSiteSettings } from "@/lib/site-settings";

/** SEO baseline; points crawlers at the language-aware sitemap. */
export default async function robots(): Promise<MetadataRoute.Robots> {
  const origin = siteOrigin(await fetchSiteSettings(routing.defaultLocale));

  return {
    rules: [
      {
        userAgent: "*",
        allow: "/",
        disallow: ["/api/"],
      },
    ],
    sitemap: `${origin}/sitemap.xml`,
  };
}
