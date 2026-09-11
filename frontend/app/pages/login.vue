<script setup lang="ts">
const { t } = useI18n()
const route = useRoute()
const router = useRouter()
const { login, resendVerification } = useAuth()
const { settings } = useSiteSettings()

const form = reactive({
  identifier: typeof route.query.identifier === 'string' ? route.query.identifier : '',
  password: ''
})
const errors = reactive<Record<string, string>>({})
const formError = ref('')
const submitting = ref(false)
const unverified = ref(false)
const resendEmail = ref('')
const resending = ref(false)
const resent = ref(false)
const registered = computed(() => route.query.registered === '1')
const resetDone = computed(() => route.query.reset === '1')
const changedDone = computed(() => route.query.changed === '1')

const usernameEnabled = computed(() => settings.value.username_login_enabled)
const emailEnabled = computed(() => settings.value.email_login_enabled)
const registrationEnabled = computed(() => settings.value.registration_enabled)

const identifierHint = computed(() => {
  if (usernameEnabled.value && emailEnabled.value) {
    return t('auth.login.hintBoth')
  }
  if (emailEnabled.value) {
    return t('auth.login.hintEmail')
  }
  return t('auth.login.hintUsername')
})

/** Only allow same-site absolute paths, preventing open redirects. */
function safeRedirect(value: unknown): string | null {
  if (typeof value !== 'string' || !value.startsWith('/') || value.startsWith('//')) {
    return null
  }
  if (value.includes('\\')) {
    return null
  }
  return value
}

function validate(): boolean {
  errors.identifier = form.identifier.trim() ? '' : t('auth.login.errors.identifierRequired')
  errors.password = form.password ? '' : t('auth.login.errors.passwordRequired')
  return !Object.values(errors).some(Boolean)
}

async function submit() {
  formError.value = ''
  unverified.value = false
  if (!validate()) {
    return
  }
  submitting.value = true
  try {
    await login({ identifier: form.identifier.trim(), password: form.password })
    await router.push(safeRedirect(route.query.redirect) || '/account')
  } catch (error) {
    if (error instanceof ApiError && error.code === 40011) {
      unverified.value = true
      resendEmail.value = form.identifier.includes('@') ? form.identifier.trim() : ''
    } else {
      formError.value = error instanceof ApiError && error.code === 10001
        ? t('auth.login.errors.validation')
        : (error instanceof ApiError ? t(apiErrorKey(error.code)) : t('errorCodes.unknown'))
    }
  } finally {
    submitting.value = false
  }
}

async function resend() {
  if (!resendEmail.value.trim()) {
    return
  }
  formError.value = ''
  resending.value = true
  try {
    await resendVerification(resendEmail.value.trim())
    resent.value = true
  } catch (error) {
    formError.value = error instanceof ApiError ? t(apiErrorKey(error.code)) : t('errorCodes.unknown')
  } finally {
    resending.value = false
  }
}

useSeo({ title: t('auth.login.title'), description: t('auth.login.subtitle') })
</script>

<template>
  <div class="mx-auto max-w-lg px-4 py-16">
    <UPageHeader
      :title="t('auth.login.title')"
      :description="t('auth.login.subtitle')"
    />

    <UAlert
      v-if="registered"
      class="mt-8"
      color="success"
      variant="soft"
      icon="i-lucide-circle-check"
      :description="t('auth.login.registered')"
    />

    <UAlert
      v-if="resetDone"
      class="mt-8"
      color="success"
      variant="soft"
      icon="i-lucide-circle-check"
      :description="t('auth.login.passwordReset')"
    />

    <UAlert
      v-if="changedDone"
      class="mt-8"
      color="success"
      variant="soft"
      icon="i-lucide-circle-check"
      :description="t('auth.login.passwordChanged')"
    />

    <form
      class="mt-8 flex flex-col gap-5"
      novalidate
      @submit.prevent="submit"
    >
      <UAlert
        v-if="!usernameEnabled && !emailEnabled"
        color="warning"
        variant="soft"
        icon="i-lucide-lock"
        :description="t('errorCodes.40012')"
      />

      <UAlert
        v-if="formError"
        color="error"
        variant="soft"
        icon="i-lucide-triangle-alert"
        :description="formError"
      />

      <UAlert
        v-if="unverified"
        color="warning"
        variant="soft"
        icon="i-lucide-mail-warning"
        :title="t('auth.login.unverifiedTitle')"
        :description="t('auth.login.unverifiedDescription')"
      >
        <template #actions>
          <div class="flex w-full flex-col gap-2 sm:flex-row sm:items-center">
            <UInput
              v-model="resendEmail"
              type="email"
              class="w-full sm:max-w-xs"
              :placeholder="t('auth.login.fields.email')"
            />
            <UButton
              color="primary"
              variant="soft"
              :loading="resending"
              :disabled="resending || resent || !resendEmail.trim()"
              @click="resend"
            >
              {{ resent ? t('auth.login.resent') : t('auth.login.resend') }}
            </UButton>
          </div>
        </template>
      </UAlert>

      <UFormField
        :label="t('auth.login.fields.identifier')"
        :error="errors.identifier"
        :hint="identifierHint"
        required
      >
        <UInput
          v-model="form.identifier"
          class="w-full"
          autocomplete="username"
        />
      </UFormField>

      <UFormField
        :label="t('auth.login.fields.password')"
        :error="errors.password"
        required
      >
        <UInput
          v-model="form.password"
          type="password"
          class="w-full"
          autocomplete="current-password"
        />
      </UFormField>

      <div class="flex items-center justify-between">
        <NuxtLink
          to="/forgot-password"
          class="text-sm text-primary hover:underline"
        >
          {{ t('auth.login.forgot') }}
        </NuxtLink>
      </div>

      <UButton
        type="submit"
        size="lg"
        block
        :loading="submitting"
        :disabled="submitting || (!usernameEnabled && !emailEnabled)"
      >
        {{ t('auth.login.submit') }}
      </UButton>

      <p
        v-if="registrationEnabled"
        class="text-center text-sm text-muted"
      >
        {{ t('auth.login.noAccount') }}
        <NuxtLink
          to="/register"
          class="text-primary hover:underline"
        >
          {{ t('auth.login.signUp') }}
        </NuxtLink>
      </p>
    </form>
  </div>
</template>
