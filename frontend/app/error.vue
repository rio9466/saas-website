<script setup lang="ts">
import type { NuxtError } from '#app'

const props = defineProps<{ error: NuxtError }>()

const { t } = useI18n()

const isNotFound = computed(() => props.error?.statusCode === 404)
const title = computed(() => (isNotFound.value ? t('error.notFoundTitle') : t('error.serverTitle')))
const description = computed(
  () => (isNotFound.value ? t('error.notFoundDescription') : t('error.serverDescription'))
)

function goHome() {
  clearError({ redirect: '/' })
}
</script>

<template>
  <UApp>
    <div class="flex min-h-screen items-center justify-center p-6">
      <div class="flex max-w-md flex-col items-center gap-4 text-center">
        <p class="text-5xl font-semibold text-primary">
          {{ error?.statusCode }}
        </p>
        <h1 class="text-xl font-semibold text-highlighted">
          {{ title }}
        </h1>
        <p class="text-muted">
          {{ description }}
        </p>
        <UButton
          color="primary"
          icon="i-lucide-house"
          @click="goHome"
        >
          {{ t('common.backHome') }}
        </UButton>
      </div>
    </div>
  </UApp>
</template>
