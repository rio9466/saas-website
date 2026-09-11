<script setup lang="ts">
withDefaults(defineProps<{
  title?: string
  description?: string
  icon?: string
}>(), {
  title: '',
  description: '',
  icon: 'i-lucide-inbox'
})

const { t } = useI18n()
</script>

<template>
  <div class="flex flex-col items-center justify-center gap-2 py-16 text-center">
    <UIcon
      :name="icon"
      class="size-8 text-muted"
    />
    <p class="font-medium text-highlighted">
      {{ title || t('common.empty') }}
    </p>
    <p
      v-if="description"
      class="text-sm text-muted max-w-md"
    >
      {{ description }}
    </p>
  </div>
</template>
