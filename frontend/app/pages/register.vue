<script setup lang="ts">
const { t } = useI18n()
const { register, resendVerification } = useAuth()
const { settings } = useSiteSettings()
const router = useRouter()

const form = reactive({
  username: '',
  email: '',
  password: '',
  confirm: ''
})
const errors = reactive<Record<string, string>>({})
const formError = ref('')
const submitting = ref(false)
const created = ref('')
const status = ref<'form' | 'pending'>('form')
const resending = ref(false)
const resent = ref(false)

const EMAIL_PATTERN = /^[^\s@]+@[^\s@]+\.[^\s@]+$/
const registrationEnabled = computed(() => settings.value.registration_enabled)

function byteLength(value: string): number {
  return new TextEncoder().encode(value).length
}

function validate(): boolean {
  errors.username = !form.username.trim()
    ? t('auth.register.errors.usernameRequired')
    : (form.username.trim().length < 3 || form.username.trim().length > 64
        ? t('auth.register.errors.usernameLength')
        : '')
  errors.email = !form.email.trim()
    ? t('auth.register.errors.emailRequired')
    : (EMAIL_PATTERN.test(form.email.trim()) ? '' : t('auth.register.errors.emailInvalid'))
  const passwordBytes = byteLength(form.password)
  errors.password = !form.password
    ? t('auth.register.errors.passwordRequired')
    : (passwordBytes < 8 || passwordBytes > 72 ? t('auth.register.errors.passwordLength') : '')
  errors.confirm = !form.confirm
    ? t('auth.register.errors.confirmRequired')
    : (form.confirm === form.password ? '' : t('auth.register.errors.confirmMismatch'))
  return !Object.values(errors).some(Boolean)
}

async function submit() {
  formError.value = ''
  if (!validate()) {
    return
  }
  submitting.value = true
  try {
    const user = await register({
      username: form.username.trim(),
      email: form.email.trim(),
      password: form.password
    })
    if (user.status === 'pending_verification') {
      created.value = user.email
      status.value = 'pending'
    } else {
      await router.push({ path: '/login', query: { identifier: user.username, registered: '1' } })
    }
  } catch (error) {
    formError.value = error instanceof ApiError
      ? (error.code === 10001 ? t('auth.register.errors.validation') : t(apiErrorKey(error.code)))
      : t('errorCodes.unknown')
  } finally {
    submitting.value = false
  }
}

async function resend() {
  formError.value = ''
  resending.value = true
  try {
    await resendVerification(created.value)
    resent.value = true
  } catch (error) {
    formError.value = error instanceof ApiError ? t(apiErrorKey(error.code)) : t('errorCodes.unknown')
  } finally {
    resending.value = false
  }
}

useSeo({ title: t('auth.register.title'), description: t('auth.register.subtitle') })
</script>

<template>
  <div class="mx-auto max-w-lg px-4 py-16">
    <UPageHeader
      :title="t('auth.register.title')"
      :description="t('auth.register.subtitle')"
    />

    <UAlert
      v-if="!registrationEnabled"
      class="mt-8"
      color="warning"
      variant="soft"
      icon="i-lucide-lock"
      :title="t('auth.register.closedTitle')"
      :description="t('auth.register.closedDescription')"
    />

    <UAlert
      v-else-if="status === 'pending'"
      class="mt-8"
      color="success"
      variant="soft"
      icon="i-lucide-mail-check"
      :title="t('auth.register.pendingTitle')"
      :description="t('auth.register.pendingDescription', { email: created })"
    >
      <template #actions>
        <div class="flex flex-wrap items-center gap-2">
          <UButton
            color="primary"
            variant="soft"
            :loading="resending"
            :disabled="resending || resent"
            @click="resend"
          >
            {{ resent ? t('auth.register.resent') : t('auth.register.resend') }}
          </UButton>
          <UButton
            to="/login"
            color="neutral"
            variant="outline"
          >
            {{ t('auth.register.backToLogin') }}
          </UButton>
        </div>
      </template>
    </UAlert>

    <form
      v-else
      class="mt-8 flex flex-col gap-5"
      novalidate
      @submit.prevent="submit"
    >
      <UAlert
        v-if="formError"
        color="error"
        variant="soft"
        icon="i-lucide-triangle-alert"
        :description="formError"
      />

      <UFormField
        :label="t('auth.register.fields.username')"
        :error="errors.username"
        required
      >
        <UInput
          v-model="form.username"
          class="w-full"
          autocomplete="username"
        />
      </UFormField>

      <UFormField
        :label="t('auth.register.fields.email')"
        :error="errors.email"
        required
      >
        <UInput
          v-model="form.email"
          type="email"
          class="w-full"
          autocomplete="email"
        />
      </UFormField>

      <UFormField
        :label="t('auth.register.fields.password')"
        :error="errors.password"
        :hint="t('auth.register.fields.passwordHint')"
        required
      >
        <UInput
          v-model="form.password"
          type="password"
          class="w-full"
          autocomplete="new-password"
        />
      </UFormField>

      <UFormField
        :label="t('auth.register.fields.confirmPassword')"
        :error="errors.confirm"
        required
      >
        <UInput
          v-model="form.confirm"
          type="password"
          class="w-full"
          autocomplete="new-password"
        />
      </UFormField>

      <UButton
        type="submit"
        size="lg"
        block
        :loading="submitting"
        :disabled="submitting"
      >
        {{ t('auth.register.submit') }}
      </UButton>

      <p class="text-center text-sm text-muted">
        {{ t('auth.register.haveAccount') }}
        <NuxtLink
          to="/login"
          class="text-primary hover:underline"
        >
          {{ t('auth.register.login') }}
        </NuxtLink>
      </p>
    </form>
  </div>
</template>
