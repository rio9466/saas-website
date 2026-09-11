<script setup lang="ts">
import { computed, onMounted, ref, watch } from "vue";
import ReCol from "@/components/ReCol";
import Segmented, { type OptionsType } from "@/components/ReSegmented";
import { message } from "@/utils/message";
import { getErrorMessage } from "@/utils/error";
import { hasPerms } from "@/utils/perms";
import { getAnalyticsOverviewApi } from "@/api/analytics";
import { listUsersApi } from "@/api/users";
import { listContactSubmissionsApi } from "@/api/contactSubmissions";
import type { AnalyticsOverview, AnalyticsRange } from "@/api/contract";
import AnalyticsSources from "./components/AnalyticsSources.vue";

defineOptions({
  name: "Dashboard"
});

/** 访问统计（overview）受 admin.analytics.read 约束；注册/未读各按自身权限展示 */
const canReadAnalytics = computed(() => hasPerms("admin.analytics.read"));
const canReadUsers = computed(() => hasPerms("admin.customer.read"));
const canReadContact = computed(() => hasPerms("admin.contact.read"));

const RANGES: AnalyticsRange[] = ["today", "7d", "30d"];
const rangeOptions: Array<OptionsType> = [
  { label: "今日" },
  { label: "7天" },
  { label: "30天" }
];
// ReSegmented 以索引作为 v-model，映射到接口的 range 参数
const rangeIndex = ref(0);

const analyticsLoading = ref(false);
const overview = ref<AnalyticsOverview | null>(null);

const usersLoading = ref(false);
const userTotal = ref(0);

const contactLoading = ref(false);
const contactNewTotal = ref(0);

async function loadOverview() {
  if (!canReadAnalytics.value) return;
  analyticsLoading.value = true;
  try {
    const res = await getAnalyticsOverviewApi(RANGES[rangeIndex.value]);
    overview.value = res?.data ?? null;
  } catch (error) {
    message(getErrorMessage(error), { type: "error" });
  } finally {
    analyticsLoading.value = false;
  }
}

async function loadUserTotal() {
  if (!canReadUsers.value) return;
  usersLoading.value = true;
  try {
    const res = await listUsersApi({ page_size: 1 });
    userTotal.value = res?.data?.total ?? 0;
  } catch (error) {
    message(getErrorMessage(error), { type: "error" });
  } finally {
    usersLoading.value = false;
  }
}

async function loadContactNewTotal() {
  if (!canReadContact.value) return;
  contactLoading.value = true;
  try {
    const res = await listContactSubmissionsApi({
      status: "new",
      page_size: 1
    });
    contactNewTotal.value = res?.data?.total ?? 0;
  } catch (error) {
    message(getErrorMessage(error), { type: "error" });
  } finally {
    contactLoading.value = false;
  }
}

watch(rangeIndex, () => {
  loadOverview();
});

onMounted(() => {
  loadOverview();
  loadUserTotal();
  loadContactNewTotal();
});
</script>

<template>
  <div class="flex flex-col gap-4">
    <el-row :gutter="24">
      <re-col
        v-if="canReadAnalytics"
        :value="6"
        :sm="12"
        :xs="24"
        class="mb-4.5"
      >
        <div v-loading="analyticsLoading" class="panel-container metric-card">
          <span class="metric-label">访问人数</span>
          <span class="metric-value">{{ overview?.pv ?? 0 }}</span>
        </div>
      </re-col>
      <re-col
        v-if="canReadAnalytics"
        :value="6"
        :sm="12"
        :xs="24"
        class="mb-4.5"
      >
        <div v-loading="analyticsLoading" class="panel-container metric-card">
          <span class="metric-label">访问来源</span>
          <span class="metric-value">{{ overview?.source_count ?? 0 }}</span>
        </div>
      </re-col>
      <re-col v-if="canReadUsers" :value="6" :sm="12" :xs="24" class="mb-4.5">
        <div v-loading="usersLoading" class="panel-container metric-card">
          <span class="metric-label">注册人数</span>
          <span class="metric-value">{{ userTotal }}</span>
        </div>
      </re-col>
      <re-col v-if="canReadContact" :value="6" :sm="12" :xs="24" class="mb-4.5">
        <div v-loading="contactLoading" class="panel-container metric-card">
          <span class="metric-label">未读留言</span>
          <span class="metric-value">{{ contactNewTotal }}</span>
        </div>
      </re-col>
    </el-row>

    <div v-if="canReadAnalytics" class="panel-container">
      <div class="panel-header">
        <span class="panel-title">访问来源</span>
        <Segmented v-model="rangeIndex" :options="rangeOptions" />
      </div>
      <div class="panel-body">
        <AnalyticsSources
          :sources="overview?.sources ?? []"
          :loading="analyticsLoading"
        />
      </div>
    </div>
  </div>
</template>

<style scoped>
@import url("@/style/business.scss");

.metric-card {
  display: flex;
  flex-direction: column;
  gap: 12px;
  padding: 20px;
}

.metric-label {
  font-size: 14px;
  color: var(--el-text-color-secondary);
}

.metric-value {
  font-size: 28px;
  font-weight: 500;
  line-height: 1.2;
  color: var(--el-text-color-primary);
}
</style>
