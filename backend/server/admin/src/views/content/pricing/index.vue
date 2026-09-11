<script setup lang="ts">
import { computed, onMounted, reactive, ref } from "vue";
import { ElMessageBox } from "element-plus";
import { message } from "@/utils/message";
import { getErrorMessage } from "@/utils/error";
import {
  listPricingPlansApi,
  createPricingPlanApi,
  updatePricingPlanApi,
  deletePricingPlanApi
} from "@/api/pricingPlans";
import type { PricingPlan } from "@/api/contract";
import { loadContentLocales, useContentLocales } from "../useContentLocales";
import { compactTranslations, listToText, textToList } from "../translations";
import type { TranslationFieldDef, TranslationMap } from "../types";
import LocaleTranslationTabs from "../components/LocaleTranslationTabs.vue";

defineOptions({ name: "ContentPricing" });

const { locales } = useContentLocales();

const loading = ref(false);
const list = ref<PricingPlan[]>([]);
const total = ref(0);
const query = reactive({ page: 1, page_size: 20 });

const translationFields: TranslationFieldDef[] = [
  { key: "name", label: "方案名称" },
  { key: "description", label: "方案说明", type: "textarea", rows: 3 },
  { key: "cta_label", label: "按钮文案" },
  { key: "cta_url", label: "按钮链接" }
];

async function loadList() {
  loading.value = true;
  try {
    const res = await listPricingPlansApi(query);
    list.value = res?.data?.items ?? [];
    total.value = res?.data?.total ?? 0;
  } catch (error) {
    message(getErrorMessage(error), { type: "error" });
  } finally {
    loading.value = false;
  }
}

const dialogVisible = ref(false);
const dialogTitle = ref("新建价格方案");
const saving = ref(false);
const editingId = ref<string | null>(null);

const form = reactive({
  code: "",
  monthly_price: "0.00",
  yearly_price: "0.00",
  currency: "USD",
  highlighted: false,
  sort_order: 0,
  visible: true,
  translations: {} as TranslationMap,
  features: {} as Record<string, string[]>
});

const nameMissing = computed(
  () => Object.keys(compactTranslations(form.translations)).length === 0
);

function openCreate() {
  editingId.value = null;
  dialogTitle.value = "新建价格方案";
  Object.assign(form, {
    code: "",
    monthly_price: "0.00",
    yearly_price: "0.00",
    currency: "USD",
    highlighted: false,
    sort_order: 0,
    visible: true,
    translations: {},
    features: {}
  });
  dialogVisible.value = true;
}

function openEdit(row: PricingPlan) {
  editingId.value = row.id;
  dialogTitle.value = `编辑价格方案：${row.code}`;
  Object.assign(form, {
    code: row.code,
    monthly_price: row.monthly_price,
    yearly_price: row.yearly_price,
    currency: row.currency,
    highlighted: row.highlighted,
    sort_order: row.sort_order,
    visible: row.visible,
    translations: JSON.parse(JSON.stringify(row.translations ?? {})),
    features: JSON.parse(JSON.stringify(row.features ?? {}))
  });
  dialogVisible.value = true;
}

function compactFeatures(): Record<string, string[]> {
  const out: Record<string, string[]> = {};
  for (const [locale, items] of Object.entries(form.features ?? {})) {
    const cleaned = (items ?? []).map(item => item.trim()).filter(Boolean);
    if (cleaned.length) out[locale] = cleaned;
  }
  return out;
}

function setFeatures(locale: string, text: string) {
  form.features[locale] = textToList(text);
}

async function submit() {
  if (!form.code.trim()) {
    message("请填写方案编码", { type: "warning" });
    return;
  }
  if (nameMissing.value) {
    message("请至少填写一种语言的方案名称", { type: "warning" });
    return;
  }
  saving.value = true;
  try {
    const payload = {
      code: form.code.trim(),
      monthly_price: form.monthly_price,
      yearly_price: form.yearly_price,
      currency: form.currency,
      highlighted: form.highlighted,
      sort_order: form.sort_order,
      visible: form.visible,
      translations: compactTranslations(form.translations),
      features: compactFeatures()
    };
    if (editingId.value) {
      await updatePricingPlanApi(editingId.value, payload);
    } else {
      await createPricingPlanApi(payload);
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

async function remove(row: PricingPlan) {
  try {
    await ElMessageBox.confirm(
      "确定删除该价格方案吗？此操作不可撤销。",
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
    await deletePricingPlanApi(row.id);
    message("已删除", { type: "success" });
    if (list.value.length === 1 && query.page > 1) query.page -= 1;
    loadList();
  } catch (error) {
    message(getErrorMessage(error), { type: "error" });
  }
}

function planName(row: PricingPlan): string {
  return (
    row.translations?.["zh-CN"]?.name ||
    row.translations?.["en"]?.name ||
    Object.values(row.translations ?? {})[0]?.name ||
    row.code
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
        <span class="panel-title">价格方案</span>
        <div class="flex items-center gap-2">
          <el-button text :loading="loading" @click="loadList">刷新</el-button>
          <el-button
            v-perms="['admin.content.manage']"
            type="primary"
            plain
            @click="openCreate"
          >
            新建方案
          </el-button>
        </div>
      </div>
      <div class="panel-body table-body">
        <el-table v-loading="loading" :data="list" stripe>
          <el-table-column prop="code" label="编码" min-width="110" />
          <el-table-column label="名称" min-width="140">
            <template #default="{ row }">{{ planName(row) }}</template>
          </el-table-column>
          <el-table-column label="月价" width="110" align="right">
            <template #default="{ row }">
              {{ row.monthly_price }} {{ row.currency }}
            </template>
          </el-table-column>
          <el-table-column label="年价" width="110" align="right">
            <template #default="{ row }">
              {{ row.yearly_price }} {{ row.currency }}
            </template>
          </el-table-column>
          <el-table-column label="推荐" width="80" align="center">
            <template #default="{ row }">
              <el-tag v-if="row.highlighted" type="warning" size="small">
                推荐
              </el-tag>
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
              <el-tag :type="row.visible ? 'success' : 'info'" size="small">
                {{ row.visible ? "可见" : "隐藏" }}
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
      width="760px"
      destroy-on-close
    >
      <el-form :model="form" label-width="120px">
        <el-form-item label="编码">
          <el-input
            v-model="form.code"
            :disabled="!!editingId"
            placeholder="唯一编码，如 pro"
          />
        </el-form-item>
        <el-form-item label="月价">
          <el-input v-model="form.monthly_price" placeholder="如 29.00" />
        </el-form-item>
        <el-form-item label="年价">
          <el-input v-model="form.yearly_price" placeholder="如 290.00" />
        </el-form-item>
        <el-form-item label="币种">
          <el-input v-model="form.currency" placeholder="如 USD" />
        </el-form-item>
        <el-form-item label="排序">
          <el-input-number v-model="form.sort_order" :min="0" />
        </el-form-item>
        <el-form-item label="推荐">
          <el-switch v-model="form.highlighted" />
        </el-form-item>
        <el-form-item label="可见">
          <el-switch v-model="form.visible" />
        </el-form-item>
        <LocaleTranslationTabs
          v-model:translations="form.translations"
          :locales="locales"
          :fields="translationFields"
        >
          <template #extra="{ locale }">
            <el-form-item label="功能清单">
              <el-input
                type="textarea"
                :rows="4"
                placeholder="每行一条功能，如：10 个席位"
                :model-value="listToText(form.features[locale])"
                @update:model-value="value => setFeatures(locale, value)"
              />
            </el-form-item>
          </template>
        </LocaleTranslationTabs>
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
