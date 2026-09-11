<script setup lang="ts">
const { t } = useI18n()
</script>

<template>
  <div
    class="flex flex-col items-center justify-center gap-3 py-16 text-muted"
    role="status"
    aria-live="polite"
  >
    <UIcon
      name="i-lucide-loader-circle"
      class="size-6 animate-spin"
    />
    <span class="text-sm">{{ t('common.loading') }}</span>
  </div>
</template>
