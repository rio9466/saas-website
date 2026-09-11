<script setup lang="ts">
const props = withDefaults(defineProps<{
  error?: unknown
  title?: string
  description?: string
}>(), {
  title: '',
  description: ''
})

const emit = defineEmits<{ retry: [] }>()

const { t } = useI18n()

const title = computed(() => props.title || t('common.error'))
const description = computed(() => {
  if (props.error instanceof ApiError) {
    return t(apiErrorKey(props.error.code))
  }
  return props.description || t('errorCodes.unknown')
})
</script>

<template>
  <div class="flex flex-col items-center justify-center gap-3 py-16 text-center">
    <UIcon
      name="i-lucide-triangle-alert"
      class="size-8 text-error"
    />
    <p class="font-medium text-highlighted">
      {{ title }}
    </p>
    <p class="text-sm text-muted max-w-md">
      {{ description }}
    </p>
    <UButton
      color="primary"
      variant="soft"
      icon="i-lucide-refresh-cw"
      @click="emit('retry')"
    >
      {{ t('common.retry') }}
    </UButton>
  </div>
</template>
