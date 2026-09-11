<script setup lang="ts">
const { t } = useI18n()
const route = useRoute()
const router = useRouter()
const { user, isAuthenticated, restoreSession, logout } = useAuth()
const localePath = useLocalePath()

const hydrated = ref(false)

// Restore the session only after hydration so the first client render matches
// the server-rendered loading shell (no hydration mismatch in the shared
// header), then redirect if the refresh cookie turned out to be invalid.
onMounted(async () => {
  if (!isAuthenticated.value) {
    const restored = await restoreSession()
    if (!restored) {
      await router.replace({ path: localePath('/login'), query: { redirect: route.fullPath } })
      return
    }
  }
  hydrated.value = true
})

const navItems = computed(() => [
  { label: t('account.nav.overview'), path: '/account', to: localePath('/account'), icon: 'i-lucide-layout-dashboard' },
  { label: t('account.nav.points'), path: '/account/points', to: localePath('/account/points'), icon: 'i-lucide-coins' },
  { label: t('account.nav.security'), path: '/account/security', to: localePath('/account/security'), icon: 'i-lucide-shield-check' },
  { label: t('account.nav.profile'), path: '/account/profile', to: localePath('/account/profile'), icon: 'i-lucide-user-round' }
])

const showLoading = computed(() => !hydrated.value || !user.value)

async function onLogout() {
  await logout()
  await router.push(localePath('/'))
}

function isActive(path: string) {
  const localized = localePath(path)
  return path === '/account' ? route.path === localized : route.path.startsWith(localized)
}
</script>

<template>
  <div class="flex min-h-screen flex-col">
    <AppHeader />
    <UMain class="flex-1">
      <div class="mx-auto grid max-w-6xl gap-8 px-4 py-10 lg:grid-cols-[16rem_minmax(0,1fr)]">
        <aside class="flex flex-col gap-4">
          <div class="rounded-lg border border-default p-4">
            <p class="text-xs uppercase tracking-wide text-dimmed">
              {{ t('account.signedInAs') }}
            </p>
            <p class="mt-1 truncate font-medium text-highlighted">
              {{ user?.nickname || user?.username || t('common.loading') }}
            </p>
            <p
              v-if="user?.email"
              class="truncate text-sm text-muted"
            >
              {{ user.email }}
            </p>
          </div>

          <nav class="flex gap-1 overflow-x-auto lg:flex-col lg:overflow-visible">
            <UButton
              v-for="item in navItems"
              :key="item.path"
              :to="item.to"
              :icon="item.icon"
              :color="isActive(item.path) ? 'primary' : 'neutral'"
              :variant="isActive(item.path) ? 'soft' : 'ghost'"
              class="shrink-0 justify-start"
            >
              {{ item.label }}
            </UButton>
          </nav>

          <UButton
            color="neutral"
            variant="outline"
            icon="i-lucide-log-out"
            class="justify-start"
            @click="onLogout"
          >
            {{ t('account.logout') }}
          </UButton>
        </aside>

        <section>
          <AppLoading v-if="showLoading" />
          <slot v-else />
        </section>
      </div>
    </UMain>
    <AppFooter />
  </div>
</template>
