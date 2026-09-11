import { useApi } from './useApi'

/**
 * Typed access to the public content endpoints (contract §4). Every request
 * goes through `useApi()` and carries the active `?locale=`, so the pages never
 * talk to the backend directly.
 */

export interface HomeSection {
  id: string
  type: string
  sort_order: number
  data: Record<string, unknown>
}

export interface HomeContent {
  locale: string
  sections: HomeSection[]
}

export interface FeatureItem {
  id: string
  icon: string
  title: string
  summary: string
  body_md: string
  image_url: string
  sort_order: number
}

export interface FeaturesContent {
  locale: string
  items: FeatureItem[]
}

export interface PricingPlan {
  id: string
  code: string
  name: string
  description: string
  monthly_price: string
  yearly_price: string
  currency: string
  highlighted: boolean
  cta_label: string
  cta_url: string
  features: string[]
  sort_order: number
}

export interface PricingContent {
  locale: string
  currency: string
  plans: PricingPlan[]
}

export interface PageSummary {
  slug: string
  title: string
  updated_at: string
}

export interface PagesContent {
  locale: string
  items: PageSummary[]
}

export interface PageContent extends PageSummary {
  body_md: string
  seo_title: string
  seo_description: string
}

export interface DocArticleSummary {
  id: string
  slug: string
  title: string
  sort_order: number
  updated_at: string
}

export interface DocCategory {
  id: string
  slug: string
  name: string
  sort_order: number
  articles: DocArticleSummary[]
}

export interface DocsContent {
  locale: string
  categories: DocCategory[]
}

export interface DocRef {
  slug: string
  title: string
}

export interface DocArticle {
  locale: string
  id: string
  slug: string
  title: string
  body_md: string
  category: { id: string, slug: string, name: string }
  prev: DocRef | null
  next: DocRef | null
  seo_title: string
  seo_description: string
  updated_at: string
}

function useContentLocale() {
  const { locale } = useI18n()
  return locale
}

/** Published home sections; unknown section types are ignored by the page. */
export function useHomeContent() {
  const locale = useContentLocale()
  const api = useApi()
  return useAsyncData<HomeContent>(
    'public-home',
    () => api.get<HomeContent>('/v1/public/home', { locale: locale.value, query: { locale: locale.value } }),
    {
      default: () => ({ locale: locale.value, sections: [] }),
      watch: [locale]
    }
  )
}

/** Published feature cards. */
export function useFeaturesContent() {
  const locale = useContentLocale()
  const api = useApi()
  return useAsyncData<FeaturesContent>(
    'public-features',
    () => api.get<FeaturesContent>('/v1/public/features', { locale: locale.value, query: { locale: locale.value } }),
    {
      default: () => ({ locale: locale.value, items: [] }),
      watch: [locale]
    }
  )
}

/** Visible pricing plans. */
export function usePricingContent() {
  const locale = useContentLocale()
  const api = useApi()
  return useAsyncData<PricingContent>(
    'public-pricing',
    () => api.get<PricingContent>('/v1/public/pricing', { locale: locale.value, query: { locale: locale.value } }),
    {
      default: () => ({ locale: locale.value, currency: '', plans: [] }),
      watch: [locale]
    }
  )
}

/** Published page summaries, used for route discovery and the sitemap. */
export function usePagesContent() {
  const locale = useContentLocale()
  const api = useApi()
  return useAsyncData<PagesContent>(
    'public-pages',
    () => api.get<PagesContent>('/v1/public/pages', { locale: locale.value, query: { locale: locale.value } }),
    {
      default: () => ({ locale: locale.value, items: [] }),
      watch: [locale]
    }
  )
}

/** One published page. A missing/unpublished slug rejects with `40002`/404. */
export function usePageContent(slug: MaybeRefOrGetter<string>) {
  const locale = useContentLocale()
  const api = useApi()
  const slugRef = toRef(slug)
  return useAsyncData<PageContent | null>(
    () => `public-page:${slugRef.value}`,
    () => slugRef.value
      ? api.get<PageContent>(`/v1/public/pages/${encodeURIComponent(slugRef.value)}`, {
          locale: locale.value,
          query: { locale: locale.value }
        })
      : Promise.resolve(null),
    {
      default: () => null,
      watch: [locale, slugRef]
    }
  )
}

/** Documentation tree (categories with their articles). */
export function useDocsContent() {
  const locale = useContentLocale()
  const api = useApi()
  return useAsyncData<DocsContent>(
    'public-docs',
    () => api.get<DocsContent>('/v1/public/docs', { locale: locale.value, query: { locale: locale.value } }),
    {
      default: () => ({ locale: locale.value, categories: [] }),
      watch: [locale]
    }
  )
}

/** One published documentation article. A missing slug rejects with `40002`/404. */
export function useDocContent(slug: MaybeRefOrGetter<string>) {
  const locale = useContentLocale()
  const api = useApi()
  const slugRef = toRef(slug)
  return useAsyncData<DocArticle | null>(
    () => `public-doc:${slugRef.value}`,
    () => slugRef.value
      ? api.get<DocArticle>(`/v1/public/docs/${encodeURIComponent(slugRef.value)}`, {
          locale: locale.value,
          query: { locale: locale.value }
        })
      : Promise.resolve(null),
    {
      default: () => null,
      watch: [locale, slugRef]
    }
  )
}

/** First article in the documentation tree, used by the docs landing page. */
export function firstDocArticle(categories: DocCategory[]): DocArticleSummary | null {
  for (const category of categories) {
    if (category.articles?.length) {
      return category.articles[0] ?? null
    }
  }
  return null
}

/** True when a content request failed because the slug is missing/unpublished. */
export function isContentNotFound(error: unknown): boolean {
  if (error instanceof ApiError) {
    return error.status === 404 || error.code === 40002
  }
  // `useAsyncData` wraps thrown errors in a NuxtError, which drops the ApiError
  // class but keeps the HTTP status / business code.
  if (error && typeof error === 'object') {
    const wrapped = error as { statusCode?: number, status?: number, code?: number, data?: { code?: number } }
    return wrapped.statusCode === 404
      || wrapped.status === 404
      || wrapped.code === 40002
      || wrapped.data?.code === 40002
  }
  return false
}
