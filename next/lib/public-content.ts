/**
 * Typed server-side access to the public content endpoints (contract §4).
 *
 * SSR talks to the backend directly through `API_INTERNAL_BASE` (never through
 * the browser proxy) and caches every response for the contract's 60s window.
 * Any failure resolves to an empty/null value so a page can still render its
 * fallback shell when the backend is unavailable (same approach as
 * `lib/site-settings.ts`).
 */

const DEFAULT_INTERNAL_BASE = "http://127.0.0.1:8080";

export interface HomeSection {
  id: string;
  type: string;
  sort_order: number;
  data: Record<string, unknown>;
}

export interface FeatureItem {
  id: string;
  icon: string;
  title: string;
  summary: string;
  body_md: string;
  image_url: string;
  sort_order: number;
}

export interface PricingPlan {
  id: string;
  code: string;
  name: string;
  description: string;
  monthly_price: string;
  yearly_price: string;
  currency: string;
  highlighted: boolean;
  cta_label: string;
  cta_url: string;
  features: string[];
  sort_order: number;
}

export interface PricingContent {
  locale: string;
  currency: string;
  plans: PricingPlan[];
}

export interface PageSummary {
  slug: string;
  title: string;
  updated_at: string;
}

export interface PageContent extends PageSummary {
  locale: string;
  body_md: string;
  seo_title: string;
  seo_description: string;
}

export interface DocArticleSummary {
  id: string;
  slug: string;
  title: string;
  sort_order: number;
  updated_at: string;
}

export interface DocCategory {
  id: string;
  slug: string;
  name: string;
  sort_order: number;
  articles: DocArticleSummary[];
}

export interface DocRef {
  slug: string;
  title: string;
}

export interface DocArticle {
  locale: string;
  id: string;
  slug: string;
  title: string;
  body_md: string;
  category: { id: string; slug: string; name: string };
  prev: DocRef | null;
  next: DocRef | null;
  seo_title: string;
  seo_description: string;
  updated_at: string;
}

async function fetchPublic<T>(
  path: string,
  locale: string,
): Promise<T | null> {
  const base = (
    process.env.API_INTERNAL_BASE || DEFAULT_INTERNAL_BASE
  ).replace(/\/+$/, "");
  const url = `${base}/api${path}${
    path.includes("?") ? "&" : "?"
  }locale=${encodeURIComponent(locale)}`;

  try {
    const response = await fetch(url, {
      headers: { Accept: "application/json", "Accept-Language": locale },
      next: { revalidate: 60 },
    });
    if (!response.ok) {
      return null;
    }
    const envelope = (await response.json()) as { code: number; data?: T };
    if (envelope.code !== 0 || !envelope.data) {
      return null;
    }
    return envelope.data;
  } catch {
    return null;
  }
}

function asArray<T>(value: T[] | undefined | null): T[] {
  return Array.isArray(value) ? value : [];
}

/** Published home sections; unknown section types are ignored by the page. */
export async function getHomeSections(locale: string): Promise<HomeSection[]> {
  const data = await fetchPublic<{ locale: string; sections: HomeSection[] }>(
    "/v1/public/home",
    locale,
  );
  return asArray(data?.sections);
}

/** Published feature cards. */
export async function getFeatures(locale: string): Promise<FeatureItem[]> {
  const data = await fetchPublic<{ locale: string; items: FeatureItem[] }>(
    "/v1/public/features",
    locale,
  );
  return asArray(data?.items);
}

/** Visible pricing plans plus their default currency. */
export async function getPricing(locale: string): Promise<PricingContent> {
  const data = await fetchPublic<PricingContent>("/v1/public/pricing", locale);
  return {
    locale: data?.locale ?? locale,
    currency: data?.currency ?? "",
    plans: asArray(data?.plans),
  };
}

/** Published page summaries, used by the sitemap and route discovery. */
export async function getPages(locale: string): Promise<PageSummary[]> {
  const data = await fetchPublic<{ locale: string; items: PageSummary[] }>(
    "/v1/public/pages",
    locale,
  );
  return asArray(data?.items);
}

/** One published page. A missing/unpublished slug resolves to `null` → 404. */
export async function getPage(
  locale: string,
  slug: string,
): Promise<PageContent | null> {
  if (!slug) {
    return null;
  }
  return fetchPublic<PageContent>(
    `/v1/public/pages/${encodeURIComponent(slug)}`,
    locale,
  );
}

/** Documentation tree (categories with their articles). */
export async function getDocs(locale: string): Promise<DocCategory[]> {
  const data = await fetchPublic<{ locale: string; categories: DocCategory[] }>(
    "/v1/public/docs",
    locale,
  );
  return asArray(data?.categories);
}

/** One published documentation article. A missing slug resolves to `null`. */
export async function getDoc(
  locale: string,
  slug: string,
): Promise<DocArticle | null> {
  if (!slug) {
    return null;
  }
  return fetchPublic<DocArticle>(
    `/v1/public/docs/${encodeURIComponent(slug)}`,
    locale,
  );
}

/** First article in the documentation tree, used by the docs landing page. */
export function firstDocArticle(
  categories: DocCategory[],
): DocArticleSummary | null {
  for (const category of categories) {
    if (category.articles?.length) {
      return category.articles[0] ?? null;
    }
  }
  return null;
}
