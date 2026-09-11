// https://nuxt.com/docs/api/configuration/nuxt-config
// `types: []` in the generated node tsconfig means `process` is untyped here.
const env = (globalThis as unknown as { process?: { env?: Record<string, string | undefined> } }).process?.env ?? {}
const apiProxyTarget = env.NUXT_API_PROXY_TARGET || 'http://127.0.0.1:8080'
const apiInternalBase = env.NUXT_API_INTERNAL_BASE || 'http://127.0.0.1:8080'

export default defineNuxtConfig({
  modules: [
    '@nuxt/eslint',
    '@nuxt/ui',
    '@nuxtjs/i18n',
    '@pinia/nuxt',
    'pinia-plugin-persistedstate/nuxt'
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

  // Persist the colour mode in a cookie so SSR can render the correct theme on
  // the first paint (no light/dark flash). `app/stores/preferences.ts` records
  // the same choice for the startup plugin.
  colorMode: {
    preference: 'system',
    fallback: 'light',
    storage: 'cookie',
    storageKey: 'nuxt-color-mode',
    classSuffix: ''
  },

  runtimeConfig: {
    apiProxyTarget,
    apiInternalBase,
    public: {
      apiBase: '/api'
    }
  },

  routeRules: {
    // Browser traffic always stays same-origin. `/api/**` is proxied to the Go
    // service by `server/middleware/api-proxy.ts`, which reads runtimeConfig
    // per request. A `routeRules` proxy target is frozen into `.output` at build
    // time, so `NUXT_API_PROXY_TARGET` could not take effect at runtime without
    // a rebuild; the middleware keeps dev and production behaviour identical.

    // Public content pages are cached (ISR-style) for the same window as the
    // backend `Cache-Control: max-age=60` (contract §7.4). The root is exempt:
    // the SWR handler strips request headers from the SSR render event, so the
    // i18n cookie/Accept-Language detection cannot run there and a cached root
    // response would leak one visitor's language (and `Set-Cookie`) to others.
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

  // The app uses system fonts; disable the remote providers so the offline
  // environment never tries to reach Google Fonts (or any other provider).
  fonts: {
    providers: {
      adobe: false,
      google: false,
      googleicons: false,
      bunny: false,
      fontshare: false,
      fontsource: false
    }
  },

  i18n: {
    restructureDir: 'i18n',
    langDir: 'locales',
    strategy: 'prefix_except_default',
    defaultLocale: 'en',
    // Remember the visitor's language between visits. The module reads this
    // cookie during SSR, so a returning visitor to `/` is rendered (or
    // redirected) in their language without a client-side flash. Prefixed URLs
    // such as `/zh-CN/...` still win because detection only runs on `/`.
    detectBrowserLanguage: {
      useCookie: true,
      cookieKey: 'i18n_redirected',
      redirectOn: 'root'
    },
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
  },

  // User preferences (locale, colour mode) are persisted to `saas-preferences`
  // by `app/stores/preferences.ts`. `stores/` is relative to the Nuxt 4 app
  // directory (`app/`), so the glob is `stores/**`, not `app/stores/**`.
  pinia: {
    storesDirs: ['stores/**']
  }
})
