/**
 * Applies the persisted user preferences on startup.
 *
 * The colour-mode plugin reads the `nuxt-color-mode` cookie with a blocking
 * head script, so the first paint already has the right theme; mirroring the
 * store here also covers the case where that cookie was cleared.
 *
 * Likewise, the i18n module persists `i18n_redirected` for 365 days and SSR
 * uses it directly. The store is the fallback: if that cookie is missing, the
 * language is restored once the app is ready.
 */
export default defineNuxtPlugin((nuxtApp) => {
  const prefs = usePreferencesStore()

  const colorMode = useColorMode()
  if (prefs.colorMode) {
    colorMode.preference = prefs.colorMode
  }

  onNuxtReady(() => {
    nuxtApp.runWithContext(() => {
      const { locale, setLocale } = useI18n()
      // The store keeps a plain string (the locale set comes from the backend);
      // narrow it to the build-time locale union here.
      type LocaleCode = Parameters<typeof setLocale>[0]
      if (prefs.locale && prefs.locale !== locale.value) {
        setLocale(prefs.locale as LocaleCode)
      }
    })
  })
})
