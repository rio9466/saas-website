<script setup lang="ts">
const { t } = useI18n()

const { data, error, refresh } = await usePricingContent()
const plans = computed(() => data.value?.plans ?? [])
const currency = computed(() => data.value?.currency ?? '')

type BillingPeriod = 'monthly' | 'yearly'
const period = ref<BillingPeriod>('monthly')
const billingOptions = computed<Array<{ label: string, value: BillingPeriod }>>(() => [
  { label: t('pricing.monthly'), value: 'monthly' },
  { label: t('pricing.yearly'), value: 'yearly' }
])

useSeo({ title: t('pricing.title'), description: t('pricing.subtitle') })
</script>

<template>
  <div class="mx-auto max-w-6xl px-4 py-16">
    <UPageHeader
      :title="t('pricing.title')"
      :description="t('pricing.subtitle')"
      class="text-center"
    />

    <div class="mt-8 flex justify-center">
      <div class="inline-flex rounded-lg border border-default bg-elevated p-1">
        <UButton
          v-for="option in billingOptions"
          :key="option.value"
          :color="period === option.value ? 'primary' : 'neutral'"
          :variant="period === option.value ? 'solid' : 'ghost'"
          size="sm"
          @click="period = option.value"
        >
          {{ option.label }}
        </UButton>
      </div>
    </div>

    <AppErrorState
      v-if="error && !plans.length"
      class="mt-10"
      :error="error"
      @retry="refresh()"
    />
    <AppEmptyState
      v-else-if="!plans.length"
      class="mt-10"
      :title="t('pricing.empty')"
    />
    <div
      v-else
      class="mt-10 grid items-stretch gap-6 lg:grid-cols-3"
    >
      <PricingPlanCard
        v-for="plan in plans"
        :key="plan.id"
        :plan="plan"
        :period="period"
        :fallback-currency="currency"
      />
    </div>
  </div>
</template>
