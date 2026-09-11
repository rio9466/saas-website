<script setup lang="ts">
import { computed, onMounted, reactive, ref } from "vue";
import { ElMessageBox } from "element-plus";
import { message } from "@/utils/message";
import { getErrorMessage } from "@/utils/error";
import {
  listDocCategoriesApi,
  createDocCategoryApi,
  updateDocCategoryApi,
  deleteDocCategoryApi,
  listDocArticlesApi,
  getDocArticleApi,
  createDocArticleApi,
  updateDocArticleApi,
  deleteDocArticleApi
} from "@/api/docs";
import type { DocArticle, DocCategory } from "@/api/contract";
import { loadContentLocales, useContentLocales } from "../useContentLocales";
import { compactTranslations } from "../translations";
import type { TranslationFieldDef, TranslationMap } from "../types";
import LocaleTranslationTabs from "../components/LocaleTranslationTabs.vue";

defineOptions({ name: "ContentDocs" });

const { locales } = useContentLocales();

const activeTab = ref("articles");

// ---------- 分类 ----------
const categoryLoading = ref(false);
const categories = ref<DocCategory[]>([]);

const categoryFields: TranslationFieldDef[] = [
  { key: "name", label: "分类名称" }
];

async function loadCategories() {
  categoryLoading.value = true;
  try {
    const res = await listDocCategoriesApi({ page: 1, page_size: 100 });
    categories.value = res?.data?.items ?? [];
  } catch (error) {
    message(getErrorMessage(error), { type: "error" });
  } finally {
    categoryLoading.value = false;
  }
}

const categoryDialogVisible = ref(false);
const categoryDialogTitle = ref("新建分类");
const categorySaving = ref(false);
const editingCategoryId = ref<string | null>(null);
const categoryForm = reactive({
  slug: "",
  sort_order: 0,
  translations: {} as TranslationMap
});

const categoryNameMissing = computed(
  () => Object.keys(compactTranslations(categoryForm.translations)).length === 0
);

function categoryName(row: DocCategory): string {
  return (
    row.translations?.["zh-CN"]?.name ||
    row.translations?.["en"]?.name ||
    Object.values(row.translations ?? {})[0]?.name ||
    row.slug
  );
}

function openCreateCategory() {
  editingCategoryId.value = null;
  categoryDialogTitle.value = "新建文档分类";
  Object.assign(categoryForm, { slug: "", sort_order: 0, translations: {} });
  categoryDialogVisible.value = true;
}

function openEditCategory(row: DocCategory) {
  editingCategoryId.value = row.id;
  categoryDialogTitle.value = `编辑分类：${categoryName(row)}`;
  Object.assign(categoryForm, {
    slug: row.slug,
    sort_order: row.sort_order,
    translations: JSON.parse(JSON.stringify(row.translations ?? {}))
  });
  categoryDialogVisible.value = true;
}

async function submitCategory() {
  if (!categoryForm.slug.trim()) {
    message("请填写分类 slug", { type: "warning" });
    return;
  }
  if (categoryNameMissing.value) {
    message("请至少填写一种语言的分类名称", { type: "warning" });
    return;
  }
  categorySaving.value = true;
  try {
    const payload = {
      slug: categoryForm.slug.trim(),
      sort_order: categoryForm.sort_order,
      translations: compactTranslations(categoryForm.translations)
    };
    if (editingCategoryId.value) {
      await updateDocCategoryApi(editingCategoryId.value, payload);
    } else {
      await createDocCategoryApi(payload);
    }
    message(editingCategoryId.value ? "保存成功" : "创建成功", {
      type: "success"
    });
    categoryDialogVisible.value = false;
    loadCategories();
  } catch (error) {
    message(getErrorMessage(error), { type: "error" });
  } finally {
    categorySaving.value = false;
  }
}

async function removeCategory(row: DocCategory) {
  try {
    await ElMessageBox.confirm(
      `确定删除分类「${categoryName(row)}」吗？其下文章将一并删除。`,
      "删除确认",
      { type: "warning", confirmButtonText: "删除", cancelButtonText: "取消" }
    );
  } catch {
    return;
  }
  try {
    await deleteDocCategoryApi(row.id);
    message("已删除", { type: "success" });
    loadCategories();
  } catch (error) {
    message(getErrorMessage(error), { type: "error" });
  }
}

// ---------- 文章 ----------
const articleLoading = ref(false);
const articles = ref<DocArticle[]>([]);
const articleTotal = ref(0);
const articleQuery = reactive({
  page: 1,
  page_size: 20
});

const articleFields: TranslationFieldDef[] = [
  { key: "title", label: "标题" },
  { key: "body_md", label: "正文（Markdown）", type: "textarea", rows: 10 },
  { key: "seo_title", label: "SEO 标题" },
  { key: "seo_description", label: "SEO 描述", type: "textarea", rows: 3 }
];

