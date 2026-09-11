<script setup lang="ts">
import type { FeatureItem, HomeSection } from '~/composables/useContent'

const props = defineProps<{
  section: HomeSection
  features: FeatureItem[]
}>()

function text(key: string): string {
  const value = props.section.data?.[key]
  return typeof value === 'string' ? value : ''
}

function array(key: string): Array<Record<string, unknown>> {
  const value = props.section.data?.[key]
  return Array.isArray(value) ? (value as Array<Record<string, unknown>>) : []
}

function entryText(entry: Record<string, unknown>, key: string): string {
  const value = entry[key]
  return typeof value === 'string' ? value : ''
}

const localizedUrl = useLocalizedUrl()
const title = computed(() => text('title'))
const subtitle = computed(() => text('subtitle') || text('description') || text('body'))
const image = computed(() => text('image_url') || text('image'))
const primaryLabel = computed(() => text('primary_cta_label') || text('cta_label'))
const primaryUrl = computed(() => localizedUrl(text('primary_cta_url') || text('cta_url') || '/register'))
const secondaryLabel = computed(() => text('secondary_cta_label'))
const secondaryUrl = computed(() => localizedUrl(text('secondary_cta_url') || '/contact'))
const stats = computed(() => array('items').length ? array('items') : array('stats'))
</script>

<template>
  <!-- hero -->
  <section
    v-if="section.type === 'hero'"
    class="mx-auto max-w-5xl px-4 py-20 text-center sm:py-28"
  >
    <h1 class="text-4xl font-bold text-highlighted sm:text-5xl">
      {{ title }}
    </h1>
    <p
      v-if="subtitle"
      class="mx-auto mt-4 max-w-2xl text-lg text-muted"
    >
      {{ subtitle }}
    </p>
    <div
      v-if="primaryLabel || secondaryLabel"
      class="mt-8 flex flex-wrap items-center justify-center gap-3"
    >
      <UButton
        v-if="primaryLabel"
        :to="primaryUrl"
        size="lg"
      >
        {{ primaryLabel }}
      </UButton>
      <UButton
        v-if="secondaryLabel"
        :to="secondaryUrl"
        size="lg"
        color="neutral"
        variant="outline"
      >
        {{ secondaryLabel }}
      </UButton>
    </div>
    <img
      v-if="image"
      :src="image"
      :alt="title"
      class="mx-auto mt-12 w-full max-w-4xl rounded-xl"
    >
  </section>

  <!-- features -->
  <section
    v-else-if="section.type === 'features'"
    class="mx-auto max-w-6xl px-4 py-16"
  >
    <h2
      v-if="title"
      class="text-center text-3xl font-semibold text-highlighted"
    >
      {{ title }}
    </h2>
    <p
      v-if="subtitle"
      class="mx-auto mt-3 max-w-2xl text-center text-muted"
    >
      {{ subtitle }}
    </p>
    <div
      v-if="features.length"
      class="mt-10 grid gap-6 sm:grid-cols-2 lg:grid-cols-3"
    >
      <FeatureCard
        v-for="feature in features"
        :key="feature.id"
        :feature="feature"
      />
    </div>
  </section>

  <!-- screenshot -->
  <section
    v-else-if="section.type === 'screenshot'"
    class="mx-auto max-w-6xl px-4 py-16"
  >
    <h2
      v-if="title"
      class="text-center text-3xl font-semibold text-highlighted"
    >
      {{ title }}
    </h2>
    <p
      v-if="subtitle"
      class="mx-auto mt-3 max-w-2xl text-center text-muted"
    >
      {{ subtitle }}
    </p>
    <img
      v-if="image"
      :src="image"
      :alt="title"
      class="mx-auto mt-10 w-full rounded-xl border border-default"
      loading="lazy"
    >
  </section>

  <!-- stats -->
  <section
    v-else-if="section.type === 'stats'"
    class="mx-auto max-w-6xl px-4 py-16"
  >
    <h2
      v-if="title"
      class="text-center text-3xl font-semibold text-highlighted"
    >
      {{ title }}
    </h2>
    <dl
      v-if="stats.length"
      class="mt-10 grid gap-8 text-center sm:grid-cols-2 lg:grid-cols-4"
    >
      <div
        v-for="(stat, index) in stats"
        :key="index"
        class="flex flex-col gap-1"
      >
        <dt class="text-3xl font-bold text-primary">
          {{ entryText(stat, 'value') }}
        </dt>
        <dd class="text-sm text-muted">
          {{ entryText(stat, 'label') }}
        </dd>
      </div>
    </dl>
  </section>

  <!-- cta -->
  <section
    v-else-if="section.type === 'cta'"
    class="mx-auto max-w-6xl px-4 py-16"
  >
    <div class="rounded-2xl border border-default bg-elevated px-6 py-12 text-center">
      <h2 class="text-3xl font-semibold text-highlighted">
        {{ title }}
      </h2>
      <p
        v-if="subtitle"
        class="mx-auto mt-3 max-w-2xl text-muted"
      >
        {{ subtitle }}
      </p>
      <UButton
        v-if="primaryLabel"
        :to="primaryUrl"
        size="lg"
        class="mt-8"
      >
        {{ primaryLabel }}
      </UButton>
    </div>
  </section>
</template>
