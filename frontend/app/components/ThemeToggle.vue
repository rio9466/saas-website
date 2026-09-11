<script setup lang="ts">
import { usePreferencesStore } from '~/stores/preferences'

const { t } = useI18n()
const colorMode = useColorMode()
const appConfig = useAppConfig()
const prefs = usePreferencesStore()

const isDark = computed(() => colorMode.value === 'dark')

function toggle() {
  const next = isDark.value ? 'light' : 'dark'
  colorMode.preference = next
  // Keep the Pinia store in sync so `app/plugins/preferences.ts` can restore
  // the choice if the colour-mode cookie is ever cleared.
  prefs.colorMode = next
}
</script>

<template>
  <UButton
    color="neutral"
    variant="ghost"
    :aria-label="t('header.toggleTheme')"
    @click="toggle"
  >
    <UIcon
      :name="appConfig.ui.icons.dark"
      class="hidden dark:inline-block"
    />
    <UIcon
      :name="appConfig.ui.icons.light"
      class="dark:hidden"
    />
  </UButton>
</template>
