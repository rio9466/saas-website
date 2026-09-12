import type { MetadataRoute } from "next";
import { routing } from "@/i18n/routing";
import { getDocs, getPages } from "@/lib/public-content";
import { localePath, siteOrigin } from "@/lib/seo";
import { fetchSiteSettings } from "@/lib/site-settings";

/** Routes that exist independently of the backend content (contract §4.9). */
const FIXED_PATHS = ["/", "/features", "/pricing", "/docs", "/contact"];

interface SitemapPath {
  path: string;
  lastModified?: string;
}

/**
 * Language-aware sitemap: fixed routes plus every published page and
 * documentation article from the public content API. Unavailable content is
 * skipped so the sitemap still renders when the backend is down.
 */
export default async function sitemap(): Promise<MetadataRoute.Sitemap> {
  const origin = siteOrigin(await fetchSiteSettings(routing.defaultLocale));
  const absolute = (path: string) => new URL(path, origin).toString();

  const entries: MetadataRoute.Sitemap = [];
  const seen = new Set<string>();

  for (const locale of routing.locales) {
    const [pages, docs] = await Promise.all([
      getPages(locale),
      getDocs(locale),
    ]);

    const paths: SitemapPath[] = FIXED_PATHS.map((path) => ({ path }));
    for (const page of pages) {
      paths.push({ path: `/${page.slug}`, lastModified: page.updated_at });
    }
    for (const category of docs) {
      for (const article of category.articles) {
        paths.push({
          path: `/docs/${article.slug}`,
          lastModified: article.updated_at,
        });
      }
    }

    for (const { path, lastModified } of paths) {
      const url = absolute(localePath(locale, path));
      if (seen.has(url)) {
        continue;
      }
      seen.add(url);
      entries.push({
        url,
        lastModified: lastModified || undefined,
        alternates: {
          languages: Object.fromEntries(
            routing.locales.map((code) => [
              code,
              absolute(localePath(code, path)),
            ]),
          ),
        },
      });
    }
  }

  return entries;
}
