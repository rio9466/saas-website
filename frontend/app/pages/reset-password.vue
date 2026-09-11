<script setup lang="ts">
const { t } = useI18n()
const route = useRoute()
const router = useRouter()
const { resetPassword } = useAuth()

const email = computed(() => String(route.query.email || '').trim())
const token = computed(() => String(route.query.token || '').trim())

const form = reactive({
  password: '',
  confirm: ''
})
const errors = reactive<Record<string, string>>({})
const formError = ref('')
const submitting = ref(false)
const invalidLink = computed(() => !email.value || !token.value)

function byteLength(value: string): number {
  return new TextEncoder().encode(value).length
}

function validate(): boolean {
  const passwordBytes = byteLength(form.password)
  errors.password = !form.password
    ? t('auth.reset.errors.passwordRequired')
    : (passwordBytes < 8 || passwordBytes > 72 ? t('auth.reset.errors.passwordLength') : '')
  errors.confirm = !form.confirm
    ? t('auth.reset.errors.confirmRequired')
    : (form.confirm === form.password ? '' : t('auth.reset.errors.confirmMismatch'))
  return !Object.values(errors).some(Boolean)
}

async function submit() {
  formError.value = ''
  if (invalidLink.value || !validate()) {
    return
  }
  submitting.value = true
  try {
    await resetPassword(email.value, token.value, form.password)
    await router.push({ path: '/login', query: { reset: '1' } })
  } catch (error) {
    formError.value = error instanceof ApiError
      ? (error.code === 40016 ? t('auth.reset.invalidDescription') : t(apiErrorKey(error.code)))
      : t('errorCodes.unknown')
  } finally {
    submitting.value = false
  }
}

useSeo({ title: t('auth.reset.title'), description: t('auth.reset.subtitle') })
</script>

<template>
  <div class="mx-auto max-w-lg px-4 py-16">
    <UPageHeader
      :title="t('auth.reset.title')"
      :description="t('auth.reset.subtitle')"
    />

    <UAlert
      v-if="invalidLink"
      class="mt-8"
      color="error"
      variant="soft"
      icon="i-lucide-triangle-alert"
      :title="t('auth.reset.invalidTitle')"
      :description="t('auth.reset.invalidDescription')"
    >
      <template #actions>
        <UButton
          to="/forgot-password"
          color="primary"
          variant="soft"
        >
          {{ t('auth.reset.requestNew') }}
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
        v-if="formError"
        color="error"
        variant="soft"
        icon="i-lucide-triangle-alert"
        :description="formError"
      />

      <UFormField
        :label="t('auth.reset.fields.password')"
        :error="errors.password"
        :hint="t('auth.reset.fields.passwordHint')"
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
        :label="t('auth.reset.fields.confirmPassword')"
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
        {{ t('auth.reset.submit') }}
      </UButton>
    </form>
  </div>
</template>
