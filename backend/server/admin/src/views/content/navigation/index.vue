<script setup lang="ts">
import { computed, onMounted, reactive, ref } from "vue";
import { ElMessageBox } from "element-plus";
import { message } from "@/utils/message";
import { getErrorMessage } from "@/utils/error";
import {
  listNavigationItemsApi,
  createNavigationItemApi,
  updateNavigationItemApi,
  deleteNavigationItemApi
} from "@/api/navigation";
import type { NavigationItem } from "@/api/contract";
import { loadContentLocales, useContentLocales } from "../useContentLocales";
import { compactTranslations } from "../translations";
import type { TranslationFieldDef, TranslationMap } from "../types";
import LocaleTranslationTabs from "../components/LocaleTranslationTabs.vue";

defineOptions({ name: "ContentNavigation" });

const { locales } = useContentLocales();

const loading = ref(false);
const list = ref<NavigationItem[]>([]);
const total = ref(0);
const query = reactive<{
  page: number;
  page_size: number;
  placement: "header" | "footer";
}>({ page: 1, page_size: 100, placement: "header" });

const translationFields: TranslationFieldDef[] = [
  { key: "label", label: "显示文案" }
];

async function loadList() {
  loading.value = true;
  try {
    const res = await listNavigationItemsApi(query);
    list.value = res?.data?.items ?? [];
    total.value = res?.data?.total ?? 0;
  } catch (error) {
    message(getErrorMessage(error), { type: "error" });
  } finally {
    loading.value = false;
  }
}

const dialogVisible = ref(false);
const dialogTitle = ref("新建导航");
const saving = ref(false);
const editingId = ref<string | null>(null);

const form = reactive({
  placement: "header" as "header" | "footer",
  parent_id: "",
  url: "",
  target: "_self" as "_self" | "_blank",
  sort_order: 0,
  visible: true,
  translations: {} as TranslationMap
});

const labelMissing = computed(
  () => Object.keys(compactTranslations(form.translations)).length === 0
);

const parentOptions = computed(() =>
  list.value.filter(
    item => item.placement === form.placement && item.id !== editingId.value
  )
);

function openCreate() {
  editingId.value = null;
  dialogTitle.value = "新建导航项";
  Object.assign(form, {
    placement: query.placement,
    parent_id: "",
    url: "",
    target: "_self",
    sort_order: 0,
    visible: true,
    translations: {}
  });
  dialogVisible.value = true;
}

function openEdit(row: NavigationItem) {
  editingId.value = row.id;
  dialogTitle.value = "编辑导航项";
  Object.assign(form, {
    placement: row.placement,
    parent_id: row.parent_id ?? "",
    url: row.url,
    target: row.target,
    sort_order: row.sort_order,
    visible: row.visible,
    translations: JSON.parse(JSON.stringify(row.translations ?? {}))
  });
  dialogVisible.value = true;
}

async function submit() {
  if (labelMissing.value) {
    message("请至少填写一种语言的显示文案", { type: "warning" });
    return;
  }
  saving.value = true;
  try {
    const payload = {
      placement: form.placement,
      parent_id: form.parent_id || "",
      url: form.url,
      target: form.target,
      sort_order: form.sort_order,
      visible: form.visible,
      translations: compactTranslations(form.translations)
    };
    if (editingId.value) {
      await updateNavigationItemApi(editingId.value, payload);
    } else {
      await createNavigationItemApi(payload);
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

async function remove(row: NavigationItem) {
  try {
    await ElMessageBox.confirm(
      "确定删除该导航项吗？其子项将失去父级。",
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
    await deleteNavigationItemApi(row.id);
    message("已删除", { type: "success" });
    loadList();
  } catch (error) {
    message(getErrorMessage(error), { type: "error" });
  }
}

function labelOf(row: NavigationItem): string {
  return (
    row.translations?.["zh-CN"]?.label ||
    row.translations?.["en"]?.label ||
    Object.values(row.translations ?? {})[0]?.label ||
    "-"
  );
}

function parentLabel(row: NavigationItem): string {
  const parent = list.value.find(item => item.id === row.parent_id);
  return parent ? labelOf(parent) : "-";
}

function switchPlacement(value: "header" | "footer") {
  query.placement = value;
  query.page = 1;
  loadList();
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
        <span class="panel-title">导航管理</span>
        <div class="flex items-center gap-2">
          <el-radio-group
            :model-value="query.placement"
            @change="switchPlacement"
          >
            <el-radio-button value="header">顶部导航</el-radio-button>
            <el-radio-button value="footer">底部导航</el-radio-button>
          </el-radio-group>
          <el-button text :loading="loading" @click="loadList">刷新</el-button>
          <el-button
            v-perms="['admin.content.manage']"
            type="primary"
            plain
            @click="openCreate"
          >
            新建导航项
          </el-button>
        </div>
      </div>
      <div class="panel-body table-body">
        <el-table v-loading="loading" :data="list" stripe>
          <el-table-column prop="id" label="ID" width="76" />
          <el-table-column label="显示文案" min-width="150">
            <template #default="{ row }">{{ labelOf(row) }}</template>
          </el-table-column>
          <el-table-column label="父级" min-width="120">
            <template #default="{ row }">{{ parentLabel(row) }}</template>
          </el-table-column>
          <el-table-column prop="url" label="链接" min-width="160" />
          <el-table-column prop="target" label="打开方式" width="100" />
          <el-table-column
            prop="sort_order"
            label="排序"
            width="80"
            align="center"
          />
          <el-table-column label="可见" width="80" align="center">
            <template #default="{ row }">
              <el-tag :type="row.visible ? 'success' : 'info'" size="small">
                {{ row.visible ? "显示" : "隐藏" }}
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
      </div>
    </div>

    <el-dialog
      v-model="dialogVisible"
      :title="dialogTitle"
      width="700px"
      destroy-on-close
    >
      <el-form :model="form" label-width="120px">
        <el-form-item label="位置">
          <el-select v-model="form.placement" style="width: 200px">
            <el-option label="顶部导航" value="header" />
            <el-option label="底部导航" value="footer" />
          </el-select>
        </el-form-item>
        <el-form-item label="父级">
          <el-select
            v-model="form.parent_id"
            clearable
            placeholder="顶级（无父级）"
            style="width: 260px"
          >
            <el-option
              v-for="item in parentOptions"
              :key="item.id"
              :label="labelOf(item)"
              :value="item.id"
            />
          </el-select>
        </el-form-item>
        <el-form-item label="链接">
          <el-input
            v-model="form.url"
            placeholder="如 /features 或 https://..."
          />
        </el-form-item>
        <el-form-item label="打开方式">
          <el-select v-model="form.target" style="width: 200px">
            <el-option label="当前窗口" value="_self" />
            <el-option label="新窗口" value="_blank" />
          </el-select>
        </el-form-item>
        <el-form-item label="排序">
          <el-input-number v-model="form.sort_order" :min="0" />
        </el-form-item>
        <el-form-item label="可见">
          <el-switch v-model="form.visible" />
        </el-form-item>
        <el-form-item label="多语言文案">
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