async function loadArticles() {
  articleLoading.value = true;
  try {
    const res = await listDocArticlesApi(articleQuery);
    articles.value = res?.data?.items ?? [];
    articleTotal.value = res?.data?.total ?? 0;
  } catch (error) {
    message(getErrorMessage(error), { type: "error" });
  } finally {
    articleLoading.value = false;
  }
}

const articleDialogVisible = ref(false);
const articleDialogTitle = ref("新建文章");
const articleSaving = ref(false);
const editingArticleId = ref<string | null>(null);
const articleForm = reactive({
  slug: "",
  category_id: "",
  sort_order: 0,
  published: true,
  translations: {} as TranslationMap
});

const categoryNameMap = computed(() => {
  const map: Record<string, string> = {};
  for (const category of categories.value) {
    map[category.id] = categoryName(category);
  }
  return map;
});

const articleTitleMissing = computed(
  () => Object.keys(compactTranslations(articleForm.translations)).length === 0
);

function articleTitle(row: DocArticle): string {
  return (
    row.translations?.["zh-CN"]?.title ||
    row.translations?.["en"]?.title ||
    Object.values(row.translations ?? {})[0]?.title ||
    row.slug
  );
}

function openCreateArticle() {
  editingArticleId.value = null;
  articleDialogTitle.value = "新建文档文章";
  Object.assign(articleForm, {
    slug: "",
    category_id: categories.value[0]?.id || "",
    sort_order: 0,
    published: true,
    translations: {}
  });
  articleDialogVisible.value = true;
}

async function openEditArticle(row: DocArticle) {
  let detail = row;
  try {
    const res = await getDocArticleApi(row.id);
    if (res?.data) detail = res.data;
  } catch (error) {
    message(getErrorMessage(error), { type: "error" });
    return;
  }
  editingArticleId.value = detail.id;
  articleDialogTitle.value = `编辑文章：${articleTitle(detail)}`;
  Object.assign(articleForm, {
    slug: detail.slug,
    category_id: detail.category_id,
    sort_order: detail.sort_order,
    published: detail.published,
    translations: JSON.parse(JSON.stringify(detail.translations ?? {}))
  });
  articleDialogVisible.value = true;
}

async function submitArticle() {
  if (!articleForm.slug.trim()) {
    message("请填写文章 slug", { type: "warning" });
    return;
  }
  if (!articleForm.category_id) {
    message("请选择所属分类", { type: "warning" });
    return;
  }
  if (articleTitleMissing.value) {
    message("请至少填写一种语言的标题", { type: "warning" });
    return;
  }
  articleSaving.value = true;
  try {
    const payload = {
      slug: articleForm.slug.trim(),
      category_id: articleForm.category_id,
      sort_order: articleForm.sort_order,
      published: articleForm.published,
      translations: compactTranslations(articleForm.translations)
    };
    if (editingArticleId.value) {
      await updateDocArticleApi(editingArticleId.value, payload);
    } else {
      await createDocArticleApi(payload);
    }
    message(editingArticleId.value ? "保存成功" : "创建成功", {
      type: "success"
    });
    articleDialogVisible.value = false;
    loadArticles();
  } catch (error) {
    message(getErrorMessage(error), { type: "error" });
  } finally {
    articleSaving.value = false;
  }
}

async function removeArticle(row: DocArticle) {
  try {
    await ElMessageBox.confirm(
      `确定删除文章「${articleTitle(row)}」吗？此操作不可撤销。`,
      "删除确认",
      { type: "warning", confirmButtonText: "删除", cancelButtonText: "取消" }
    );
  } catch {
    return;
  }
  try {
    await deleteDocArticleApi(row.id);
    message("已删除", { type: "success" });
    if (articles.value.length === 1 && articleQuery.page > 1)
      articleQuery.page -= 1;
    loadArticles();
  } catch (error) {
    message(getErrorMessage(error), { type: "error" });
  }
}

onMounted(async () => {
  try {
    await loadContentLocales();
  } catch {
    // 语言列表失败时回退，仍可编辑
  }
  await loadCategories();
  loadArticles();
});
</script>

