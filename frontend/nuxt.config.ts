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
    '/api/**': { proxy: `${apiProxyTarget}/api/**` }
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
  }
})
