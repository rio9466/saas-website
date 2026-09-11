export type ColorModePreference = '' | 'light' | 'dark' | 'system'

/** One year, in seconds. */
const PREFERENCES_COOKIE_MAX_AGE = 60 * 60 * 24 * 365

export interface PreferencesState {
  locale: string
  colorMode: ColorModePreference
}

/**
 * User preferences shared by the locale switcher / theme toggle and applied on
 * startup by `app/plugins/preferences.ts`. Persisted to the `saas-preferences`
 * cookie (`piniaPluginPersistedstate` is provided by the
 * `pinia-plugin-persistedstate/nuxt` module), so SSR can read it.
 *
 * The i18n / colour-mode modules keep their own durable cookies
 * (`i18n_redirected`, `nuxt-color-mode`); this store is a second record that the
 * startup plugin can use if either of those is missing.
 */
export const usePreferencesStore = defineStore('preferences', {
  state: (): PreferencesState => ({
    locale: '',
    colorMode: ''
  }),
  persist: {
    key: 'saas-preferences',
    storage: piniaPluginPersistedstate.cookies({
      maxAge: PREFERENCES_COOKIE_MAX_AGE,
      sameSite: 'lax'
    })
  }
})
