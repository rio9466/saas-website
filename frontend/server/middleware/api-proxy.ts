// Same-origin proxy for the Go API (contract §2.1/§7.1): the browser must only
// ever call `/api/**` on this origin, and Nitro forwards it to the backend.
//
// The target is read from runtimeConfig on every request so that
// `NUXT_API_PROXY_TARGET` can be changed when starting `.output` without a
// rebuild (a `routeRules` proxy target is frozen at build time).
//
// h3's `proxyRequest` forwards the client's `content-length` header, which
// Node's `fetch`/undici rejects with `UND_ERR_INVALID_ARG: invalid content-length
// header` once it also receives the proxied body — poisoning the connection
// pool so every later POST fails. Read the raw body first (while the header is
// still present), strip the hop-by-hop/forbidden headers, and let `sendProxy`
// forward body, cookies, Origin/Referer and the backend's Set-Cookie response.
const PAYLOAD_METHODS = new Set(['PATCH', 'POST', 'PUT', 'DELETE'])
const STRIPPED_HEADERS = new Set([
  'content-length',
  'transfer-encoding',
  'accept-encoding',
  'connection',
  'keep-alive',
  'upgrade',
  'expect',
  'host',
  'accept'
])

export default defineEventHandler(async (event) => {
  if (!event.path.startsWith('/api/')) {
    return
  }

  const target = String(useRuntimeConfig().apiProxyTarget || '').replace(/\/+$/, '')
  if (!target) {
    return
  }

  const body = PAYLOAD_METHODS.has(event.method) ? await readRawBody(event, false) : undefined

  const headers: Record<string, string> = {}
  for (const [name, value] of Object.entries(getRequestHeaders(event))) {
    if (value === undefined || STRIPPED_HEADERS.has(name)) {
      continue
    }
    headers[name] = Array.isArray(value) ? value.join(', ') : value
  }

  const requestUrl = getRequestURL(event)
  return sendProxy(event, `${target}${requestUrl.pathname}${requestUrl.search}`, {
    fetchOptions: {
      method: event.method,
      body,
      headers
    }
  })
})
