<script setup lang="ts">
const { t } = useI18n()
const route = useRoute()
const { verifyEmail, resendVerification } = useAuth()
const localePath = useLocalePath()

const email = computed(() => String(route.query.email || '').trim())
const token = computed(() => String(route.query.token || '').trim())
const state = ref<'verifying' | 'success' | 'error'>('verifying')
const errorKey = ref('')
const resending = ref(false)
const resent = ref(false)

async function verify() {
  if (!email.value || !token.value) {
    state.value = 'error'
    errorKey.value = 'auth.verifyEmail.missing'
    return
  }
  try {
    await verifyEmail(email.value, token.value)
    state.value = 'success'
  } catch (error) {
    state.value = 'error'
    errorKey.value = error instanceof ApiError
      ? (error.code === 40016 ? 'auth.verifyEmail.invalidDescription' : apiErrorKey(error.code))
      : 'errorCodes.unknown'
  }
}

async function resend() {
  resending.value = true
  errorKey.value = ''
  try {
    await resendVerification(email.value)
    resent.value = true
  } catch (error) {
    errorKey.value = error instanceof ApiError ? apiErrorKey(error.code) : 'errorCodes.unknown'
  } finally {
    resending.value = false
  }
}

onMounted(verify)

useSeo({ title: t('auth.verifyEmail.title') })
</script>

<template>
  <div class="mx-auto max-w-lg px-4 py-16">
    <UPageHeader :title="t('auth.verifyEmail.title')" />

    <AppLoading v-if="state === 'verifying'" />

    <UAlert
      v-else-if="state === 'success'"
      class="mt-8"
      color="success"
      variant="soft"
      icon="i-lucide-circle-check"
      :title="t('auth.verifyEmail.successTitle')"
      :description="t('auth.verifyEmail.successDescription')"
    >
      <template #actions>
        <UButton
          :to="localePath('/login')"
          color="primary"
          variant="soft"
        >
          {{ t('auth.verifyEmail.goToLogin') }}
        </UButton>
      </template>
    </UAlert>

    <UAlert
      v-else
      class="mt-8"
      color="error"
      variant="soft"
      icon="i-lucide-triangle-alert"
      :title="t('auth.verifyEmail.invalidTitle')"
      :description="t(errorKey)"
    >
      <template #actions>
        <div class="flex flex-wrap items-center gap-2">
          <UButton
            v-if="email"
            color="primary"
            variant="soft"
            :loading="resending"
            :disabled="resending || resent"
            @click="resend"
          >
            {{ resent ? t('auth.verifyEmail.resent') : t('auth.verifyEmail.resend') }}
          </UButton>
          <UButton
            :to="localePath('/login')"
            color="neutral"
            variant="outline"
          >
            {{ t('auth.verifyEmail.goToLogin') }}
          </UButton>
        </div>
      </template>
    </UAlert>
  </div>
</template>
