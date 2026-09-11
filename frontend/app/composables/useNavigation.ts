import { useApi } from './useApi'

export type NavigationPlacement = 'header' | 'footer'

export interface NavigationItem {
  id: string
  label: string
  url: string
  target: string
  sort_order: number
  children: NavigationItem[]
}

type Translate = (key: string) => string

const FALLBACK_ROUTES: Array<{ key: string, url: string }> = [
  { key: 'nav.home', url: '/' },
  { key: 'nav.features', url: '/features' },
  { key: 'nav.pricing', url: '/pricing' },
  { key: 'nav.about', url: '/about' },
  { key: 'nav.docs', url: '/docs' },
  { key: 'nav.contact', url: '/contact' }
]

function fallbackNavigation(placement: NavigationPlacement, t: Translate): NavigationItem[] {
  const routes = placement === 'footer' ? FALLBACK_ROUTES : FALLBACK_ROUTES.slice(0, 5)
  return routes.map((route, index) => ({
    id: `fallback-${index + 1}`,
    label: t(route.key),
    url: route.url,
    target: '_self',
    sort_order: index + 1,
    children: []
  }))
}

/**
 * Navigation entries from `GET /api/v1/public/navigation` (contract §4.2).
 * Falls back to built-in routes so the header/footer always render.
 */
export function useNavigation(placement: NavigationPlacement = 'header') {
  const { t, locale } = useI18n()
  const api = useApi()

  const { data } = useAsyncData(
    `navigation:${placement}`,
    () => api.get<{ locale: string, items: NavigationItem[] }>('/v1/public/navigation', {
      query: { locale: locale.value, placement }
    }),
    {
      default: () => ({ locale: locale.value, items: fallbackNavigation(placement, t) }),
      watch: [locale]
    }
  )

  return computed<NavigationItem[]>(() => data.value?.items ?? fallbackNavigation(placement, t))
}
