import { ref } from "vue";
import { getPublicSettingsApi } from "@/api/siteSettings";
import type { LocaleOption } from "@/api/contract";

/**
 * 已启用语言（supported_locales）来源：公开设置 `GET /api/v1/public/settings` 的
 * `locales`（仅启用语言，带默认语言）。管理端没有独立语言列表接口，故取公开设置。
 * 结果在模块级缓存；失败时按默认语言给出最小回退，不硬编码语言集合。
 */
const locales = ref<LocaleOption[]>([]);
let inflight: Promise<LocaleOption[]> | null = null;

function fallbackLocales(defaultLocale?: string): LocaleOption[] {
  const code = defaultLocale || "en";
  return [{ code, label: code }];
}

/** 读取已启用语言（首次触发请求，后续走缓存）。 */
export function loadContentLocales(): Promise<LocaleOption[]> {
  if (locales.value.length > 0) return Promise.resolve(locales.value);
  if (!inflight) {
    inflight = getPublicSettingsApi()
      .then(res => {
        const list = res?.data?.locales ?? [];
        locales.value = list.length
          ? list
          : fallbackLocales(res?.data?.default_locale);
        return locales.value;
      })
      .catch(error => {
        locales.value = fallbackLocales();
        throw error;
      })
      .finally(() => {
        inflight = null;
      });
  }
  return inflight;
}

export function useContentLocales() {
  return { locales, loadContentLocales };
}
