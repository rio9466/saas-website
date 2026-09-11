<script setup lang="ts">
import type { SeoOptions } from '~/composables/useSeo'

const route = useRoute()
const slug = computed(() => String(route.params.slug ?? ''))

const { data, error, refresh } = await usePageContent(slug)

if (error.value && isContentNotFound(error.value)) {
  throw createError({ statusCode: 404, statusMessage: 'Not Found', fatal: true })
}

const page = computed(() => data.value)

// Reactive SEO so client-side navigation between slugs updates title/description.
const seo = reactive<SeoOptions>({
  title: page.value?.seo_title || page.value?.title || '',
  description: page.value?.seo_description || ''
})
useSeo(seo)
watch(page, () => {
  seo.title = page.value?.seo_title || page.value?.title || ''
  seo.description = page.value?.seo_description || ''
})
</script>

<template>
  <div class="mx-auto max-w-3xl px-4 py-16">
    <AppErrorState
      v-if="error"
      :error="error"
      @retry="refresh()"
    />
    <article v-else-if="page">
      <h1 class="text-4xl font-bold text-highlighted">
        {{ page.title }}
      </h1>
      <MarkdownContent
        :content="page.body_md"
        class="mt-6"
      />
    </article>
    <AppLoading v-else />
  </div>
</template>
