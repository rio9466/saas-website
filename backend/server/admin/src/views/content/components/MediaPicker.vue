<script setup lang="ts">
import { ref } from "vue";
import { message } from "@/utils/message";
import { getErrorMessage } from "@/utils/error";
import { listMediaApi, uploadMediaApi, deleteMediaApi } from "@/api/media";
import type { MediaAsset } from "@/api/contract";

/**
 * 媒体选择器：可直接输入 URL，也可从媒体库选择或上传新文件。
 * 用于站点设置（Logo/Favicon/OG）与内容表单（功能配图、首页图等）。
 */
withDefaults(
  defineProps<{
    modelValue?: string;
    placeholder?: string;
    accept?: string;
  }>(),
  {
    modelValue: "",
    placeholder: "可直接填写 URL，或从媒体库选择",
    accept: "image/*"
  }
);

const emit = defineEmits<{ "update:modelValue": [string] }>();

const dialogVisible = ref(false);
const loading = ref(false);
const uploading = ref(false);
const list = ref<MediaAsset[]>([]);
const total = ref(0);
const page = ref(1);
const pageSize = ref(12);
const fileInput = ref<HTMLInputElement>();

async function loadMedia() {
  loading.value = true;
  try {
    const res = await listMediaApi({
      page: page.value,
      page_size: pageSize.value
    });
    list.value = res?.data?.items ?? [];
    total.value = res?.data?.total ?? 0;
  } catch (error) {
    message(getErrorMessage(error), { type: "error" });
  } finally {
    loading.value = false;
  }
}

function openPicker() {
  dialogVisible.value = true;
  page.value = 1;
  loadMedia();
}

function select(asset: MediaAsset) {
  emit("update:modelValue", asset.url);
  dialogVisible.value = false;
}

function pickFile() {
  fileInput.value?.click();
}

async function onFileChange(event: Event) {
  const target = event.target as HTMLInputElement;
  const file = target.files?.[0];
  target.value = "";
  if (!file) return;
  uploading.value = true;
  try {
    const res = await uploadMediaApi(file);
    const url = res?.data?.url;
    if (url) emit("update:modelValue", url);
    message("上传成功", { type: "success" });
  } catch (error) {
    message(getErrorMessage(error), { type: "error" });
  } finally {
    uploading.value = false;
  }
}

async function removeAsset(asset: MediaAsset) {
  try {
    await deleteMediaApi(asset.id);
    message("已删除", { type: "success" });
    loadMedia();
  } catch (error) {
    message(getErrorMessage(error), { type: "error" });
  }
}

function onPageChange(value: number) {
  page.value = value;
  loadMedia();
}
</script>

<template>
  <div class="media-picker">
    <el-input
      :model-value="modelValue"
      :placeholder="placeholder"
      clearable
      @update:model-value="value => emit('update:modelValue', value)"
    >
      <template #append>
        <el-button :disabled="uploading" @click="openPicker">媒体库</el-button>
        <el-button :loading="uploading" @click="pickFile">上传</el-button>
      </template>
    </el-input>
    <input
      ref="fileInput"
      type="file"
      class="hidden-file"
      :accept="accept"
      @change="onFileChange"
    />
    <div v-if="modelValue" class="media-preview">
      <el-image :src="modelValue" fit="contain" class="preview-image" />
    </div>

    <el-dialog
      v-model="dialogVisible"
      title="媒体库"
      width="720px"
      destroy-on-close
    >
      <div v-loading="loading" class="media-grid">
        <div
          v-for="asset in list"
          :key="asset.id"
          class="media-card"
          @click="select(asset)"
        >
          <el-image :src="asset.url" fit="contain" class="media-thumb" />
          <div class="media-meta">
            <span class="media-url" :title="asset.url">{{ asset.url }}</span>
            <el-button
              link
              type="danger"
              size="small"
              @click.stop="removeAsset(asset)"
            >
              删除
            </el-button>
          </div>
        </div>
        <el-empty v-if="!loading && list.length === 0" description="暂无媒体" />
      </div>
      <template #footer>
        <div class="dialog-footer">
          <el-pagination
            v-if="total > pageSize"
            :current-page="page"
            :page-size="pageSize"
            :total="total"
            layout="prev, pager, next"
            @current-change="onPageChange"
          />
          <el-button @click="dialogVisible = false">关闭</el-button>
        </div>
      </template>
    </el-dialog>
  </div>
</template>

<style scoped>
.hidden-file {
  display: none;
}

.media-preview {
  margin-top: 8px;
}

.preview-image {
  width: 120px;
  height: 60px;
  border: 1px solid var(--el-border-color-lighter);
  border-radius: 4px;
}

.media-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(150px, 1fr));
  gap: 12px;
  min-height: 200px;
}

.media-card {
  padding: 8px;
  cursor: pointer;
  border: 1px solid var(--el-border-color-lighter);
  border-radius: 4px;
}

.media-card:hover {
  border-color: var(--el-color-primary);
}

.media-thumb {
  width: 100%;
  height: 90px;
}

.media-meta {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-top: 6px;
}

.media-url {
  max-width: 90px;
  overflow: hidden;
  text-overflow: ellipsis;
  font-size: 12px;
  color: var(--el-text-color-secondary);
  white-space: nowrap;
}

.dialog-footer {
  display: flex;
  align-items: center;
  justify-content: space-between;
}
</style>
