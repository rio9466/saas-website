<script setup lang="ts">
const { t, locale } = useI18n()
const api = useApi()

const form = reactive({
  name: '',
  email: '',
  company: '',
  message: '',
  consent: false,
  website: ''
})

const errors = reactive<Record<string, string>>({})
const submitting = ref(false)
const submitted = ref(false)
const formError = ref('')

const EMAIL_PATTERN = /^[^\s@]+@[^\s@]+\.[^\s@]+$/

function validate(): boolean {
  errors.name = form.name.trim() ? '' : t('contact.errors.nameRequired')
  errors.email = !form.email.trim()
    ? t('contact.errors.emailRequired')
    : (EMAIL_PATTERN.test(form.email.trim()) ? '' : t('contact.errors.emailInvalid'))
  errors.message = form.message.trim() ? '' : t('contact.errors.messageRequired')
  errors.consent = form.consent ? '' : t('contact.errors.consentRequired')
  return !Object.values(errors).some(Boolean)
}

function reset() {
  form.name = ''
  form.email = ''
  form.company = ''
  form.message = ''
  form.consent = false
  form.website = ''
  Object.keys(errors).forEach((key) => {
    errors[key] = ''
  })
}

async function submit() {
  formError.value = ''
  if (!validate()) {
    return
  }
  submitting.value = true
  try {
    await api.post('/v1/public/contact', {
      name: form.name.trim(),
      email: form.email.trim(),
      company: form.company.trim(),
      message: form.message.trim(),
      locale: locale.value,
      consent: form.consent,
      website: form.website
    })
    reset()
    submitted.value = true
  } catch (error) {
    formError.value = error instanceof ApiError
      ? (error.code === 10001 ? t('contact.errors.validation') : t(apiErrorKey(error.code)))
      : t('errorCodes.unknown')
  } finally {
    submitting.value = false
  }
}

function submitAnother() {
  submitted.value = false
  formError.value = ''
}
</script>

<template>
  <div>
    <UAlert
      v-if="submitted"
      color="success"
      variant="soft"
      icon="i-lucide-circle-check"
      :title="t('contact.successTitle')"
      :description="t('contact.successDescription')"
    >
      <template #actions>
        <UButton
          color="neutral"
          variant="outline"
          @click="submitAnother"
        >
          {{ t('contact.sendAnother') }}
        </UButton>
      </template>
    </UAlert>

    <form
      v-else
      class="flex flex-col gap-5"
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

      <!-- Honeypot: rendered and hidden off-screen (not display:none) so bots fill it. -->
      <div
        class="absolute -left-[9999px] top-auto h-px w-px overflow-hidden"
        aria-hidden="true"
      >
        <label for="contact-website">Website</label>
        <input
          id="contact-website"
          v-model="form.website"
          type="text"
          tabindex="-1"
          autocomplete="off"
          name="website"
        >
      </div>

      <div class="grid gap-5 sm:grid-cols-2">
        <UFormField
          :label="t('contact.fields.name')"
          :error="errors.name"
          required
        >
          <UInput
            v-model="form.name"
            class="w-full"
            autocomplete="name"
          />
        </UFormField>
        <UFormField
          :label="t('contact.fields.email')"
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
      </div>

      <UFormField :label="t('contact.fields.company')">
        <UInput
          v-model="form.company"
          class="w-full"
          autocomplete="organization"
        />
      </UFormField>

      <UFormField
        :label="t('contact.fields.message')"
        :error="errors.message"
        required
      >
        <UTextarea
          v-model="form.message"
          class="w-full"
          :rows="5"
        />
      </UFormField>

      <UFormField :error="errors.consent">
        <UCheckbox
          v-model="form.consent"
          :label="t('contact.fields.consent')"
        />
      </UFormField>

      <UButton
        type="submit"
        size="lg"
        :loading="submitting"
        :disabled="submitting"
        class="self-start"
      >
        {{ t('contact.submit') }}
      </UButton>
    </form>
  </div>
</template>
