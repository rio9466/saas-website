/**
 * Dynamic sitemap.xml (task Scope item 8).
 *
 * Chosen over `@nuxtjs/sitemap` because that module resolves `sitemap.urls`
 * at build time, which would freeze published pages/articles unless the backend
 * were reachable during the build. A Nitro route generates the sitemap per
 * request from the live `/public/pages` + `/public/docs` list, so newly
 * published content and every enabled locale are always included.
 */
interface PublicSettings {
  public_frontend_url: string
  default_locale: string
  locales: Array<{ code: string }>
}

interface PublicPages {
  items: Array<{ slug: string }>
}

interface PublicDocs {
  categories: Array<{ articles: Array<{ slug: string }> }>
}

const FIXED_ROUTES = ['/', '/features', '/pricing', '/docs', '/contact']

export default defineEventHandler(async (event) => {
  const config = useRuntimeConfig(event)
  const requestUrl = getRequestURL(event)
  const apiBase = `${config.apiInternalBase.replace(/\/$/, '')}${config.public.apiBase}`

  async function fetchPublic<T>(path: string, locale?: string): Promise<T | null> {
    try {
      const envelope = await $fetch<{ code: number, data: T }>(`${apiBase}${path}`, {
        query: locale ? { locale } : undefined
      })
      return envelope.code === 0 ? envelope.data : null
    } catch {
      return null
    }
  }

  const settings = await fetchPublic<PublicSettings>('/v1/public/settings')
  const locales = settings?.locales?.map(entry => entry.code) ?? []
  const defaultLocale = settings?.default_locale || locales[0] || 'en'
  const activeLocales = locales.length ? locales : [defaultLocale]
  const origin = (settings?.public_frontend_url || requestUrl.origin).replace(/\/$/, '')

  const pageSlugs = new Set<string>()
  const docSlugs = new Set<string>()
  for (const locale of activeLocales) {
    const pages = await fetchPublic<PublicPages>('/v1/public/pages', locale)
    pages?.items?.forEach((item) => {
      if (item.slug) {
        pageSlugs.add(item.slug)
      }
    })

    const docs = await fetchPublic<PublicDocs>('/v1/public/docs', locale)
    docs?.categories?.forEach((category) => {
      category.articles?.forEach((article) => {
        if (article.slug) {
          docSlugs.add(article.slug)
        }
      })
    })
  }

  function localePath(locale: string, path: string): string {
    const normalized = path.startsWith('/') ? path : `/${path}`
    if (locale === defaultLocale) {
      return normalized
    }
    return normalized === '/' ? `/${locale}` : `/${locale}${normalized}`
  }

  const urls = new Map<string, { loc: string, alternatives: Array<{ hreflang: string, href: string }> }>()

  function addPath(path: string) {
    const alternatives = activeLocales.map(locale => ({
      hreflang: locale,
      href: `${origin}${localePath(locale, path)}`
    }))
    alternatives.push({ hreflang: 'x-default', href: `${origin}${localePath(defaultLocale, path)}` })

    for (const locale of activeLocales) {
      const loc = `${origin}${localePath(locale, path)}`
      urls.set(loc, { loc, alternatives })
    }
  }

  FIXED_ROUTES.forEach(addPath)
  ;[...pageSlugs].forEach(slug => addPath(`/${encodeURIComponent(slug)}`))
  ;[...docSlugs].forEach(slug => addPath(`/docs/${encodeURIComponent(slug)}`))

  const escapeXml = (value: string): string => value.replace(/[<>&'"]/g, char => ({
    '<': '&lt;',
    '>': '&gt;',
    '&': '&amp;',
    '\'': '&apos;',
    '"': '&quot;'
  }[char] as string))

  const body = [...urls.values()].map((entry) => {
    const links = entry.alternatives
      .map(alt => `    <xhtml:link rel="alternate" hreflang="${escapeXml(alt.hreflang)}" href="${escapeXml(alt.href)}"/>`)
      .join('\n')
    return `  <url>\n    <loc>${escapeXml(entry.loc)}</loc>\n${links}\n  </url>`
  }).join('\n')

  const xml = `<?xml version="1.0" encoding="UTF-8"?>\n`
    + `<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9" xmlns:xhtml="http://www.w3.org/1999/xhtml">\n`
    + `${body}\n`
    + `</urlset>`

  setHeader(event, 'Content-Type', 'application/xml; charset=utf-8')
  setHeader(event, 'Cache-Control', 'public, max-age=600')
  return xml
})
