<script setup lang="ts">
definePageMeta({ layout: 'account', middleware: 'auth' })

const { t } = useI18n()
const router = useRouter()
const { changePassword } = useAuth()
const localePath = useLocalePath()

const form = reactive({
  current: '',
  next: '',
  confirm: ''
})
const errors = reactive<Record<string, string>>({})
const formError = ref('')
const submitting = ref(false)

function byteLength(value: string): number {
  return new TextEncoder().encode(value).length
}

function validate(): boolean {
  errors.current = form.current ? '' : t('account.security.errors.currentRequired')
  const nextBytes = byteLength(form.next)
  errors.next = !form.next
    ? t('account.security.errors.newRequired')
    : (nextBytes < 8 || nextBytes > 72 ? t('account.security.errors.newLength') : '')
  errors.confirm = !form.confirm
    ? t('account.security.errors.confirmRequired')
    : (form.confirm === form.next ? '' : t('account.security.errors.confirmMismatch'))
  return !Object.values(errors).some(Boolean)
}

async function submit() {
  formError.value = ''
  if (!validate()) {
    return
  }
  submitting.value = true
  try {
    await changePassword(form.current, form.next)
    await router.push({ path: localePath('/login'), query: { changed: '1' } })
  } catch (error) {
    if (error instanceof ApiError && error.code === 40005) {
      errors.current = t('errorCodes.40005')
    } else {
      formError.value = error instanceof ApiError ? t(apiErrorKey(error.code)) : t('errorCodes.unknown')
    }
  } finally {
    submitting.value = false
  }
}

useSeo({ title: t('account.security.title') })
</script>

<template>
  <div>
    <UPageHeader
      :title="t('account.security.title')"
      :description="t('account.security.subtitle')"
    />

    <form
      class="mt-8 flex max-w-md flex-col gap-5"
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
        :label="t('account.security.currentPassword')"
        :error="errors.current"
        required
      >
        <UInput
          v-model="form.current"
          type="password"
          class="w-full"
          autocomplete="current-password"
        />
      </UFormField>

      <UFormField
        :label="t('account.security.newPassword')"
        :error="errors.next"
        :hint="t('account.security.passwordHint')"
        required
      >
        <UInput
          v-model="form.next"
          type="password"
          class="w-full"
          autocomplete="new-password"
        />
      </UFormField>

      <UFormField
        :label="t('account.security.confirmPassword')"
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
        :loading="submitting"
        :disabled="submitting"
        class="self-start"
      >
        {{ t('account.security.submit') }}
      </UButton>
    </form>
  </div>
</template>
