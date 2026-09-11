<script setup lang="ts">
const props = withDefaults(defineProps<{
  content?: string
}>(), {
  content: ''
})

// `body_md` is parsed and sanitised by `renderMarkdown` before it reaches
// `v-html`; raw HTML is never rendered directly (contract §7.5).
const html = computed(() => renderMarkdown(props.content).html)
</script>

<template>
  <!-- eslint-disable vue/no-v-html -- `html` is sanitised by renderMarkdown -->
  <div
    class="markdown"
    v-html="html"
  />
  <!-- eslint-enable vue/no-v-html -->
</template>

<style scoped>
.markdown {
  color: var(--ui-text);
  line-height: 1.75;
  word-break: break-word;
}

.markdown :deep(> :first-child) {
  margin-top: 0;
}

.markdown :deep(h1),
.markdown :deep(h2),
.markdown :deep(h3),
.markdown :deep(h4) {
  color: var(--ui-text-highlighted);
  font-weight: 600;
  line-height: 1.3;
  margin: 1.75em 0 0.75em;
  scroll-margin-top: 5rem;
}

.markdown :deep(h1) { font-size: 1.75rem; }
.markdown :deep(h2) { font-size: 1.4rem; }
.markdown :deep(h3) { font-size: 1.15rem; }
.markdown :deep(h4) { font-size: 1rem; }

.markdown :deep(p),
.markdown :deep(ul),
.markdown :deep(ol),
.markdown :deep(blockquote),
.markdown :deep(pre),
.markdown :deep(table) {
  margin: 0.85em 0;
}

.markdown :deep(ul),
.markdown :deep(ol) {
  padding-left: 1.4em;
}

.markdown :deep(ul) { list-style: disc; }
.markdown :deep(ol) { list-style: decimal; }
.markdown :deep(li) { margin: 0.25em 0; }

.markdown :deep(a) {
  color: var(--ui-primary);
  text-decoration: underline;
  text-underline-offset: 2px;
}

.markdown :deep(blockquote) {
  border-left: 3px solid var(--ui-border);
  color: var(--ui-text-muted);
  padding-left: 1em;
}

.markdown :deep(code) {
  background: var(--ui-bg-muted);
  border-radius: 0.25rem;
  font-size: 0.875em;
  padding: 0.15em 0.4em;
}

.markdown :deep(pre) {
  background: var(--ui-bg-muted);
  border: 1px solid var(--ui-border);
  border-radius: 0.5rem;
  overflow-x: auto;
  padding: 1em;
}

.markdown :deep(pre code) {
  background: transparent;
  padding: 0;
}

.markdown :deep(hr) {
  border: 0;
  border-top: 1px solid var(--ui-border);
  margin: 2em 0;
}

.markdown :deep(img) {
  border-radius: 0.5rem;
  max-width: 100%;
}

.markdown :deep(table) {
  border-collapse: collapse;
  display: block;
  overflow-x: auto;
  width: 100%;
}

.markdown :deep(th),
.markdown :deep(td) {
  border: 1px solid var(--ui-border);
  padding: 0.5em 0.75em;
  text-align: left;
}
</style>
