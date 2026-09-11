<script setup lang="ts">
import type { PointTransactionPage } from '~/composables/useAuth'

definePageMeta({ layout: 'account', middleware: 'auth' })

const { t, locale } = useI18n()
const { user } = useAuth()
const api = useApi()

const page = ref(1)
const pageSize = 20
const data = ref<PointTransactionPage | null>(null)
const pending = ref(true)
const error = ref<unknown>(null)

/** Format a fixed-point decimal string with 4 fraction digits, never via float. */
function formatPoints(raw: string): string {
  const value = (raw || '').trim()
  const negative = value.startsWith('-')
  const unsigned = negative ? value.slice(1) : value
  const [intRaw = '0', fracRaw = ''] = unsigned.split('.')
  const intPart = intRaw.replace(/^0+(?=\d)/, '') || '0'
  let frac = fracRaw.replace(/\D.*$/, '')
  frac = frac.length >= 4 ? frac.slice(0, 4) : frac.padEnd(4, '0')
  return `${negative ? '-' : ''}${intPart}.${frac}`
}

function formatDateTime(value: string): string {
  if (!value) {
    return '—'
  }
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) {
    return value
  }
  return new Intl.DateTimeFormat(locale.value, { dateStyle: 'medium', timeStyle: 'short' }).format(date)
}

function deltaClass(delta: string): string {
  const value = (delta || '').trim()
  if (value.startsWith('-')) {
    return 'text-error'
  }
  if (/^0*(\.0*)?$/.test(value.replace(/^\+/, ''))) {
    return 'text-muted'
  }
  return 'text-success'
}

async function load() {
  pending.value = true
  error.value = null
  try {
    data.value = await api.get<PointTransactionPage>('/v1/me/point-transactions', {
      query: { page: page.value, page_size: pageSize }
    })
  } catch (err) {
    error.value = err
  } finally {
    pending.value = false
  }
}

const totalPages = computed(() => {
  if (!data.value) {
    return 1
  }
  return Math.max(1, Math.ceil(data.value.total / (data.value.page_size || pageSize)))
})

function changePage(next: number) {
  if (next < 1 || next > totalPages.value || next === page.value) {
    return
  }
  page.value = next
}

watch(page, load)
onMounted(load)

useSeo({ title: t('account.points.title') })
</script>

<template>
  <div>
    <UPageHeader
      :title="t('account.points.title')"
      :description="t('account.points.subtitle')"
    />

    <div
      v-if="user"
      class="mt-8 grid gap-4 sm:grid-cols-3"
    >
      <div class="rounded-lg border border-default p-4">
        <p class="text-sm text-muted">
          {{ t('account.points.balance') }}
        </p>
        <p class="mt-1 font-mono text-2xl font-semibold text-highlighted">
          {{ formatPoints(user.points_balance) }}
        </p>
      </div>
      <div class="rounded-lg border border-default p-4">
        <p class="text-sm text-muted">
          {{ t('account.points.consumption') }}
        </p>
        <p class="mt-1 font-mono text-2xl font-semibold text-highlighted">
          {{ formatPoints(user.consumption_points) }}
        </p>
      </div>
      <div class="rounded-lg border border-default p-4">
        <p class="text-sm text-muted">
          {{ t('account.points.level') }}
        </p>
        <p class="mt-1 text-2xl font-semibold text-highlighted">
          {{ user.level?.name || t('account.overview.noLevel') }}
        </p>
      </div>
    </div>

    <div class="mt-8">
      <AppLoading v-if="pending" />
      <AppErrorState
        v-else-if="error"
        :error="error"
        @retry="load"
      />
      <AppEmptyState
        v-else-if="!data?.items.length"
        :description="t('account.points.empty')"
        icon="i-lucide-coins"
      />
      <div
        v-else
        class="overflow-x-auto rounded-lg border border-default"
      >
        <table class="w-full text-left text-sm">
          <thead class="bg-elevated/50 text-muted">
            <tr>
              <th class="px-4 py-3 font-medium">
                {{ t('account.points.table.time') }}
              </th>
              <th class="px-4 py-3 font-medium">
                {{ t('account.points.table.reason') }}
              </th>
              <th class="px-4 py-3 text-right font-medium">
                {{ t('account.points.table.change') }}
              </th>
              <th class="px-4 py-3 text-right font-medium">
                {{ t('account.points.table.balance') }}
              </th>
              <th class="px-4 py-3 text-right font-medium">
                {{ t('account.points.table.consumptionChange') }}
              </th>
            </tr>
          </thead>
          <tbody>
            <tr
              v-for="item in data.items"
              :key="item.id"
              class="border-t border-default"
            >
              <td class="whitespace-nowrap px-4 py-3 text-muted">
                {{ formatDateTime(item.created_at) }}
              </td>
              <td class="px-4 py-3">
                {{ item.reason || '—' }}
              </td>
              <td
                class="px-4 py-3 text-right font-mono"
                :class="deltaClass(item.points_delta)"
              >
                {{ formatPoints(item.points_delta) }}
              </td>
              <td class="px-4 py-3 text-right font-mono">
                {{ formatPoints(item.balance_after) }}
              </td>
              <td class="px-4 py-3 text-right font-mono text-muted">
                {{ formatPoints(item.consumption_delta) }}
              </td>
            </tr>
          </tbody>
        </table>
      </div>

      <div
        v-if="data && data.total > 0"
        class="mt-4 flex items-center justify-between"
      >
        <p class="text-sm text-muted">
          {{ t('account.points.pageInfo', { page: data.page, total: totalPages }) }}
        </p>
        <div class="flex gap-2">
          <UButton
            color="neutral"
            variant="outline"
            size="sm"
            icon="i-lucide-chevron-left"
            :disabled="page <= 1"
            @click="changePage(page - 1)"
          >
            {{ t('account.points.previous') }}
          </UButton>
          <UButton
            color="neutral"
            variant="outline"
            size="sm"
            trailing-icon="i-lucide-chevron-right"
            :disabled="page >= totalPages"
            @click="changePage(page + 1)"
          >
            {{ t('account.points.next') }}
          </UButton>
        </div>
      </div>
    </div>
  </div>
</template>
