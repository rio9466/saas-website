import { useSiteSettings } from './useSiteSettings'

export interface SeoOptions {
  title?: string
  description?: string
  ogImage?: string
  ogType?: string
}

function absoluteUrl(baseUrl: string, path: string): string {
  const base = baseUrl.replace(/\/$/, '')
  const normalized = path.startsWith('/') ? path : `/${path}`
  return `${base}${normalized}`
}

/**
 * Site-wide SEO base (contract §7.3): per-page title/description/og overrides
 * with site defaults, plus `<html lang>`, canonical and hreflang alternates
 * (including `x-default`).
 */
export function useSeo(options: SeoOptions = {}) {
  const { t, locale } = useI18n()
  const { settings, enabledLocales, defaultLocale } = useSiteSettings()
  const switchLocalePath = useSwitchLocalePath()
  const route = useRoute()
  const requestUrl = useRequestURL()
  type LocaleCode = Parameters<typeof switchLocalePath>[0]

  useHead(() => {
    const site = settings.value
    const baseUrl = site.public_frontend_url || requestUrl.origin
    const title = options.title
      ? `${options.title} · ${site.site_name}`
      : (site.seo_default_title || t('seo.defaultTitle'))
    const description = options.description || site.seo_default_description || t('seo.defaultDescription')
    const canonical = absoluteUrl(baseUrl, route.path)
    const ogImage = options.ogImage || site.seo_default_og_image_url

    const alternates = enabledLocales.value.map(entry => ({
      key: `hreflang:${entry.code}`,
      rel: 'alternate' as const,
      hreflang: entry.code,
      href: absoluteUrl(baseUrl, switchLocalePath(entry.code as LocaleCode) || '/')
    }))
    alternates.push({
      key: 'hreflang:x-default',
      rel: 'alternate' as const,
      hreflang: 'x-default',
      href: absoluteUrl(baseUrl, switchLocalePath(defaultLocale.value as LocaleCode) || '/')
    })

    const meta = [
      { key: 'description', name: 'description', content: description },
      { key: 'og:site_name', property: 'og:site_name', content: site.site_name },
      { key: 'og:title', property: 'og:title', content: title },
      { key: 'og:description', property: 'og:description', content: description },
      { key: 'og:type', property: 'og:type', content: options.ogType || 'website' },
      { key: 'og:url', property: 'og:url', content: canonical },
      { key: 'og:locale', property: 'og:locale', content: locale.value },
      { key: 'twitter:card', name: 'twitter:card', content: 'summary_large_image' }
    ]
    if (ogImage) {
      meta.push({ key: 'og:image', property: 'og:image', content: ogImage })
    }

    return {
      htmlAttrs: { lang: locale.value },
      title,
      link: [
        { key: 'canonical', rel: 'canonical', href: canonical },
        ...alternates
      ],
      meta
    }
  })
}
