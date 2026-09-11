<script setup lang="ts">
import type { AnalyticsSourceItem } from "@/api/contract";

defineProps<{
  sources: AnalyticsSourceItem[];
  loading?: boolean;
}>();

/** `direct` 为后端约定的直接访问标识，展示为中文 */
function sourceLabel(source: string): string {
  return source === "direct" ? "直接访问" : source;
}
</script>

<template>
  <div v-loading="loading" class="source-list">
    <template v-if="sources.length">
      <div v-for="item in sources" :key="item.source" class="source-row">
        <span class="source-name">{{ sourceLabel(item.source) }}</span>
        <span class="source-count">{{ item.count }}</span>
      </div>
    </template>
    <el-empty v-else description="暂无数据" :image-size="60" />
  </div>
</template>

<style scoped>
.source-list {
  min-height: 120px;
}

.source-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 10px 0;
  border-bottom: 1px solid var(--el-border-color-lighter);
}

.source-row:last-child {
  border-bottom: none;
}

.source-name {
  overflow: hidden;
  text-overflow: ellipsis;
  color: var(--el-text-color-regular);
  white-space: nowrap;
}

.source-count {
  flex-shrink: 0;
  margin-left: 12px;
  font-weight: 500;
  color: var(--el-text-color-primary);
}
</style>
