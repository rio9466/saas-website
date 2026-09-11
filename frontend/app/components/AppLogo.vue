<script setup lang="ts">
const props = withDefaults(defineProps<{
  name?: string
  logo?: string
  logoDark?: string
}>(), {
  name: '',
  logo: '',
  logoDark: ''
})

const colorMode = useColorMode()

const source = computed(() => (colorMode.value === 'dark' && props.logoDark ? props.logoDark : props.logo))
</script>

<template>
  <span class="inline-flex items-center gap-2 font-semibold text-highlighted">
    <img
      v-if="source"
      :src="source"
      :alt="name"
      class="h-7 w-auto"
    >
    <UIcon
      v-else
      name="i-lucide-hexagon"
      class="size-7 text-primary"
    />
    <span v-if="name">{{ name }}</span>
  </span>
</template>
