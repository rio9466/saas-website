<script setup lang="ts">
import { computed, onMounted, reactive, ref } from "vue";
import { ElMessageBox } from "element-plus";
import { message } from "@/utils/message";
import { getErrorMessage } from "@/utils/error";
import {
  listHomeSectionsApi,
  createHomeSectionApi,
  updateHomeSectionApi,
  deleteHomeSectionApi
} from "@/api/homeSections";
import type { HomeSection } from "@/api/contract";
import { loadContentLocales, useContentLocales } from "../useContentLocales";
import { compactTranslations, isEmptyValue } from "../translations";
import type { TranslationFieldDef, TranslationMap } from "../types";
import LocaleTranslationTabs from "../components/LocaleTranslationTabs.vue";
import JsonTextarea from "../components/JsonTextarea.vue";
import MediaPicker from "../components/MediaPicker.vue";

defineOptions({ name: "ContentHome" });

const { locales } = useContentLocales();

const SECTION_TYPES = [
  "hero",
  "features",
  "screenshot",
  "stats",
  "cta"
] as const;
const typeOptions = [
  { value: "hero", label: "hero（首屏）" },
  { value: "features", label: "features（功能）" },
  { value: "screenshot", label: "screenshot（截图）" },
  { value: "stats", label: "stats（数据统计）" },
  { value: "cta", label: "cta（行动号召）" }
];

/** 非翻译基础数据字段（写入 data） */
const BASE_FIELDS: Record<
  string,
  Array<{ key: string; label: string; type?: "url" | "media" }>
> = {
  hero: [
    { key: "image_url", label: "主图", type: "media" },
    { key: "primary_cta_url", label: "主按钮链接" },
    { key: "secondary_cta_url", label: "次按钮链接" }
  ],
  screenshot: [{ key: "image_url", label: "截图", type: "media" }],
  cta: [{ key: "cta_url", label: "按钮链接" }]
};

/** 可翻译字段（写入 translations[locale]） */
const TRANSLATION_FIELDS: Record<string, TranslationFieldDef[]> = {
  hero: [
    { key: "title", label: "标题" },
    { key: "subtitle", label: "副标题", type: "textarea", rows: 3 },
    { key: "primary_cta_label", label: "主按钮文案" },
    { key: "secondary_cta_label", label: "次按钮文案" }
  ],
  features: [
    { key: "title", label: "标题" },
    { key: "subtitle", label: "副标题", type: "textarea", rows: 3 }
  ],
  screenshot: [
    { key: "title", label: "标题" },
    { key: "subtitle", label: "副标题", type: "textarea", rows: 3 }
  ],
  stats: [
    { key: "title", label: "标题" },
    {
      key: "items",
      label: "统计项",
      type: "kv-list",
      tip: "每行：数值|说明"
    }
  ],
  cta: [
    { key: "title", label: "标题" },
    { key: "subtitle", label: "正文", type: "textarea", rows: 3 },
    { key: "cta_label", label: "按钮文案" }
  ]
};

const loading = ref(false);
const list = ref<HomeSection[]>([]);
const total = ref(0);
const query = reactive({ page: 1, page_size: 20 });

async function loadList() {
  loading.value = true;
  try {
    const res = await listHomeSectionsApi(query);
    list.value = res?.data?.items ?? [];
    total.value = res?.data?.total ?? 0;
  } catch (error) {
    message(getErrorMessage(error), { type: "error" });
  } finally {
    loading.value = false;
  }
}

const dialogVisible = ref(false);
const dialogTitle = ref("新建首页区块");
const saving = ref(false);
const editingId = ref<string | null>(null);

const form = reactive({
  type: "hero",
  sort_order: 0,
  published: true,
  data: {} as Record<string, unknown>,
  translations: {} as TranslationMap
});

const isKnownType = computed(() =>
  (SECTION_TYPES as readonly string[]).includes(form.type)
);

const baseFields = computed(() => BASE_FIELDS[form.type] ?? []);
const translationFields = computed(() => TRANSLATION_FIELDS[form.type] ?? []);

function openCreate() {
  editingId.value = null;
  dialogTitle.value = "新建首页区块";
  Object.assign(form, {
    type: "hero",
    sort_order: 0,
    published: true,
    data: {},
    translations: {}
  });
  dialogVisible.value = true;
}

function openEdit(row: HomeSection) {
  editingId.value = row.id;
  dialogTitle.value = `编辑首页区块：${row.type}`;
  Object.assign(form, {
    type: row.type,
    sort_order: row.sort_order,
    published: row.published,
    data: JSON.parse(JSON.stringify(row.data ?? {})),
    translations: JSON.parse(JSON.stringify(row.translations ?? {}))
  });
  dialogVisible.value = true;
}

function getData(key: string): string {
  const value = form.data[key];
  return typeof value === "string" ? value : "";
}

function setData(key: string, value: unknown) {
  if (isEmptyValue(value)) {
    delete form.data[key];
  } else {
    form.data[key] = value;
  }
}

