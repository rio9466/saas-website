<script setup lang="ts">
import type { UpdateProfilePayload } from '~/composables/useAuth'

definePageMeta({ layout: 'account', middleware: 'auth' })

const { t } = useI18n()
const { user, updateProfile } = useAuth()

const form = reactive({
  nickname: user.value?.nickname || '',
  avatar_url: user.value?.avatar_url || ''
})
const formError = ref('')
const saved = ref(false)
const submitting = ref(false)

async function submit() {
  formError.value = ''
  saved.value = false

  const payload: UpdateProfilePayload = {}
  if (form.nickname.trim() !== (user.value?.nickname || '')) {
    payload.nickname = form.nickname.trim()
  }
  if (form.avatar_url.trim() !== (user.value?.avatar_url || '')) {
    payload.avatar_url = form.avatar_url.trim()
  }
  if (!Object.keys(payload).length) {
    saved.value = true
    return
  }

  submitting.value = true
  try {
    const updated = await updateProfile(payload)
    form.nickname = updated.nickname
    form.avatar_url = updated.avatar_url
    saved.value = true
  } catch (error) {
    formError.value = error instanceof ApiError ? t(apiErrorKey(error.code)) : t('errorCodes.unknown')
  } finally {
    submitting.value = false
  }
}

useSeo({ title: t('account.profile.title') })
</script>

<template>
  <div>
    <UPageHeader
      :title="t('account.profile.title')"
      :description="t('account.profile.subtitle')"
    />

    <form
      class="mt-8 flex max-w-md flex-col gap-5"
      novalidate
      @submit.prevent="submit"
    >
      <UAlert
        v-if="saved"
        color="success"
        variant="soft"
        icon="i-lucide-circle-check"
        :description="t('account.profile.saved')"
      />

      <UAlert
        v-if="formError"
        color="error"
        variant="soft"
        icon="i-lucide-triangle-alert"
        :description="formError"
      />

      <UFormField
        :label="t('account.profile.nickname')"
        :hint="t('account.profile.nicknameHint')"
      >
        <UInput
          v-model="form.nickname"
          class="w-full"
          autocomplete="nickname"
        />
      </UFormField>

      <UFormField :label="t('account.profile.avatarUrl')">
        <UInput
          v-model="form.avatar_url"
          class="w-full"
          placeholder="https://…"
        />
      </UFormField>

      <UButton
        type="submit"
        size="lg"
        :loading="submitting"
        :disabled="submitting"
        class="self-start"
      >
        {{ t('account.profile.submit') }}
      </UButton>
    </form>
  </div>
</template>
