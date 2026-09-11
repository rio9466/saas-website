// Same-origin proxy for the Go API (contract §2.1/§7.1): the browser must only
// ever call `/api/**` on this origin, and Nitro forwards it to the backend.
//
// The target is read from runtimeConfig on every request so that
// `NUXT_API_PROXY_TARGET` can be changed when starting `.output` without a
// rebuild. `proxyRequest` streams the request body, preserves the query string
// and forwards Origin/Referer plus the backend's Set-Cookie response headers.
export default defineEventHandler((event) => {
  if (!event.path.startsWith('/api/')) {
    return
  }

  const target = String(useRuntimeConfig().apiProxyTarget || '').replace(/\/+$/, '')
  if (!target) {
    return
  }

  const requestUrl = getRequestURL(event)
  return proxyRequest(event, `${target}${requestUrl.pathname}${requestUrl.search}`)
})
