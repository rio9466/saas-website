import { useApi } from './useApi'

export interface SiteLocale {
  code: string
  label: string
}

export interface SocialLink {
  platform: string
  url: string
}

export interface SiteSettings {
  platform_name: string
  public_frontend_url: string
  public_api_url: string
  registration_enabled: boolean
  username_login_enabled: boolean
  email_login_enabled: boolean
  email_verification_required: boolean
  default_avatar_url: string
  site_name: string
  logo_url: string
  logo_dark_url: string
  favicon_url: string
  tagline: string
  footer_text: string
  icp_record: string
  contact_email: string
  contact_phone: string
  contact_address: string
  social_links: SocialLink[]
  seo_default_title: string
  seo_default_description: string
  seo_default_og_image_url: string
  default_locale: string
  locales: SiteLocale[]
}

type Translate = (key: string) => string

function fallbackSiteSettings(t: Translate): SiteSettings {
  const name = t('site.fallbackName')
  const description = t('site.fallbackTagline')
  return {
    platform_name: name,
    public_frontend_url: '',
    public_api_url: '/api',
    registration_enabled: true,
    username_login_enabled: true,
    email_login_enabled: true,
    email_verification_required: true,
    default_avatar_url: '',
    site_name: name,
    logo_url: '',
    logo_dark_url: '',
    favicon_url: '/favicon.ico',
    tagline: description,
    footer_text: '',
    icp_record: '',
    contact_email: '',
    contact_phone: '',
    contact_address: '',
    social_links: [],
    seo_default_title: name,
    seo_default_description: description,
    seo_default_og_image_url: '',
    default_locale: 'en',
    locales: [
      { code: 'en', label: t('locale.names.en') },
      { code: 'zh-CN', label: t('locale.names.zh-CN') }
    ]
  }
}

/**
 * Site-wide branding and runtime locale set from
 * `GET /api/v1/public/settings` (contract §4.1). Falls back to built-in,
 * i18n-driven defaults so pages keep rendering when the API is unavailable.
 */
export function useSiteSettings() {
  const { t, locale, locales: configuredLocales } = useI18n()
  const api = useApi()

  const { data, pending, error, refresh } = useAsyncData(
    'site-settings',
    () => api.get<SiteSettings>('/v1/public/settings', { locale: locale.value }),
    { default: () => fallbackSiteSettings(t), watch: [locale] }
  )

  const settings = computed<SiteSettings>(() => {
    const fallback = fallbackSiteSettings(t)
    const fetched = data.value
    if (!fetched) {
      return fallback
    }
    return {
      ...fallback,
      ...fetched,
      social_links: fetched.social_links ?? [],
      locales: fetched.locales?.length ? fetched.locales : fallback.locales
    }
  })

  // Only locales enabled by the backend are offered; prefix routing stays build-time.
  const enabledLocales = computed<SiteLocale[]>(() => {
    const configured = (configuredLocales.value ?? []) as Array<{ code: string }>
    const enabledCodes = new Set(settings.value.locales.map(entry => entry.code))
    return configured
      .filter(entry => enabledCodes.size === 0 || enabledCodes.has(entry.code))
      .map(entry => ({
        code: entry.code,
        label: settings.value.locales.find(item => item.code === entry.code)?.label || t(`locale.names.${entry.code}`)
      }))
  })

  const defaultLocale = computed(() => settings.value.default_locale || 'en')

  return { settings, enabledLocales, defaultLocale, pending, error, refresh }
}
