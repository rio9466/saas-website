<script setup lang="ts">
import type { DocArticle, DocCategory } from '~/composables/useContent'

const props = defineProps<{
  categories: DocCategory[]
  article: DocArticle
}>()

const { t } = useI18n()

const toc = computed(() => renderMarkdown(props.article.body_md).toc.filter(link => link.level >= 2 && link.level <= 3))

const activeSlug = computed(() => props.article.slug)
const activeCategorySlug = computed(() => props.article.category?.slug ?? '')
</script>

<template>
  <div class="mx-auto grid max-w-7xl grid-cols-1 gap-10 px-4 py-10 lg:grid-cols-[16rem_minmax(0,1fr)] xl:grid-cols-[16rem_minmax(0,1fr)_14rem]">
    <!-- Sidebar -->
    <aside class="lg:sticky lg:top-24 lg:self-start">
      <nav
        class="flex flex-col gap-6"
        :aria-label="t('docs.navigation')"
      >
        <div
          v-for="category in categories"
          :key="category.id"
          class="flex flex-col gap-2"
        >
          <p
            class="text-xs font-semibold uppercase tracking-wide"
            :class="category.slug === activeCategorySlug ? 'text-primary' : 'text-dimmed'"
          >
            {{ category.name }}
          </p>
          <ul class="flex flex-col gap-0.5">
            <li
              v-for="item in category.articles"
              :key="item.id"
            >
              <NuxtLink
                :to="`/docs/${item.slug}`"
                class="block rounded-md px-3 py-1.5 text-sm transition-colors"
                :class="item.slug === activeSlug
                  ? 'bg-muted font-medium text-primary'
                  : 'text-muted hover:bg-muted hover:text-highlighted'"
              >
                {{ item.title }}
              </NuxtLink>
            </li>
          </ul>
        </div>
      </nav>
    </aside>

    <!-- Article -->
    <article class="min-w-0">
      <nav
        class="flex flex-wrap items-center gap-1 text-sm text-muted"
        :aria-label="t('docs.breadcrumb')"
      >
        <NuxtLink
          to="/docs"
          class="hover:text-highlighted"
        >
          {{ t('nav.docs') }}
        </NuxtLink>
        <span v-if="article.category?.name">/</span>
        <span v-if="article.category?.name">{{ article.category.name }}</span>
      </nav>

      <h1 class="mt-3 text-3xl font-bold text-highlighted">
        {{ article.title }}
      </h1>

      <MarkdownContent
        :content="article.body_md"
        class="mt-6"
      />

      <!-- Table of contents (mobile) -->
      <div
        v-if="toc.length"
        class="mt-10 rounded-lg border border-default bg-elevated p-4 xl:hidden"
      >
        <DocsToc :links="toc" />
      </div>

      <!-- Prev / next -->
      <nav
        v-if="article.prev || article.next"
        class="mt-12 grid gap-4 border-t border-default pt-6 sm:grid-cols-2"
        :aria-label="t('docs.surround')"
      >
        <NuxtLink
          v-if="article.prev"
          :to="`/docs/${article.prev.slug}`"
          class="flex flex-col gap-1 rounded-lg border border-default p-4 hover:border-primary"
        >
          <span class="text-xs text-dimmed">{{ t('docs.previous') }}</span>
          <span class="font-medium text-highlighted">{{ article.prev.title }}</span>
        </NuxtLink>
        <NuxtLink
          v-if="article.next"
          :to="`/docs/${article.next.slug}`"
          class="flex flex-col gap-1 rounded-lg border border-default p-4 hover:border-primary sm:col-start-2 sm:text-right"
        >
          <span class="text-xs text-dimmed">{{ t('docs.next') }}</span>
          <span class="font-medium text-highlighted">{{ article.next.title }}</span>
        </NuxtLink>
      </nav>
    </article>

    <!-- TOC (desktop) -->
    <aside
      v-if="toc.length"
      class="hidden xl:sticky xl:top-24 xl:block xl:self-start"
    >
      <DocsToc :links="toc" />
    </aside>
  </div>
</template>
