<script setup lang="ts">
const appConfig = useAppConfig()

// Apply site-wide SEO defaults for every page in this layout. Pages can call
// `useSeo({ title, description })` to override them.
useSeo()

// Expose brand design tokens as CSS custom properties for every page in this layout.
const designTokens = computed(() => ({
  '--ui-radius': appConfig.design.radius,
  '--font-sans': appConfig.design.fontFamily
}))
</script>

<template>
  <div
    class="flex min-h-screen flex-col"
    :style="designTokens"
  >
    <AppHeader />
    <UMain class="flex-1">
      <slot />
    </UMain>
    <AppFooter />
  </div>
</template>
