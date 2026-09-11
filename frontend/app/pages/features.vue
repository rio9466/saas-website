<script setup lang="ts">
const { t } = useI18n()

const { data, error, refresh } = await useFeaturesContent()
const features = computed(() => data.value?.items ?? [])

useSeo({ title: t('features.title'), description: t('features.subtitle') })
</script>

<template>
  <div class="mx-auto max-w-6xl px-4 py-16">
    <UPageHeader
      :title="t('features.title')"
      :description="t('features.subtitle')"
    />

    <AppErrorState
      v-if="error && !features.length"
      class="mt-10"
      :error="error"
      @retry="refresh()"
    />
    <AppEmptyState
      v-else-if="!features.length"
      class="mt-10"
      :title="t('features.empty')"
    />
    <div
      v-else
      class="mt-10 grid gap-6 sm:grid-cols-2 lg:grid-cols-3"
    >
      <FeatureCard
        v-for="feature in features"
        :key="feature.id"
        :feature="feature"
      />
    </div>
  </div>
</template>
