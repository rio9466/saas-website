// Client-side guard for `/account/**` (contract §2, task FE-03).
//
// SSR page requests never carry the refresh cookie (`Path=/api/v1/auth`), so
// authentication can only be decided in the browser. On the very first
// hydration the account layout renders a loading shell identical to the server
// output and restores the session on mount; this middleware handles every later
// client-side navigation, redirecting unauthenticated visitors to the login
// page with a return path.
export default defineNuxtRouteMiddleware(async (to) => {
  if (import.meta.server) {
    return
  }

  const { isAuthenticated, restoreSession } = useAuth()
  if (isAuthenticated.value) {
    return
  }

  if (useNuxtApp().isHydrating) {
    return
  }

  const restored = await restoreSession()
  if (!restored) {
    return navigateTo(
      { path: '/login', query: { redirect: to.fullPath } },
      { replace: true }
    )
  }
})
