/**
 * Reports one page view per client-side route change to
 * `POST /api/v1/public/page-view` (contract §4.10).
 *
 * The request is fire-and-forget: it must never block navigation nor surface an
 * error. `navigator.sendBeacon` is preferred (it survives an immediate
 * navigation), with a `keepalive` `$fetch` as fallback. Nothing is persisted
 * locally and no auth token is involved.
 */

// Private, non-public areas are never reported.
const IGNORED_PREFIXES = ['/account']

export default defineNuxtPlugin((nuxtApp) => {
  const config = useRuntimeConfig()
  const endpoint = `${config.public.apiBase}/v1/public/page-view`

  useRouter().afterEach((to, _from, failure) => {
    // Redirected/aborted navigations (for example `/account` -> `/login`)
    // already produce their own page view; skip them to avoid double counting.
    if (failure) {
      return
    }

    const locale = nuxtApp.$i18n.locale.value

    // Strip the non-default locale prefix before matching private areas.
    const prefix = `/${locale}`
    const basePath = to.path.startsWith(`${prefix}/`) ? to.path.slice(prefix.length) : to.path
    if (IGNORED_PREFIXES.some(path => basePath === path || basePath.startsWith(`${path}/`))) {
      return
    }

    const payload = {
      path: to.path,
      referrer: document.referrer,
      locale
    }
    const body = JSON.stringify(payload)

    try {
      if (typeof navigator.sendBeacon === 'function') {
        const blob = new Blob([body], { type: 'application/json' })
        if (navigator.sendBeacon(endpoint, blob)) {
          return
        }
      }
    } catch {
      // Beacon failed synchronously; fall through to the fetch fallback.
    }

    // `keepalive` lets the request outlive a navigation that follows it.
    $fetch(endpoint, {
      method: 'POST',
      body: payload,
      keepalive: true
    }).catch(() => {
      // Statistics only: failures are silent.
    })
  })
})
