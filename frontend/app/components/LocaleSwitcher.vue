<script setup lang="ts">
const { t, locale } = useI18n()
const switchLocalePath = useSwitchLocalePath()
const { enabledLocales } = useSiteSettings()
type LocaleCode = Parameters<typeof switchLocalePath>[0]

const items = computed(() => enabledLocales.value.map(entry => ({
  label: entry.label,
  to: switchLocalePath(entry.code as LocaleCode) || '/',
  icon: entry.code === locale.value ? 'i-lucide-check' : undefined
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
