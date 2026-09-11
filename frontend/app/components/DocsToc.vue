<script setup lang="ts">
import type { MarkdownHeading } from '~/composables/useMarkdown'

defineProps<{ links: MarkdownHeading[] }>()

const { t } = useI18n()
</script>

<template>
  <nav :aria-label="t('docs.tableOfContents')">
    <p class="mb-3 text-sm font-semibold text-highlighted">
      {{ t('docs.tableOfContents') }}
    </p>
    <ul class="flex flex-col gap-1 text-sm">
      <li
        v-for="link in links"
        :key="link.id"
        :class="link.level === 3 ? 'pl-3' : ''"
      >
        <a
          :href="`#${link.id}`"
          class="block text-muted hover:text-highlighted"
        >
          {{ link.text }}
        </a>
      </li>
    </ul>
  </nav>
</template>
