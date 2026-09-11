<script setup lang="ts">
import type { PricingPlan } from '~/composables/useContent'

const props = defineProps<{
  plan: PricingPlan
  period: 'monthly' | 'yearly'
  fallbackCurrency?: string
}>()

const { t, locale } = useI18n()

const rawPrice = computed(() => (props.period === 'yearly' ? props.plan.yearly_price : props.plan.monthly_price))

const formattedPrice = computed(() => {
  const value = rawPrice.value
  if (!value) {
    return '—'
  }
  const amount = Number(value)
  if (!Number.isFinite(amount)) {
    return value
  }
  const currency = props.plan.currency || props.fallbackCurrency || 'USD'
  try {
    return new Intl.NumberFormat(locale.value, {
      style: 'currency',
      currency,
      maximumFractionDigits: 2
    }).format(amount)
  } catch {
    return `${amount} ${currency}`
  }
})

const periodLabel = computed(() => (props.period === 'yearly' ? t('pricing.perYear') : t('pricing.perMonth')))
</script>

<template>
  <article
    class="flex h-full flex-col gap-4 rounded-xl border bg-elevated p-6"
    :class="plan.highlighted ? 'border-primary shadow-lg' : 'border-default'"
  >
    <div class="flex items-start justify-between gap-2">
      <h3 class="text-lg font-semibold text-highlighted">
        {{ plan.name }}
      </h3>
      <UBadge
        v-if="plan.highlighted"
        color="primary"
        variant="soft"
      >
        {{ t('pricing.recommended') }}
      </UBadge>
    </div>

    <p
      v-if="plan.description"
      class="text-sm text-muted"
    >
      {{ plan.description }}
    </p>

    <p class="flex items-baseline gap-1">
      <span class="text-3xl font-semibold text-highlighted">{{ formattedPrice }}</span>
      <span class="text-sm text-muted">{{ periodLabel }}</span>
    </p>

    <ul
      v-if="plan.features?.length"
      class="flex flex-1 flex-col gap-2 text-sm text-muted"
    >
      <li
        v-for="(feature, index) in plan.features"
        :key="index"
        class="flex items-start gap-2"
      >
        <UIcon
          name="i-lucide-check"
          class="mt-0.5 size-4 shrink-0 text-primary"
        />
        <span>{{ feature }}</span>
      </li>
    </ul>

    <UButton
      v-if="plan.cta_url || plan.cta_label"
      :to="plan.cta_url || '/contact'"
      :color="plan.highlighted ? 'primary' : 'neutral'"
      :variant="plan.highlighted ? 'solid' : 'outline'"
      block
    >
      {{ plan.cta_label || t('pricing.contactUs') }}
    </UButton>
  </article>
</template>
