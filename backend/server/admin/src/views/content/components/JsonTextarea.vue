<script setup lang="ts">
import { ref } from "vue";

/**
 * JSON 文本域：内部维护文本缓冲，仅当 JSON 合法时向上抛出解析结果。
 * 初始值只在挂载时读取一次（弹窗均为 destroy-on-close，每次重新挂载）。
 */
const props = withDefaults(
  defineProps<{
    modelValue?: unknown;
    rows?: number;
    placeholder?: string;
    disabled?: boolean;
  }>(),
  { rows: 8, placeholder: "{}", disabled: false }
);

const emit = defineEmits<{
  "update:modelValue": [unknown];
  "update:invalid": [boolean];
}>();

function stringify(value: unknown): string {
  if (value == null || value === "") return "";
  if (typeof value === "string") return value;
  try {
    return JSON.stringify(value, null, 2);
  } catch {
    return "";
  }
}

const text = ref(stringify(props.modelValue));
const invalid = ref(false);

function onChange() {
  const trimmed = text.value.trim();
  if (!trimmed) {
    invalid.value = false;
    emit("update:modelValue", undefined);
    emit("update:invalid", false);
    return;
  }
  try {
    const parsed = JSON.parse(trimmed);
    invalid.value = false;
    emit("update:modelValue", parsed);
    emit("update:invalid", false);
  } catch {
    invalid.value = true;
    emit("update:invalid", true);
  }
}
</script>

<template>
  <el-input
    v-model="text"
    type="textarea"
    :rows="rows"
    :placeholder="placeholder"
    :disabled="disabled"
    :class="{ 'json-invalid': invalid }"
    @input="onChange"
  />
  <div v-if="invalid" class="text-xs json-error">JSON 格式不正确</div>
</template>

<style scoped>
.json-invalid :deep(.el-textarea__inner) {
  border-color: var(--el-color-danger);
}

.json-error {
  margin-top: 4px;
  color: var(--el-color-danger);
}
</style>
