<script setup lang="ts">
const { t } = useI18n()
const { settings } = useSiteSettings()

const hasContactInfo = computed(() => Boolean(
  settings.value.contact_email
  || settings.value.contact_phone
  || settings.value.contact_address
  || settings.value.social_links.length
))

useSeo({ title: t('contact.title'), description: t('contact.subtitle') })
</script>

<template>
  <div class="mx-auto max-w-5xl px-4 py-16">
    <UPageHeader
      :title="t('contact.title')"
      :description="t('contact.subtitle')"
    />

    <div class="mt-10 grid gap-10 lg:grid-cols-[minmax(0,1fr)_18rem]">
      <ContactForm />

      <aside
        v-if="hasContactInfo"
        class="flex flex-col gap-4"
      >
        <h2 class="text-sm font-semibold uppercase tracking-wide text-dimmed">
          {{ t('contact.infoTitle') }}
        </h2>

        <a
          v-if="settings.contact_email"
          :href="`mailto:${settings.contact_email}`"
          class="flex items-center gap-2 text-sm text-muted hover:text-highlighted"
        >
          <UIcon
            name="i-lucide-mail"
            class="size-4"
          />
          {{ settings.contact_email }}
        </a>

        <a
          v-if="settings.contact_phone"
          :href="`tel:${settings.contact_phone}`"
          class="flex items-center gap-2 text-sm text-muted hover:text-highlighted"
        >
          <UIcon
            name="i-lucide-phone"
            class="size-4"
          />
          {{ settings.contact_phone }}
        </a>

        <p
          v-if="settings.contact_address"
          class="flex items-start gap-2 text-sm text-muted"
        >
          <UIcon
            name="i-lucide-map-pin"
            class="mt-0.5 size-4 shrink-0"
          />
          {{ settings.contact_address }}
        </p>

        <div
          v-if="settings.social_links.length"
          class="flex flex-col gap-2 pt-2"
        >
          <a
            v-for="link in settings.social_links"
            :key="link.platform"
            :href="link.url"
            target="_blank"
            rel="noopener noreferrer"
            class="text-sm text-muted hover:text-highlighted"
          >
            {{ link.platform }}
          </a>
        </div>
      </aside>
    </div>
  </div>
</template>
