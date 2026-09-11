// https://nuxt.com/docs/api/configuration/nuxt-config
// `types: []` in the generated node tsconfig means `process` is untyped here.
const env = (globalThis as unknown as { process?: { env?: Record<string, string | undefined> } }).process?.env ?? {}
const apiProxyTarget = env.NUXT_API_PROXY_TARGET || 'http://127.0.0.1:8080'
const apiInternalBase = env.NUXT_API_INTERNAL_BASE || 'http://127.0.0.1:8080'

export default defineNuxtConfig({
  modules: [
    '@nuxt/eslint',
    '@nuxt/ui',
    '@nuxtjs/i18n'
  ],

  devtools: {
    enabled: true
  },

  app: {
    head: {
      meta: [
        { name: 'viewport', content: 'width=device-width, initial-scale=1' }
      ],
      link: [
        { rel: 'icon', href: '/favicon.ico' }
      ]
    }
  },

  css: ['~/assets/css/main.css'],

  runtimeConfig: {
    apiProxyTarget,
    apiInternalBase,
    public: {
      apiBase: '/api'
    }
  },

  routeRules: {
    // Browser traffic always stays same-origin; Nitro proxies /api/** to the Go service.
    '/api/**': { proxy: `${apiProxyTarget}/api/**` },
    // Public content pages are cached (ISR-style) for the same window as the
    // backend `Cache-Control: max-age=60` (contract §7.4).
    '/': { swr: 60 },
    '/features': { swr: 60 },
    '/pricing': { swr: 60 },
    '/contact': { swr: 60 },
    '/docs': { swr: 60 },
    '/docs/**': { swr: 60 }
  },

  compatibilityDate: '2026-06-30',

  eslint: {
    config: {
      stylistic: {
        commaDangle: 'never',
        braceStyle: '1tbs'
      }
    }
  },

  i18n: {
    restructureDir: 'i18n',
    langDir: 'locales',
    strategy: 'prefix_except_default',
    defaultLocale: 'en',
    detectBrowserLanguage: false,
    locales: [
      { code: 'en', language: 'en-US', file: 'en.json' },
      { code: 'zh-CN', language: 'zh-CN', file: 'zh-CN.json' }
    ]
  },

  // Icons must render during SSR in the standalone Node output. The local icon
  // endpoint defaults to /api/_nuxt_icon, which the /api/** proxy above would
  // forward to the Go backend (breaking SSR icons), so move it off /api/**.
  icon: {
    localApiEndpoint: '/_nuxt_icon',
    mode: 'svg',
    clientBundle: {
      scan: true,
      sizeLimitKb: 512
    }
  }
})
