<script setup lang="ts">
// `NuxtLinkLocale` localizes same-site paths and leaves external URLs
// untouched, so this is a drop-in replacement for `<NuxtLink>` everywhere.
defineOptions({ inheritAttrs: false })

const props = defineProps<{ to: string }>()
</script>

<template>
  <NuxtLinkLocale
    v-bind="$attrs"
    :to="props.to"
  >
    <slot />
  </NuxtLinkLocale>
</template>
