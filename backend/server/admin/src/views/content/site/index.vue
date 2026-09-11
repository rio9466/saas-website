<script setup lang="ts">
import { onMounted, reactive, ref } from "vue";
import { message } from "@/utils/message";
import { getErrorMessage, getErrorCode } from "@/utils/error";
import { getSiteSettingsApi, updateSiteSettingsApi } from "@/api/siteSettings";
import type { SiteSettings } from "@/api/contract";
import { loadContentLocales, useContentLocales } from "../useContentLocales";
import { compactTranslations } from "../translations";
import type { TranslationFieldDef, TranslationMap } from "../types";
import LocaleTranslationTabs from "../components/LocaleTranslationTabs.vue";
import MediaPicker from "../components/MediaPicker.vue";

defineOptions({ name: "ContentSiteSettings" });

const { locales } = useContentLocales();

const translationFields: TranslationFieldDef[] = [
  { key: "tagline", label: "标语" },
  { key: "footer_text", label: "页脚文案", type: "textarea", rows: 3 },
  { key: "seo_default_title", label: "默认 SEO 标题" },
  {
    key: "seo_default_description",
    label: "默认 SEO 描述",
    type: "textarea",
    rows: 3
  },
  { key: "icp_record", label: "备案号" }
];

const loading = ref(false);
const saving = ref(false);

const form = reactive({
  site_name: "",
  logo_url: "",
  logo_dark_url: "",
  favicon_url: "",
  contact_email: "",
  contact_phone: "",
  contact_address: "",
  seo_default_og_image_url: "",
  default_locale: "",
  social_links: [] as Array<{ platform: string; url: string }>,
  translations: {} as TranslationMap,
  version: 0
});

async function loadSettings() {
  loading.value = true;
  try {
    const res = await getSiteSettingsApi();
    const s = res?.data;
    if (!s) return;
    Object.assign(form, {
      site_name: s.site_name,
      logo_url: s.logo_url ?? "",
      logo_dark_url: s.logo_dark_url ?? "",
      favicon_url: s.favicon_url ?? "",
      contact_email: s.contact_email ?? "",
      contact_phone: s.contact_phone ?? "",
      contact_address: s.contact_address ?? "",
      seo_default_og_image_url: s.seo_default_og_image_url ?? "",
      default_locale: s.default_locale,
      social_links: (s.social_links ?? []).map(link => ({
        platform: link.platform ?? "",
        url: link.url ?? ""
      })),
      translations: JSON.parse(JSON.stringify(s.translations ?? {})),
      version: s.version
    });
  } catch (error) {
    message(getErrorMessage(error), { type: "error" });
  } finally {
    loading.value = false;
  }
}

function addSocialLink() {
  form.social_links.push({ platform: "", url: "" });
}

function removeSocialLink(index: number) {
  form.social_links.splice(index, 1);
}

function compactSocialLinks(): Array<{ platform: string; url: string }> {
  return form.social_links
    .map(link => ({ platform: link.platform.trim(), url: link.url.trim() }))
    .filter(link => link.platform || link.url);
}

async function save() {
  if (!form.site_name.trim()) {
    message("请填写站点名称", { type: "warning" });
    return;
  }
  if (!form.default_locale) {
    message("请选择默认语言", { type: "warning" });
    return;
  }
  saving.value = true;
  try {
    const res = await updateSiteSettingsApi({
      site_name: form.site_name.trim(),
      logo_url: form.logo_url,
      logo_dark_url: form.logo_dark_url,
      favicon_url: form.favicon_url,
      contact_email: form.contact_email,
      contact_phone: form.contact_phone,
      contact_address: form.contact_address,
      social_links: compactSocialLinks(),
      seo_default_og_image_url: form.seo_default_og_image_url,
      default_locale: form.default_locale,
      translations: compactTranslations(form.translations),
      version: form.version
    });
    const s = res?.data;
    if (s) {
      form.version = s.version;
      form.translations = JSON.parse(JSON.stringify(s.translations ?? {}));
    }
    message("设置已保存", { type: "success" });
  } catch (error) {
    if (getErrorCode(error) === 40013) {
      message("内容已被他人修改，请刷新后重试", { type: "warning" });
    } else {
      message(getErrorMessage(error), { type: "error" });
    }
  } finally {
    saving.value = false;
  }
}

onMounted(async () => {
  try {
    await loadContentLocales();
  } catch {
    // 语言列表失败时回退，仍可编辑
  }
  await loadSettings();
});
</script>

<template>
  <div class="flex flex-col gap-4">
    <div v-loading="loading" class="panel-container">
      <div class="panel-header">
        <span class="panel-title">站点设置</span>
        <div class="flex items-center gap-2">
          <span class="text-xs text-secondary">版本 {{ form.version }}</span>
          <el-button text :loading="loading" @click="loadSettings"
            >重新载入</el-button
          >
        </div>
      </div>
      <div class="panel-body">
        <el-form :model="form" label-width="150px" class="site-form">
          <el-form-item label="站点名称">
            <el-input v-model="form.site_name" />
          </el-form-item>
          <el-form-item label="Logo（浅色）">
            <MediaPicker v-model="form.logo_url" />
          </el-form-item>
          <el-form-item label="Logo（深色）">
            <MediaPicker v-model="form.logo_dark_url" />
          </el-form-item>
          <el-form-item label="Favicon">
            <MediaPicker v-model="form.favicon_url" />
          </el-form-item>
          <el-form-item label="默认 OG 图片">
            <MediaPicker v-model="form.seo_default_og_image_url" />
          </el-form-item>
          <el-form-item label="联系邮箱">
            <el-input v-model="form.contact_email" />
          </el-form-item>
          <el-form-item label="联系电话">
            <el-input v-model="form.contact_phone" />
          </el-form-item>
          <el-form-item label="联系地址">
            <el-input v-model="form.contact_address" />
          </el-form-item>
          <el-form-item label="默认语言">
            <el-select v-model="form.default_locale" style="width: 220px">
              <el-option
                v-for="loc in locales"
                :key="loc.code"
                :label="`${loc.label}（${loc.code}）`"
                :value="loc.code"
              />
            </el-select>
          </el-form-item>

          <el-form-item label="社交链接">
            <div class="social-list">
              <div
                v-for="(link, index) in form.social_links"
                :key="index"
                class="social-row"
              >
                <el-input
                  v-model="link.platform"
                  placeholder="平台，如 github"
                  style="width: 180px"
                />
                <el-input
                  v-model="link.url"
                  placeholder="https://..."
                  style="flex: 1"
                />
                <el-button link type="danger" @click="removeSocialLink(index)"
                  >移除</el-button
                >
              </div>
              <el-button link type="primary" @click="addSocialLink"
                >添加社交链接</el-button
              >
            </div>
          </el-form-item>

          <el-form-item label="多语言内容">
            <LocaleTranslationTabs
              v-model:translations="form.translations"
              :locales="locales"
              :fields="translationFields"
            />
          </el-form-item>

          <el-form-item>
            <el-button type="primary" :loading="saving" @click="save"
              >保存设置</el-button
            >
            <el-button @click="loadSettings">放弃修改</el-button>
          </el-form-item>
        </el-form>
      </div>
    </div>
  </div>
</template>

<style scoped>
@import url("@/style/business.scss");

.site-form {
  max-width: 760px;
}

.social-list {
  display: flex;
  flex-direction: column;
  gap: 8px;
  width: 100%;
}

.social-row {
  display: flex;
  gap: 8px;
  align-items: center;
}
</style>
