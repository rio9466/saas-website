<script setup lang="ts">
const { t } = useI18n()

const { data: homeData } = await useHomeContent()
const { data: featuresData } = await useFeaturesContent()

// Unknown section types are ignored for forward compatibility (contract §4.3).
const KNOWN_SECTION_TYPES = new Set(['hero', 'features', 'screenshot', 'stats', 'cta'])

const sections = computed(() => (homeData.value?.sections ?? []).filter(section => KNOWN_SECTION_TYPES.has(section.type)))
const features = computed(() => featuresData.value?.items ?? [])

useSeo()
</script>

<template>
  <div>
    <template v-if="sections.length">
      <HomeSection
        v-for="section in sections"
        :key="section.id"
        :section="section"
        :features="features"
      />
    </template>

    <!-- Fallback hero keeps the page renderable when no sections are published. -->
    <section
      v-else
      class="mx-auto max-w-4xl px-4 py-24 text-center sm:py-32"
    >
      <h1 class="text-4xl font-bold text-highlighted sm:text-5xl">
        {{ t('home.fallbackTitle') }}
      </h1>
      <p class="mx-auto mt-4 max-w-2xl text-lg text-muted">
        {{ t('home.fallbackDescription') }}
      </p>
      <div class="mt-8 flex flex-wrap items-center justify-center gap-3">
        <UButton
          to="/register"
          size="lg"
        >
          {{ t('home.fallbackPrimary') }}
        </UButton>
        <UButton
          to="/contact"
          size="lg"
          color="neutral"
          variant="outline"
        >
          {{ t('home.fallbackSecondary') }}
        </UButton>
      </div>
    </section>
  </div>
</template>