async function submit() {
  if (!form.type.trim()) {
    message("请选择区块类型", { type: "warning" });
    return;
  }
  saving.value = true;
  try {
    const payload = {
      type: form.type.trim(),
      sort_order: form.sort_order,
      published: form.published,
      data: form.data,
      translations: compactTranslations(form.translations)
    };
    if (editingId.value) {
      await updateHomeSectionApi(editingId.value, payload);
    } else {
      await createHomeSectionApi(payload);
    }
    message(editingId.value ? "保存成功" : "创建成功", { type: "success" });
    dialogVisible.value = false;
    loadList();
  } catch (error) {
    message(getErrorMessage(error), { type: "error" });
  } finally {
    saving.value = false;
  }
}

async function remove(row: HomeSection) {
  try {
    await ElMessageBox.confirm(
      "确定删除该首页区块吗？此操作不可撤销。",
      "删除确认",
      {
        type: "warning",
        confirmButtonText: "删除",
        cancelButtonText: "取消"
      }
    );
  } catch {
    return;
  }
  try {
    await deleteHomeSectionApi(row.id);
    message("已删除", { type: "success" });
    if (list.value.length === 1 && query.page > 1) query.page -= 1;
    loadList();
  } catch (error) {
    message(getErrorMessage(error), { type: "error" });
  }
}

function sectionTitle(row: HomeSection): string {
  const translated = Object.values(row.translations ?? {}).find(
    item => typeof item?.title === "string" && item.title
  );
  if (translated) return translated.title as string;
  const dataTitle = row.data?.title;
  return typeof dataTitle === "string" && dataTitle ? dataTitle : "-";
}

onMounted(async () => {
  try {
    await loadContentLocales();
  } catch {
    // 语言列表失败时回退，仍可编辑
  }
  loadList();
});
</script>

<template>
  <div class="flex flex-col gap-4">
    <div class="panel-container">
      <div class="panel-header">
        <span class="panel-title">首页区块</span>
        <div class="flex items-center gap-2">
          <el-button text :loading="loading" @click="loadList">刷新</el-button>
          <el-button
            v-perms="['admin.content.manage']"
            type="primary"
            plain
            @click="openCreate"
          >
            新建区块
          </el-button>
        </div>
      </div>
      <div class="panel-body table-body">
        <el-table v-loading="loading" :data="list" stripe>
          <el-table-column prop="id" label="ID" width="76" />
          <el-table-column prop="type" label="类型" min-width="120" />
          <el-table-column label="标题" min-width="180">
            <template #default="{ row }">{{ sectionTitle(row) }}</template>
          </el-table-column>
          <el-table-column
            prop="sort_order"
            label="排序"
            width="80"
            align="center"
          />
          <el-table-column label="状态" width="92">
            <template #default="{ row }">
              <el-tag :type="row.published ? 'success' : 'info'" size="small">
                {{ row.published ? "已发布" : "草稿" }}
              </el-tag>
            </template>
          </el-table-column>
          <el-table-column label="操作" width="150" fixed="right">
            <template #default="{ row }">
              <el-button
                v-perms="['admin.content.manage']"
                link
                type="primary"
                @click="openEdit(row)"
              >
                编辑
              </el-button>
              <el-button
                v-perms="['admin.content.manage']"
                link
                type="danger"
                @click="remove(row)"
              >
                删除
              </el-button>
            </template>
          </el-table-column>
        </el-table>
        <div class="flex justify-end pt-3">
          <el-pagination
            background
            layout="total, prev, pager, next"
            :total="total"
            :page-size="query.page_size"
            :current-page="query.page"
            @current-change="
              page => {
                query.page = page;
                loadList();
              }
            "
          />
        </div>
      </div>
    </div>

    <el-dialog
      v-model="dialogVisible"
      :title="dialogTitle"
      width="780px"
      destroy-on-close
    >
      <el-form :model="form" label-width="140px">
        <el-form-item label="类型">
          <el-select
            v-model="form.type"
            filterable
            allow-create
            default-first-option
            style="width: 240px"
          >
            <el-option
              v-for="option in typeOptions"
              :key="option.value"
              :label="option.label"
              :value="option.value"
            />
          </el-select>
        </el-form-item>
        <el-form-item label="排序">
          <el-input-number v-model="form.sort_order" :min="0" />
        </el-form-item>
        <el-form-item label="发布">
          <el-switch v-model="form.published" />
        </el-form-item>

        <template v-if="isKnownType">
          <el-form-item
            v-for="field in baseFields"
            :key="field.key"
            :label="field.label"
          >
            <MediaPicker
              v-if="field.type === 'media'"
              :model-value="getData(field.key)"
              @update:model-value="value => setData(field.key, value)"
            />
            <el-input
              v-else
              :model-value="getData(field.key)"
              placeholder="如 /register 或 https://..."
              @update:model-value="value => setData(field.key, value)"
            />
          </el-form-item>
        </template>
        <el-form-item v-else label="基础数据（JSON）">
          <JsonTextarea
            :model-value="form.data"
            :rows="6"
            @update:model-value="
              value => (form.data = (value as Record<string, unknown>) ?? {})
            "
          />
        </el-form-item>

        <LocaleTranslationTabs
          v-model:translations="form.translations"
          :locales="locales"
          :fields="translationFields"
          :raw-json="!isKnownType"
        />
      </el-form>
      <template #footer>
        <el-button @click="dialogVisible = false">取消</el-button>
        <el-button type="primary" :loading="saving" @click="submit"
          >保存</el-button
        >
      </template>
    </el-dialog>
  </div>
</template>

<style scoped>
@import url("@/style/business.scss");
</style>
