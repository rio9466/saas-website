<script setup lang="ts">
const { t } = useI18n()
const { settings } = useSiteSettings()
const navItems = useNavigation('footer')

const year = new Date().getFullYear()

const SOCIAL_ICONS: Record<string, string> = {
  github: 'i-simple-icons-github',
  x: 'i-simple-icons-x',
  twitter: 'i-simple-icons-x',
  linkedin: 'i-simple-icons-linkedin',
  youtube: 'i-simple-icons-youtube',
  facebook: 'i-simple-icons-facebook',
  weibo: 'i-simple-icons-sinaweibo',
  bilibili: 'i-simple-icons-bilibili'
}

function socialIcon(platform: string) {
  return SOCIAL_ICONS[platform.toLowerCase()] || 'i-lucide-link'
}

function targetOf(target: string) {
  return target || '_self'
}
</script>

<template>
  <UFooter>
    <template #left>
      <div class="flex flex-col gap-3">
        <AppLogo
          :name="settings.site_name"
          :logo="settings.logo_url"
          :logo-dark="settings.logo_dark_url"
        />
        <p
          v-if="settings.tagline"
          class="text-sm text-muted max-w-xs"
        >
          {{ settings.tagline }}
        </p>
      </div>
    </template>

    <nav
      class="flex flex-wrap items-center justify-center gap-x-4 gap-y-2"
      :aria-label="t('footer.navigation')"
    >
      <NuxtLink
        v-for="item in navItems"
        :key="item.id"
        :to="item.url"
        :target="targetOf(item.target)"
        class="text-sm text-muted hover:text-highlighted"
      >
        {{ item.label }}
      </NuxtLink>
    </nav>

    <template #right>
      <div class="flex flex-col gap-1 text-sm text-muted">
        <a
          v-if="settings.contact_email"
          :href="`mailto:${settings.contact_email}`"
        >
          {{ settings.contact_email }}
        </a>
        <a
          v-if="settings.contact_phone"
          :href="`tel:${settings.contact_phone}`"
        >
          {{ settings.contact_phone }}
        </a>
        <span v-if="settings.contact_address">{{ settings.contact_address }}</span>
        <div
          v-if="settings.social_links.length"
          class="flex items-center gap-1 pt-1"
        >
          <UButton
            v-for="link in settings.social_links"
            :key="link.platform"
            :to="link.url"
            target="_blank"
            rel="noopener noreferrer"
            color="neutral"
            variant="ghost"
            size="sm"
            :icon="socialIcon(link.platform)"
            :aria-label="link.platform"
          />
        </div>
      </div>
    </template>

    <template #bottom>
      <div class="flex flex-col items-center gap-1 text-center text-xs text-muted">
        <p>
          {{ settings.footer_text || `© ${year} ${settings.site_name}. ${t('footer.rights')}` }}
        </p>
        <p v-if="settings.icp_record">
          {{ t('footer.icp') }}: {{ settings.icp_record }}
        </p>
      </div>
    </template>
  </UFooter>
</template>
