<script setup lang="ts">
import { onMounted, reactive, ref } from "vue";
import { message } from "@/utils/message";
import { getErrorMessage } from "@/utils/error";
import {
  listAuditEventsApi,
  getAuditEventApi,
  type ListAuditEventsParams
} from "@/api/audit";
import type { AuditEvent } from "@/api/contract";
import {
  auditActionLabel,
  AUDIT_ACTION_FILTER_OPTIONS
} from "@/utils/auditActions";

defineOptions({
  name: "Audit"
});

const loading = ref(false);
const list = ref<AuditEvent[]>([]);
const total = ref(0);
const detailLoading = ref(false);
const detailVisible = ref(false);
const detail = ref<AuditEvent | null>(null);

const query = reactive({
  page: 1,
  page_size: 20,
  action: "",
  resource_type: "",
  outcome: "",
  actor_id: "",
  request_id: "",
  from: "",
  to: ""
});

async function loadList() {
  loading.value = true;
  try {
    const params: ListAuditEventsParams = {
      page: query.page,
      page_size: query.page_size
    };
    if (query.action) params.action = query.action;
    if (query.resource_type) params.resource_type = query.resource_type;
    if (query.outcome)
      params.outcome = query.outcome as ListAuditEventsParams["outcome"];
    if (query.actor_id) params.actor_id = query.actor_id;
    if (query.request_id) params.request_id = query.request_id;
    if (query.from) params.from = query.from;
    if (query.to) params.to = query.to;
    const res = await listAuditEventsApi(params);
    list.value = res?.data?.items ?? [];
    total.value = res?.data?.total ?? 0;
  } catch (error) {
    message(getErrorMessage(error), { type: "error" });
  } finally {
    loading.value = false;
  }
}

function handleQuery() {
  query.page = 1;
  loadList();
}

function handleReset() {
  query.action = "";
  query.resource_type = "";
  query.outcome = "";
  query.actor_id = "";
  query.request_id = "";
  query.from = "";
  query.to = "";
  query.page = 1;
  loadList();
}

function outcomeTag(outcome: AuditEvent["outcome"]): {
  type: "success" | "danger" | "warning";
  text: string;
} {
  switch (outcome) {
    case "succeeded":
      return { type: "success", text: "成功" };
    case "failed":
      return { type: "danger", text: "失败" };
    default:
      return { type: "warning", text: "待确认" };
  }
}

async function openDetail(row: AuditEvent) {
  detailLoading.value = true;
  detailVisible.value = true;
  try {
    const res = await getAuditEventApi(row.id);
    detail.value = res?.data ?? null;
  } catch (error) {
    message(getErrorMessage(error), { type: "error" });
    detailVisible.value = false;
  } finally {
    detailLoading.value = false;
  }
}

/** 只读展示变更详情（后端已脱敏，前端不做导出/变更） */
function formatDetails(ev: AuditEvent | null): string {
  if (!ev) return "";
  return JSON.stringify(ev.details ?? {}, null, 2);
}

// 本页面为只读，不提供任何变更/导出控件

onMounted(() => {
  loadList();
});
</script>

