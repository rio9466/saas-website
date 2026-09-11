<script setup lang="ts">
const { t } = useI18n()

const { data: docsData, error: docsError, refresh } = await useDocsContent()
const categories = computed(() => docsData.value?.categories ?? [])
const firstSlug = computed(() => firstDocArticle(categories.value)?.slug ?? '')

const { data: articleData } = await useDocContent(firstSlug)
const article = computed(() => articleData.value)

useSeo({ title: t('docs.title'), description: t('docs.subtitle') })
</script>

<template>
  <div>
    <DocsLayout
      v-if="article"
      :categories="categories"
      :article="article"
    />
    <div
      v-else
      class="mx-auto max-w-3xl px-4 py-16"
    >
      <AppErrorState
        v-if="docsError"
        :error="docsError"
        @retry="refresh()"
      />
      <AppEmptyState
        v-else
        :title="t('docs.empty')"
      />
    </div>
  </div>
</template>
