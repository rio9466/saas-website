export default defineAppConfig({
  ui: {
    colors: {
      primary: 'green',
      neutral: 'slate'
    }
  },
  // Brand design tokens consumed by app/layouts/default.vue as CSS custom
  // properties (radius, typography). Colors are handled by Nuxt UI colors above.
  design: {
    radius: '0.375rem',
    fontFamily: '\'Public Sans\', ui-sans-serif, system-ui, sans-serif'
  }
})
