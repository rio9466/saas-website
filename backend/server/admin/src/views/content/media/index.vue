<script setup lang="ts">
import { onMounted, reactive, ref } from "vue";
import { ElMessageBox } from "element-plus";
import { message } from "@/utils/message";
import { getErrorMessage } from "@/utils/error";
import { listMediaApi, uploadMediaApi, deleteMediaApi } from "@/api/media";
import type { MediaAsset } from "@/api/contract";

defineOptions({ name: "ContentMedia" });

const loading = ref(false);
const uploading = ref(false);
const list = ref<MediaAsset[]>([]);
const total = ref(0);
const query = reactive({ page: 1, page_size: 24 });
const fileInput = ref<HTMLInputElement>();

async function loadList() {
  loading.value = true;
  try {
    const res = await listMediaApi(query);
    list.value = res?.data?.items ?? [];
    total.value = res?.data?.total ?? 0;
  } catch (error) {
    message(getErrorMessage(error), { type: "error" });
  } finally {
    loading.value = false;
  }
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
    await uploadMediaApi(file);
    message("上传成功", { type: "success" });
    query.page = 1;
    loadList();
  } catch (error) {
    message(getErrorMessage(error), { type: "error" });
  } finally {
    uploading.value = false;
  }
}

async function copyUrl(asset: MediaAsset) {
  try {
    await navigator.clipboard.writeText(asset.url);
    message("已复制 URL", { type: "success" });
  } catch {
    message("复制失败，请手动复制", { type: "warning" });
  }
}

async function remove(asset: MediaAsset) {
  try {
    await ElMessageBox.confirm(
      `确定删除媒体「${asset.url}」吗？已引用该文件的页面可能显示异常。`,
      "删除确认",
      { type: "warning", confirmButtonText: "删除", cancelButtonText: "取消" }
    );
  } catch {
    return;
  }
  try {
    await deleteMediaApi(asset.id);
    message("已删除", { type: "success" });
    if (list.value.length === 1 && query.page > 1) query.page -= 1;
    loadList();
  } catch (error) {
    message(getErrorMessage(error), { type: "error" });
  }
}

function formatSize(size: number): string {
  if (size < 1024) return `${size} B`;
  if (size < 1024 * 1024) return `${(size / 1024).toFixed(1)} KB`;
  return `${(size / 1024 / 1024).toFixed(2)} MB`;
}

onMounted(() => {
  loadList();
});
</script>

<template>
  <div class="flex flex-col gap-4">
    <div class="panel-container">
      <div class="panel-header">
        <span class="panel-title">媒体库</span>
        <div class="flex items-center gap-2">
          <el-button text :loading="loading" @click="loadList">刷新</el-button>
          <el-button
            v-perms="['admin.content.manage']"
            type="primary"
            plain
            :loading="uploading"
            @click="pickFile"
          >
            上传媒体
          </el-button>
          <input
            ref="fileInput"
            type="file"
            class="hidden-file"
            accept="image/*"
            @change="onFileChange"
          />
        </div>
      </div>
      <div v-loading="loading" class="panel-body">
        <div class="media-grid">
          <div v-for="asset in list" :key="asset.id" class="media-card">
            <el-image
              :src="asset.url"
              fit="contain"
              class="media-thumb"
              :preview-src-list="[asset.url]"
              preview-teleported
            />
            <div class="media-info">
              <span class="media-url" :title="asset.url">{{ asset.url }}</span>
              <span class="media-meta">
                {{ asset.width }}×{{ asset.height }} ·
                {{ formatSize(asset.size) }}
              </span>
            </div>
            <div class="media-actions">
              <el-button
                v-perms="['admin.content.manage']"
                link
                type="primary"
                @click="copyUrl(asset)"
              >
                复制 URL
              </el-button>
              <el-button
                v-perms="['admin.content.manage']"
                link
                type="danger"
                @click="remove(asset)"
              >
                删除
              </el-button>
            </div>
          </div>
        </div>
        <el-empty v-if="!loading && list.length === 0" description="暂无媒体" />
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
  </div>
</template>

<style scoped>
@import url("@/style/business.scss");

.hidden-file {
  display: none;
}

.media-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(200px, 1fr));
  gap: 16px;
}

.media-card {
  padding: 10px;
  border: 1px solid var(--el-border-color-lighter);
  border-radius: 4px;
}

.media-thumb {
  width: 100%;
  height: 140px;
  background: var(--el-fill-color-light);
}

.media-info {
  display: flex;
  flex-direction: column;
  gap: 2px;
  margin-top: 8px;
}

.media-url {
  overflow: hidden;
  text-overflow: ellipsis;
  font-size: 12px;
  color: var(--el-text-color-primary);
  white-space: nowrap;
}

.media-meta {
  font-size: 12px;
  color: var(--el-text-color-secondary);
}

.media-actions {
  display: flex;
  justify-content: space-between;
  margin-top: 4px;
}
</style>
