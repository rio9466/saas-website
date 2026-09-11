<script setup lang="ts">
import type { FeatureItem } from '~/composables/useContent'

const props = defineProps<{ feature: FeatureItem }>()

const icon = computed(() => props.feature.icon || 'i-lucide-sparkles')
</script>

<template>
  <article class="flex flex-col gap-3 rounded-lg border border-default bg-elevated p-6">
    <UIcon
      :name="icon"
      class="size-6 text-primary"
    />
    <h3 class="font-semibold text-highlighted">
      {{ feature.title }}
    </h3>
    <p
      v-if="feature.summary"
      class="text-sm text-muted"
    >
      {{ feature.summary }}
    </p>
    <img
      v-if="feature.image_url"
      :src="feature.image_url"
      :alt="feature.title"
      class="w-full rounded-md"
      loading="lazy"
    >
    <MarkdownContent
      v-if="feature.body_md"
      :content="feature.body_md"
      class="text-sm"
    />
  </article>
</template>