<template>
  <div class="flex flex-col gap-4">
    <!-- 筛选面板 -->
    <div class="panel-container">
      <div class="panel-body">
        <div class="flex flex-wrap items-center justify-between gap-3">
          <div class="flex flex-wrap items-center gap-2">
            <el-select
              v-model="query.action"
              placeholder="动作"
              clearable
              filterable
              style="width: 200px"
            >
              <el-option
                v-for="opt in AUDIT_ACTION_FILTER_OPTIONS"
                :key="opt.value"
                :label="opt.label"
                :value="opt.value"
              />
            </el-select>
            <el-input
              v-model="query.resource_type"
              placeholder="资源类型"
              clearable
              style="width: 140px"
              @keyup.enter="handleQuery"
            />
            <el-select
              v-model="query.outcome"
              placeholder="结果"
              clearable
              style="width: 110px"
            >
              <el-option label="成功" value="succeeded" />
              <el-option label="失败" value="failed" />
              <el-option label="待确认" value="pending" />
            </el-select>
            <el-input
              v-model="query.actor_id"
              placeholder="操作人 ID"
              clearable
              style="width: 120px"
              @keyup.enter="handleQuery"
            />
            <el-input
              v-model="query.request_id"
              placeholder="请求 ID"
              clearable
              style="width: 180px"
              @keyup.enter="handleQuery"
            />
            <el-date-picker
              v-model="query.from"
              type="datetime"
              placeholder="开始时间"
              value-format="YYYY-MM-DDTHH:mm:ssZ"
              style="width: 190px"
            />
            <el-date-picker
              v-model="query.to"
              type="datetime"
              placeholder="结束时间"
              value-format="YYYY-MM-DDTHH:mm:ssZ"
              style="width: 190px"
            />
          </div>
          <div class="flex items-center gap-2">
            <el-button type="primary" @click="handleQuery">查询</el-button>
            <el-button @click="handleReset">重置</el-button>
          </div>
        </div>
      </div>
    </div>

    <!-- 表格面板 -->
    <div class="panel-container">
      <div class="panel-header">
        <span class="panel-title">审计日志（只读）</span>
        <el-button text :loading="loading" @click="loadList">刷新</el-button>
      </div>
      <div class="panel-body table-body">
        <el-table v-loading="loading" :data="list" stripe>
          <el-table-column prop="id" label="ID" width="110" />
          <el-table-column label="结果" width="80">
            <template #default="{ row }">
              <el-tag :type="outcomeTag(row.outcome).type" size="small">
                {{ outcomeTag(row.outcome).text }}
              </el-tag>
            </template>
          </el-table-column>
          <el-table-column prop="action" label="动作" min-width="170">
            <template #default="{ row }">
              <el-tooltip
                :content="row.action"
                placement="top"
                :show-after="400"
              >
                <span class="cursor-help">{{
                  auditActionLabel(row.action)
                }}</span>
              </el-tooltip>
            </template>
          </el-table-column>
          <el-table-column
            prop="resource_type"
            label="资源类型"
            min-width="130"
          />
          <el-table-column prop="resource_id" label="资源 ID" min-width="110" />
          <el-table-column
            prop="actor_username"
            label="操作人"
            min-width="120"
          />
          <el-table-column prop="request_id" label="请求 ID" min-width="180" />
          <el-table-column prop="event_at" label="事件时间" min-width="170" />
          <el-table-column label="操作" width="80" fixed="right">
            <template #default="{ row }">
              <el-button link type="primary" @click="openDetail(row)">
                详情
              </el-button>
            </template>
          </el-table-column>
        </el-table>
        <div class="flex justify-end pt-3">
          <el-pagination
            background
            layout="total, prev, pager, next, sizes"
            :total="total"
            :page-size="query.page_size"
            :current-page="query.page"
            :page-sizes="[10, 20, 50, 100]"
            @current-change="
              page => {
                query.page = page;
                loadList();
              }
            "
            @size-change="
              size => {
                query.page_size = size;
                query.page = 1;
                loadList();
              }
            "
          />
        </div>
      </div>
    </div>

    <!-- 详情对话框（只读） -->
    <el-dialog
      v-model="detailVisible"
      :title="`审计事件详情：${detail?.id ?? ''}`"
      width="720px"
      destroy-on-close
    >
      <div v-loading="detailLoading">
        <el-descriptions v-if="detail" :column="2" border size="small">
          <el-descriptions-item label="ID">{{
            detail.id
          }}</el-descriptions-item>
          <el-descriptions-item label="结果">
            <el-tag :type="outcomeTag(detail.outcome).type" size="small">
              {{ outcomeTag(detail.outcome).text }}
            </el-tag>
          </el-descriptions-item>
          <el-descriptions-item label="动作">{{
            auditActionLabel(detail.action)
          }}</el-descriptions-item>
          <el-descriptions-item label="资源">
            {{ detail.resource_type }} / {{ detail.resource_id ?? "—" }}
          </el-descriptions-item>
          <el-descriptions-item label="操作人">
            {{ detail.actor_display_name || detail.actor_username || "—" }}
            （{{ detail.actor_id ?? "系统" }}）
          </el-descriptions-item>
          <el-descriptions-item label="操作人角色">
            {{ (detail.actor_role_codes ?? []).join("、") || "—" }}
          </el-descriptions-item>
          <el-descriptions-item label="请求 ID" :span="2">
            {{ detail.request_id }}
          </el-descriptions-item>
          <el-descriptions-item label="来源 IP">{{
            detail.source_ip ?? "—"
          }}</el-descriptions-item>
          <el-descriptions-item label="事件时间">{{
            detail.event_at
          }}</el-descriptions-item>
        </el-descriptions>
        <p
          class="mb-1 mt-3 text-sm font-medium"
          style="color: var(--el-text-color-primary)"
        >
          变更详情（已脱敏）
        </p>
        <pre class="details-pre">{{ formatDetails(detail) }}</pre>
      </div>
    </el-dialog>
  </div>
</template>

<style scoped>
@import url("@/style/business.scss");
</style>
