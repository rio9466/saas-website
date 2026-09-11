<script setup lang="ts">
import { computed, onMounted, reactive, ref } from "vue";
import { ElMessageBox } from "element-plus";
import { message } from "@/utils/message";
import { getErrorMessage } from "@/utils/error";
import {
  listFeaturesApi,
  createFeatureApi,
  updateFeatureApi,
  deleteFeatureApi
} from "@/api/features";
import type { Feature } from "@/api/contract";
import { loadContentLocales, useContentLocales } from "../useContentLocales";
import { compactTranslations } from "../translations";
import type { TranslationFieldDef, TranslationMap } from "../types";
import LocaleTranslationTabs from "../components/LocaleTranslationTabs.vue";
import type { FormInstance } from "element-plus";

defineOptions({ name: "ContentFeatures" });

const { locales } = useContentLocales();

const loading = ref(false);
const list = ref<Feature[]>([]);
const total = ref(0);
const query = reactive({ page: 1, page_size: 20 });

const translationFields: TranslationFieldDef[] = [
  { key: "title", label: "标题" },
  { key: "summary", label: "摘要", type: "textarea", rows: 3 },
  { key: "body_md", label: "正文（Markdown）", type: "textarea", rows: 8 },
  {
    key: "image_url",
    label: "配图",
    type: "media",
    tip: "从媒体库选择或上传；仅作为内容图片引用"
  }
];

async function loadList() {
  loading.value = true;
  try {
    const res = await listFeaturesApi(query);
    list.value = res?.data?.items ?? [];
    total.value = res?.data?.total ?? 0;
  } catch (error) {
    message(getErrorMessage(error), { type: "error" });
  } finally {
    loading.value = false;
  }
}

const dialogVisible = ref(false);
const dialogTitle = ref("新建功能");
const saving = ref(false);
const formRef = ref<FormInstance>();
const editingId = ref<string | null>(null);

const form = reactive({
  icon: "",
  sort_order: 0,
  published: true,
  translations: {} as TranslationMap
});

const titleMissing = computed(
  () => Object.keys(compactTranslations(form.translations)).length === 0
);

function openCreate() {
  editingId.value = null;
  dialogTitle.value = "新建功能";
  Object.assign(form, {
    icon: "",
    sort_order: 0,
    published: true,
    translations: {}
  });
  dialogVisible.value = true;
}

function openEdit(row: Feature) {
  editingId.value = row.id;
  dialogTitle.value = "编辑功能";
  Object.assign(form, {
    icon: row.icon,
    sort_order: row.sort_order,
    published: row.published,
    translations: JSON.parse(JSON.stringify(row.translations ?? {}))
  });
  dialogVisible.value = true;
}

async function submit() {
  if (titleMissing.value) {
    message("请至少填写一种语言的标题", { type: "warning" });
    return;
  }
  saving.value = true;
  try {
    const payload = {
      icon: form.icon,
      sort_order: form.sort_order,
      published: form.published,
      translations: compactTranslations(form.translations)
    };
    if (editingId.value) {
      await updateFeatureApi(editingId.value, payload);
    } else {
      await createFeatureApi(payload);
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

async function remove(row: Feature) {
  try {
    await ElMessageBox.confirm(
      `确定删除该功能条目吗？此操作不可撤销。`,
      "删除确认",
      { type: "warning", confirmButtonText: "删除", cancelButtonText: "取消" }
    );
  } catch {
    return;
  }
  try {
    await deleteFeatureApi(row.id);
    message("已删除", { type: "success" });
    if (list.value.length === 1 && query.page > 1) query.page -= 1;
    loadList();
  } catch (error) {
    message(getErrorMessage(error), { type: "error" });
  }
}

function primaryTitle(row: Feature): string {
  return (
    row.translations?.["zh-CN"]?.title ||
    row.translations?.["en"]?.title ||
    Object.values(row.translations ?? {})[0]?.title ||
    "-"
  );
}

onMounted(async () => {
  try {
    await loadContentLocales();
  } catch {
    // 语言列表失败时用回退语言，仍可编辑
  }
  loadList();
});
</script>

<template>
  <div class="flex flex-col gap-4">
    <div class="panel-container">
      <div class="panel-header">
        <span class="panel-title">功能条目</span>
        <div class="flex items-center gap-2">
          <el-button text :loading="loading" @click="loadList">刷新</el-button>
          <el-button
            v-perms="['admin.content.manage']"
            type="primary"
            plain
            @click="openCreate"
          >
            新建功能
          </el-button>
        </div>
      </div>
      <div class="panel-body table-body">
        <el-table v-loading="loading" :data="list" stripe>
          <el-table-column prop="id" label="ID" width="76" />
          <el-table-column label="标题" min-width="160">
            <template #default="{ row }">{{ primaryTitle(row) }}</template>
          </el-table-column>
          <el-table-column prop="icon" label="图标" min-width="140" />
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
      width="720px"
      destroy-on-close
    >
      <el-form ref="formRef" :model="form" label-width="120px">
        <el-form-item label="图标">
          <el-input v-model="form.icon" placeholder="如 i-lucide-zap" />
        </el-form-item>
        <el-form-item label="排序">
          <el-input-number v-model="form.sort_order" :min="0" />
        </el-form-item>
        <el-form-item label="发布">
          <el-switch v-model="form.published" />
        </el-form-item>
        <el-form-item label="多语言内容">
          <LocaleTranslationTabs
            v-model:translations="form.translations"
            :locales="locales"
            :fields="translationFields"
          />
        </el-form-item>
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
