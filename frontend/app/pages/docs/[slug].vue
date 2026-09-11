<script setup lang="ts">
import type { SeoOptions } from '~/composables/useSeo'

const route = useRoute()
const slug = computed(() => String(route.params.slug ?? ''))

const docsAsync = useDocsContent()
const articleAsync = useDocContent(slug)
await Promise.all([docsAsync, articleAsync])

const categories = computed(() => docsAsync.data.value?.categories ?? [])
const article = computed(() => articleAsync.data.value)
const { error: articleError, refresh } = articleAsync

if (articleError.value && isContentNotFound(articleError.value)) {
  throw createError({ statusCode: 404, statusMessage: 'Not Found', fatal: true })
}

// Reactive SEO so client-side navigation between articles updates the head.
const seo = reactive<SeoOptions>({
  title: article.value?.seo_title || article.value?.title || '',
  description: article.value?.seo_description || ''
})
useSeo(seo)
watch(article, () => {
  seo.title = article.value?.seo_title || article.value?.title || ''
  seo.description = article.value?.seo_description || ''
})
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
        :error="articleError"
        @retry="refresh()"
      />
    </div>
  </div>
</template>
