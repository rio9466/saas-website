/**
 * Returns a helper that adds the active locale prefix to same-site absolute
 * paths (e.g. `/features` → `/zh-CN/features`). External URLs (`http(s)://`,
 * `mailto:`, `tel:`, protocol-relative `//…`) and empty values are returned
 * unchanged.
 *
 * Use it for string `to` props (`UButton`, `UHeader`) and programmatic
 * navigation; prefer `AppLink` for `<NuxtLink>`-style anchors.
 */
export function useLocalizedUrl() {
  const localePath = useLocalePath()
  return (url: string | null | undefined): string => {
    if (!url) {
      return ''
    }
    return url.startsWith('/') && !url.startsWith('//') ? localePath(url) : url
  }
}
