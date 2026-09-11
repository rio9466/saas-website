<script setup lang="ts">
const { t } = useI18n()
const { settings } = useSiteSettings()
const navItems = useNavigation('header')
const { isAuthenticated, user } = useAuth()
const localePath = useLocalePath()
const localizedUrl = useLocalizedUrl()

function targetOf(target: string) {
  return target || '_self'
}
</script>

<template>
  <UHeader
    :title="settings.site_name"
    :to="localePath('/')"
    mode="slideover"
  >
    <template #left>
      <AppLink
        to="/"
        :aria-label="settings.site_name"
        class="flex items-center"
      >
        <AppLogo
          :name="settings.site_name"
          :logo="settings.logo_url"
          :logo-dark="settings.logo_dark_url"
        />
      </AppLink>
    </template>

    <nav class="hidden lg:flex items-center gap-1">
      <UButton
        v-for="item in navItems"
        :key="item.id"
        :to="localizedUrl(item.url)"
        :target="targetOf(item.target)"
        color="neutral"
        variant="ghost"
        size="sm"
      >
        {{ item.label }}
      </UButton>
    </nav>

    <template #right>
      <LocaleSwitcher />
      <ThemeToggle />
      <UButton
        v-if="isAuthenticated"
        :to="localePath('/account')"
        color="neutral"
        variant="ghost"
        size="sm"
        icon="i-lucide-user"
      >
        {{ user?.nickname || t('header.console') }}
      </UButton>
      <template v-else>
        <UButton
          :to="localePath('/login')"
          color="neutral"
          variant="ghost"
          size="sm"
        >
          {{ t('header.login') }}
        </UButton>
        <UButton
          :to="localePath('/register')"
          color="primary"
          size="sm"
        >
          {{ t('header.register') }}
        </UButton>
      </template>
    </template>

    <template #toggle="{ open, toggle }">
      <UButton
        class="lg:hidden"
        color="neutral"
        variant="ghost"
        :icon="open ? 'i-lucide-x' : 'i-lucide-menu'"
        :aria-label="open ? t('header.closeMenu') : t('header.openMenu')"
        @click="toggle"
      />
    </template>

    <template #body>
      <nav class="flex flex-col gap-1">
        <UButton
          v-for="item in navItems"
          :key="item.id"
          :to="localizedUrl(item.url)"
          :target="targetOf(item.target)"
          color="neutral"
          variant="ghost"
          class="justify-start"
        >
          {{ item.label }}
        </UButton>
      </nav>
    </template>
  </UHeader>
</template>
