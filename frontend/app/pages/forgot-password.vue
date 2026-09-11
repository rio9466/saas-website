<script setup lang="ts">
const { t } = useI18n()
const { forgotPassword } = useAuth()
const localePath = useLocalePath()

const email = ref('')
const error = ref('')
const fieldError = ref('')
const submitting = ref(false)
const sent = ref(false)

const EMAIL_PATTERN = /^[^\s@]+@[^\s@]+\.[^\s@]+$/

async function submit() {
  error.value = ''
  fieldError.value = ''
  if (!email.value.trim()) {
    fieldError.value = t('auth.forgot.errors.emailRequired')
    return
  }
  if (!EMAIL_PATTERN.test(email.value.trim())) {
    fieldError.value = t('auth.forgot.errors.emailInvalid')
    return
  }
  submitting.value = true
  try {
    await forgotPassword(email.value.trim())
    sent.value = true
  } catch (err) {
    error.value = err instanceof ApiError && err.code === 10001
      ? t('auth.forgot.errors.emailInvalid')
      : (err instanceof ApiError ? t(apiErrorKey(err.code)) : t('errorCodes.unknown'))
  } finally {
    submitting.value = false
  }
}

useSeo({ title: t('auth.forgot.title'), description: t('auth.forgot.subtitle') })
</script>

<template>
  <div class="mx-auto max-w-lg px-4 py-16">
    <UPageHeader
      :title="t('auth.forgot.title')"
      :description="t('auth.forgot.subtitle')"
    />

    <UAlert
      v-if="sent"
      class="mt-8"
      color="success"
      variant="soft"
      icon="i-lucide-mail-check"
      :title="t('auth.forgot.sentTitle')"
      :description="t('auth.forgot.sentDescription')"
    >
      <template #actions>
        <UButton
          :to="localePath('/login')"
          color="neutral"
          variant="outline"
        >
          {{ t('auth.forgot.backToLogin') }}
        </UButton>
      </template>
    </UAlert>

    <form
      v-else
      class="mt-8 flex flex-col gap-5"
      novalidate
      @submit.prevent="submit"
    >
      <UAlert
        v-if="error"
        color="error"
        variant="soft"
        icon="i-lucide-triangle-alert"
        :description="error"
      />

      <UFormField
        :label="t('auth.forgot.email')"
        :error="fieldError"
        required
      >
        <UInput
          v-model="email"
          type="email"
          class="w-full"
          autocomplete="email"
        />
      </UFormField>

      <UButton
        type="submit"
        size="lg"
        block
        :loading="submitting"
        :disabled="submitting"
      >
        {{ t('auth.forgot.submit') }}
      </UButton>

      <p class="text-center text-sm text-muted">
        <AppLink
          to="/login"
          class="text-primary hover:underline"
        >
          {{ t('auth.forgot.backToLogin') }}
        </AppLink>
      </p>
    </form>
  </div>
</template>
