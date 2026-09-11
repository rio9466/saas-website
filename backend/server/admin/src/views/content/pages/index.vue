<script setup lang="ts">
import { computed, onMounted, reactive, ref } from "vue";
import { ElMessageBox } from "element-plus";
import { message } from "@/utils/message";
import { getErrorMessage } from "@/utils/error";
import {
  listPagesApi,
  getPageApi,
  createPageApi,
  updatePageApi,
  deletePageApi
} from "@/api/pages";
import type { ContentPage } from "@/api/contract";
import { loadContentLocales, useContentLocales } from "../useContentLocales";
import { compactTranslations } from "../translations";
import type { TranslationFieldDef, TranslationMap } from "../types";
import LocaleTranslationTabs from "../components/LocaleTranslationTabs.vue";

defineOptions({ name: "ContentPages" });

const { locales } = useContentLocales();

const loading = ref(false);
const list = ref<ContentPage[]>([]);
const total = ref(0);
const query = reactive({ page: 1, page_size: 20 });

const translationFields: TranslationFieldDef[] = [
  { key: "title", label: "标题" },
  { key: "body_md", label: "正文（Markdown）", type: "markdown" },
  { key: "seo_title", label: "SEO 标题" },
  { key: "seo_description", label: "SEO 描述", type: "textarea", rows: 3 }
];

async function loadList() {
  loading.value = true;
  try {
    const res = await listPagesApi(query);
    list.value = res?.data?.items ?? [];
    total.value = res?.data?.total ?? 0;
  } catch (error) {
    message(getErrorMessage(error), { type: "error" });
  } finally {
    loading.value = false;
  }
}

const dialogVisible = ref(false);
const dialogTitle = ref("新建页面");
const saving = ref(false);
const editingId = ref<string | null>(null);

const form = reactive({
  slug: "",
  published: true,
  sort_order: 0,
  translations: {} as TranslationMap
});

const titleMissing = computed(
  () => Object.keys(compactTranslations(form.translations)).length === 0
);

function openCreate() {
  editingId.value = null;
  dialogTitle.value = "新建页面";
  Object.assign(form, {
    slug: "",
    published: true,
    sort_order: 0,
    translations: {}
  });
  dialogVisible.value = true;
}

async function openEdit(row: ContentPage) {
  let detail = row;
  try {
    const res = await getPageApi(row.id);
    if (res?.data) detail = res.data;
  } catch (error) {
    message(getErrorMessage(error), { type: "error" });
    return;
  }
  editingId.value = detail.id;
  dialogTitle.value = `编辑页面：${detail.slug}`;
  Object.assign(form, {
    slug: detail.slug,
    published: detail.published,
    sort_order: detail.sort_order,
    translations: JSON.parse(JSON.stringify(detail.translations ?? {}))
  });
  dialogVisible.value = true;
}

async function submit() {
  if (!form.slug.trim()) {
    message("请填写页面 slug", { type: "warning" });
    return;
  }
  if (titleMissing.value) {
    message("请至少填写一种语言的标题", { type: "warning" });
    return;
  }
  saving.value = true;
  try {
    const payload = {
      slug: form.slug.trim(),
      published: form.published,
      sort_order: form.sort_order,
      translations: compactTranslations(form.translations)
    };
    if (editingId.value) {
      await updatePageApi(editingId.value, payload);
    } else {
      await createPageApi(payload);
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

async function remove(row: ContentPage) {
  try {
    await ElMessageBox.confirm(
      `确定删除页面「${row.slug}」吗？此操作不可撤销。`,
      "删除确认",
      { type: "warning", confirmButtonText: "删除", cancelButtonText: "取消" }
    );
  } catch {
    return;
  }
  try {
    await deletePageApi(row.id);
    message("已删除", { type: "success" });
    if (list.value.length === 1 && query.page > 1) query.page -= 1;
    loadList();
  } catch (error) {
    message(getErrorMessage(error), { type: "error" });
  }
}

function pageTitle(row: ContentPage): string {
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
    // 语言列表失败时回退，仍可编辑
  }
  loadList();
});
</script>

<template>
  <div class="flex flex-col gap-4">
    <div class="panel-container">
      <div class="panel-header">
        <span class="panel-title">页面</span>
        <div class="flex items-center gap-2">
          <el-button text :loading="loading" @click="loadList">刷新</el-button>
          <el-button
            v-perms="['admin.content.manage']"
            type="primary"
            plain
            @click="openCreate"
          >
            新建页面
          </el-button>
        </div>
      </div>
      <div class="panel-body table-body">
        <el-table v-loading="loading" :data="list" stripe>
          <el-table-column prop="slug" label="Slug" min-width="140" />
          <el-table-column label="标题" min-width="180">
            <template #default="{ row }">{{ pageTitle(row) }}</template>
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
      <el-form :model="form" label-width="120px">
        <el-form-item label="Slug">
          <el-input v-model="form.slug" placeholder="如 about" />
        </el-form-item>
        <el-form-item label="排序">
          <el-input-number v-model="form.sort_order" :min="0" />
        </el-form-item>
        <el-form-item label="发布">
          <el-switch v-model="form.published" />
        </el-form-item>
        <LocaleTranslationTabs
          v-model:translations="form.translations"
          :locales="locales"
          :fields="translationFields"
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
