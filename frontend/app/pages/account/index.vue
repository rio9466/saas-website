<script setup lang="ts">
definePageMeta({ layout: 'account', middleware: 'auth' })

const { t, locale } = useI18n()
const { user } = useAuth()

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

useSeo({ title: t('account.overview.title') })
</script>

<template>
  <div v-if="user">
    <UPageHeader
      :title="t('account.overview.title')"
      :description="t('account.overview.subtitle')"
    />

    <div class="mt-8 flex items-center gap-4">
      <UAvatar
        :src="user.avatar_url || undefined"
        :alt="user.nickname || user.username"
        icon="i-lucide-user"
        size="xl"
      />
      <div>
        <p class="text-lg font-semibold text-highlighted">
          {{ user.nickname || user.username }}
        </p>
        <p class="text-sm text-muted">
          {{ user.email }}
        </p>
      </div>
    </div>

    <dl class="mt-8 grid gap-4 sm:grid-cols-2">
      <div class="rounded-lg border border-default p-4">
        <dt class="text-sm text-muted">
          {{ t('account.overview.username') }}
        </dt>
        <dd class="mt-1 font-medium text-highlighted">
          {{ user.username }}
        </dd>
      </div>
      <div class="rounded-lg border border-default p-4">
        <dt class="text-sm text-muted">
          {{ t('account.overview.email') }}
        </dt>
        <dd class="mt-1 flex items-center gap-2 font-medium text-highlighted">
          <span class="truncate">{{ user.email }}</span>
          <UBadge
            v-if="user.status === 'pending_verification'"
            color="warning"
            variant="soft"
            size="sm"
          >
            {{ t('account.overview.unverified') }}
          </UBadge>
        </dd>
      </div>
      <div class="rounded-lg border border-default p-4">
        <dt class="text-sm text-muted">
          {{ t('account.overview.level') }}
        </dt>
        <dd class="mt-1 font-medium text-highlighted">
          {{ user.level?.name || t('account.overview.noLevel') }}
        </dd>
      </div>
      <div class="rounded-lg border border-default p-4">
        <dt class="text-sm text-muted">
          {{ t('account.overview.joinedAt') }}
        </dt>
        <dd class="mt-1 font-medium text-highlighted">
          {{ formatDateTime(user.created_at) }}
        </dd>
      </div>
      <div class="rounded-lg border border-default p-4">
        <dt class="text-sm text-muted">
          {{ t('account.overview.pointsBalance') }}
        </dt>
        <dd class="mt-1 font-mono text-lg font-semibold text-highlighted">
          {{ formatPoints(user.points_balance) }}
        </dd>
      </div>
      <div class="rounded-lg border border-default p-4">
        <dt class="text-sm text-muted">
          {{ t('account.overview.consumption') }}
        </dt>
        <dd class="mt-1 font-mono text-lg font-semibold text-highlighted">
          {{ formatPoints(user.consumption_points) }}
        </dd>
      </div>
    </dl>
  </div>
</template>