<template>
  <div class="flex flex-col gap-4">
    <div class="panel-container">
      <div class="panel-header">
        <span class="panel-title">文档中心</span>
        <div class="flex items-center gap-2">
          <el-button
            v-if="activeTab === 'articles'"
            v-perms="['admin.content.manage']"
            type="primary"
            plain
            @click="openCreateArticle"
          >
            新建文章
          </el-button>
          <el-button
            v-else
            v-perms="['admin.content.manage']"
            type="primary"
            plain
            @click="openCreateCategory"
          >
            新建分类
          </el-button>
        </div>
      </div>
      <div class="panel-body">
        <el-tabs v-model="activeTab">
          <el-tab-pane label="文章" name="articles">
            <div class="flex items-center gap-2 mb-3">
              <el-button text :loading="articleLoading" @click="loadArticles"
                >刷新</el-button
              >
            </div>
            <el-table v-loading="articleLoading" :data="articles" stripe>
              <el-table-column prop="slug" label="Slug" min-width="140" />
              <el-table-column label="标题" min-width="180">
                <template #default="{ row }">{{ articleTitle(row) }}</template>
              </el-table-column>
              <el-table-column label="分类" min-width="140">
                <template #default="{ row }">
                  {{ categoryNameMap[row.category_id] ?? row.category_id }}
                </template>
              </el-table-column>
              <el-table-column
                prop="sort_order"
                label="排序"
                width="80"
                align="center"
              />
              <el-table-column label="状态" width="92">
                <template #default="{ row }">
                  <el-tag
                    :type="row.published ? 'success' : 'info'"
                    size="small"
                  >
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
                    @click="openEditArticle(row)"
                  >
                    编辑
                  </el-button>
                  <el-button
                    v-perms="['admin.content.manage']"
                    link
                    type="danger"
                    @click="removeArticle(row)"
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
                :total="articleTotal"
                :page-size="articleQuery.page_size"
                :current-page="articleQuery.page"
                @current-change="
                  page => {
                    articleQuery.page = page;
                    loadArticles();
                  }
                "
              />
            </div>
          </el-tab-pane>

          <el-tab-pane label="分类" name="categories">
            <el-table v-loading="categoryLoading" :data="categories" stripe>
              <el-table-column prop="slug" label="Slug" min-width="140" />
              <el-table-column label="名称" min-width="180">
                <template #default="{ row }">{{ categoryName(row) }}</template>
              </el-table-column>
              <el-table-column
                prop="sort_order"
                label="排序"
                width="80"
                align="center"
              />
              <el-table-column label="操作" width="150" fixed="right">
                <template #default="{ row }">
                  <el-button
                    v-perms="['admin.content.manage']"
                    link
                    type="primary"
                    @click="openEditCategory(row)"
                  >
                    编辑
                  </el-button>
                  <el-button
                    v-perms="['admin.content.manage']"
                    link
                    type="danger"
                    @click="removeCategory(row)"
                  >
                    删除
                  </el-button>
                </template>
              </el-table-column>
            </el-table>
          </el-tab-pane>
        </el-tabs>
      </div>
    </div>

    <el-dialog
      v-model="articleDialogVisible"
      :title="articleDialogTitle"
      width="780px"
      destroy-on-close
    >
      <el-form :model="articleForm" label-width="120px">
        <el-form-item label="Slug">
          <el-input
            v-model="articleForm.slug"
            placeholder="如 getting-started"
          />
        </el-form-item>
        <el-form-item label="所属分类">
          <el-select v-model="articleForm.category_id" style="width: 240px">
            <el-option
              v-for="cat in categories"
              :key="cat.id"
              :label="categoryName(cat)"
              :value="cat.id"
            />
          </el-select>
        </el-form-item>
        <el-form-item label="排序">
          <el-input-number v-model="articleForm.sort_order" :min="0" />
        </el-form-item>
        <el-form-item label="发布">
          <el-switch v-model="articleForm.published" />
        </el-form-item>
        <el-form-item label="多语言内容">
          <LocaleTranslationTabs
            v-model:translations="articleForm.translations"
            :locales="locales"
            :fields="articleFields"
          />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="articleDialogVisible = false">取消</el-button>
        <el-button
          type="primary"
          :loading="articleSaving"
          @click="submitArticle"
          >保存</el-button
        >
      </template>
    </el-dialog>

    <el-dialog
      v-model="categoryDialogVisible"
      :title="categoryDialogTitle"
      width="640px"
      destroy-on-close
    >
      <el-form :model="categoryForm" label-width="120px">
        <el-form-item label="Slug">
          <el-input v-model="categoryForm.slug" placeholder="如 guides" />
        </el-form-item>
        <el-form-item label="排序">
          <el-input-number v-model="categoryForm.sort_order" :min="0" />
        </el-form-item>
        <el-form-item label="多语言名称">
          <LocaleTranslationTabs
            v-model:translations="categoryForm.translations"
            :locales="locales"
            :fields="categoryFields"
          />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="categoryDialogVisible = false">取消</el-button>
        <el-button
          type="primary"
          :loading="categorySaving"
          @click="submitCategory"
          >保存</el-button
        >
      </template>
    </el-dialog>
  </div>
</template>

<style scoped>
@import url("@/style/business.scss");
</style>
