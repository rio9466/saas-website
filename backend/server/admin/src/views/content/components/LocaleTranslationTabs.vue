<script setup lang="ts">
import { ref, watch } from "vue";
import type { LocaleOption } from "@/api/contract";
import type { TranslationFieldDef, TranslationMap } from "../types";
import { listToText, textToList } from "../translations";
import JsonTextarea from "./JsonTextarea.vue";
import MediaPicker from "./MediaPicker.vue";

/**
 * 多语言翻译编辑器：按已启用语言渲染语言 Tab。
 * - 未填写的语言显示「未翻译」，并可点击「添加该语言」后编辑；
 * - 支持 input / textarea / string-list（每行一条）/ kv-list（每行 键|值）/
 *   media（媒体选择器）/ json；
 * - rawJson 模式用于字段不定（如未知首页区块类型）时整体编辑该语言对象；
 * - 通过 v-model:translations 双向同步（不可变更新），父级提交时用
 *   compactTranslations 过滤只保留填写的语言。
 */
const props = withDefaults(
  defineProps<{
    translations: TranslationMap;
    locales: LocaleOption[];
    fields?: TranslationFieldDef[];
    missingText?: string;
    rawJson?: boolean;
    disabled?: boolean;
  }>(),
  { fields: () => [], missingText: "未翻译", rawJson: false, disabled: false }
);

const emit = defineEmits<{
  "update:translations": [TranslationMap];
  invalid: [boolean];
}>();

const active = ref("");

watch(
  () => props.locales,
  list => {
    if (!list.some(l => l.code === active.value)) {
      active.value = list[0]?.code ?? "";
    }
  },
  { immediate: true, deep: true }
);

function has(code: string): boolean {
  return Object.prototype.hasOwnProperty.call(props.translations, code);
}

function localeObject(code: string): Record<string, unknown> {
  return props.translations[code] ?? {};
}

function update(next: TranslationMap) {
  emit("update:translations", next);
}

function add(code: string) {
  update({ ...props.translations, [code]: {} });
}

function remove(code: string) {
  const next = { ...props.translations };
  delete next[code];
  update(next);
}

function get(code: string, key: string): unknown {
  return props.translations[code]?.[key];
}

function set(code: string, key: string, value: unknown) {
  update({
    ...props.translations,
    [code]: { ...localeObject(code), [key]: value }
  });
}

/** rawJson：整体替换该语言的翻译对象 */
function setLocaleObject(code: string, value: unknown) {
  update({
    ...props.translations,
    [code]: (value as Record<string, unknown>) ?? {}
  });
}

/** kv-list：Array<{value,label}> <-> 每行 `value|label` */
function kvToText(value: unknown): string {
  if (!Array.isArray(value)) return "";
  return value
    .map(item => {
      const record = (item ?? {}) as Record<string, unknown>;
      const v = typeof record.value === "string" ? record.value : "";
      const l = typeof record.label === "string" ? record.label : "";
      return `${v}|${l}`;
    })
    .join("\n");
}

function textToKv(text: string): Array<{ value: string; label: string }> {
  return text
    .split("\n")
    .map(line => line.trim())
    .filter(Boolean)
    .map(line => {
      const index = line.indexOf("|");
      if (index === -1) return { value: line.trim(), label: "" };
      return {
        value: line.slice(0, index).trim(),
        label: line.slice(index + 1).trim()
      };
    });
}
</script>

<template>
  <el-tabs v-model="active" class="locale-tabs">
    <el-tab-pane
      v-for="loc in locales"
      :key="loc.code"
      :name="loc.code"
      :disabled="disabled && !has(loc.code)"
    >
      <template #label>
        <span class="tab-label">
          {{ loc.label }}
          <el-tag
            v-if="!has(loc.code)"
            size="small"
            type="info"
            effect="plain"
            class="ml-1"
          >
            {{ missingText }}
          </el-tag>
        </span>
      </template>

      <div v-if="!has(loc.code)" class="locale-missing">
        <span class="text-secondary">该语言尚未翻译</span>
        <el-button
          type="primary"
          plain
          size="small"
          :disabled="disabled"
          @click="add(loc.code)"
        >
          添加 {{ loc.label }} 翻译
        </el-button>
      </div>
      <slot v-if="!has(loc.code)" name="extra" :locale="loc.code" />

      <template v-else>
        <el-form-item v-if="rawJson" label="该语言数据（JSON）">
          <JsonTextarea
            :model-value="localeObject(loc.code)"
            :rows="10"
            :disabled="disabled"
            @update:model-value="value => setLocaleObject(loc.code, value)"
            @update:invalid="emit('invalid', true)"
          />
        </el-form-item>

        <el-form-item
          v-for="field in fields"
          :key="field.key"
          :label="field.label"
        >
          <el-input
            v-if="(field.type ?? 'input') === 'input'"
            :model-value="get(loc.code, field.key) as string"
            :placeholder="field.placeholder"
            :disabled="disabled"
            @update:model-value="value => set(loc.code, field.key, value)"
          />
          <el-input
            v-else-if="field.type === 'textarea'"
            type="textarea"
            :rows="field.rows ?? 6"
            :placeholder="field.placeholder"
            :disabled="disabled"
            :model-value="get(loc.code, field.key) as string"
            @update:model-value="value => set(loc.code, field.key, value)"
          />
          <el-input
            v-else-if="field.type === 'string-list'"
            type="textarea"
            :rows="field.rows ?? 5"
            :placeholder="field.placeholder ?? '每行一条'"
            :disabled="disabled"
            :model-value="listToText(get(loc.code, field.key) as string[])"
            @update:model-value="
              value => set(loc.code, field.key, textToList(value))
            "
          />
          <el-input
            v-else-if="field.type === 'kv-list'"
            type="textarea"
            :rows="field.rows ?? 4"
            :placeholder="field.placeholder ?? '每行：数值|说明'"
            :disabled="disabled"
            :model-value="kvToText(get(loc.code, field.key))"
            @update:model-value="
              value => set(loc.code, field.key, textToKv(value))
            "
          />
          <MediaPicker
            v-else-if="field.type === 'media'"
            :model-value="(get(loc.code, field.key) as string) ?? ''"
            :placeholder="field.placeholder"
            @update:model-value="value => set(loc.code, field.key, value)"
          />
          <JsonTextarea
            v-else
            :model-value="get(loc.code, field.key)"
            :rows="6"
            :disabled="disabled"
            @update:model-value="value => set(loc.code, field.key, value)"
            @update:invalid="emit('invalid', true)"
          />
          <div v-if="field.tip" class="text-xs text-secondary mt-1">
            {{ field.tip }}
          </div>
        </el-form-item>

        <slot name="extra" :locale="loc.code" />

        <el-form-item>
          <el-button
            link
            type="danger"
            :disabled="disabled"
            @click="remove(loc.code)"
          >
            移除该语言
          </el-button>
        </el-form-item>
      </template>
    </el-tab-pane>
  </el-tabs>
</template>

<style scoped>
.locale-tabs :deep(.el-tabs__content) {
  overflow: visible;
}

.tab-label {
  display: inline-flex;
  align-items: center;
}

.locale-missing {
  display: flex;
  gap: 12px;
  align-items: center;
  padding: 8px 0 16px;
}
</style>
