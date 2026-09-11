<script setup lang="ts">
import { usePreferencesStore } from '~/stores/preferences'

const { t, locale, setLocale } = useI18n()
const { enabledLocales } = useSiteSettings()
const prefs = usePreferencesStore()
type LocaleCode = Parameters<typeof setLocale>[0]

async function switchTo(code: LocaleCode) {
  if (code === locale.value) {
    return
  }
  // `setLocale` swaps the locale and rewrites the current route in the target
  // language; the module also persists the choice to the durable
  // `i18n_redirected` cookie (365 days). Using it instead of a `to` link avoids
  // NuxtLink re-localizing the already-switched path (which kept the user on the
  // current locale).
  await setLocale(code)
  // Mirror the choice in the preferences store so the startup plugin can
  // restore it even if the i18n cookie was cleared.
  prefs.locale = code
}

const items = computed(() => enabledLocales.value.map(entry => ({
  label: entry.label,
  onSelect: () => switchTo(entry.code as LocaleCode)
})))

const currentLabel = computed(
  () => enabledLocales.value.find(entry => entry.code === locale.value)?.label || locale.value
)
</script>

<template>
  <UDropdownMenu
    :items="items"
    :content="{ align: 'end' }"
  >
    <UButton
      color="neutral"
      variant="ghost"
      icon="i-lucide-languages"
      :label="currentLabel"
      :aria-label="t('locale.switchLabel')"
    />
  </UDropdownMenu>
</template>
