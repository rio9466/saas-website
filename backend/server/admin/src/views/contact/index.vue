<script setup lang="ts">
import { onMounted, reactive, ref } from "vue";
import dayjs from "dayjs";
import { message } from "@/utils/message";
import { getErrorMessage } from "@/utils/error";
import {
  listContactSubmissionsApi,
  getContactSubmissionApi,
  updateContactSubmissionStatusApi
} from "@/api/contactSubmissions";
import type { ContactStatus, ContactSubmission } from "@/api/contract";

defineOptions({ name: "ContactInbox" });

const STATUS_LABELS: Record<ContactStatus, string> = {
  new: "未读",
  read: "已读",
  handled: "已处理"
};

const STATUS_TAG: Record<ContactStatus, "danger" | "info" | "success"> = {
  new: "danger",
  read: "info",
  handled: "success"
};

const loading = ref(false);
const list = ref<ContactSubmission[]>([]);
const total = ref(0);
const query = reactive<{
  page: number;
  page_size: number;
  status: "" | ContactStatus;
}>({ page: 1, page_size: 20, status: "" });

async function loadList() {
  loading.value = true;
  try {
    const res = await listContactSubmissionsApi({
      page: query.page,
      page_size: query.page_size,
      status: query.status || undefined
    });
    list.value = res?.data?.items ?? [];
    total.value = res?.data?.total ?? 0;
  } catch (error) {
    message(getErrorMessage(error), { type: "error" });
  } finally {
    loading.value = false;
  }
}

function handleFilter() {
  query.page = 1;
  loadList();
}

function formatTime(value: string): string {
  if (!value) return "-";
  const time = dayjs(value);
  return time.isValid() ? time.format("YYYY-MM-DD HH:mm") : value;
}

// ---------- 详情 ----------
const drawerVisible = ref(false);
const detailLoading = ref(false);
const updating = ref(false);
const detail = ref<ContactSubmission | null>(null);

async function openDetail(row: ContactSubmission) {
  drawerVisible.value = true;
  detailLoading.value = true;
  try {
    const res = await getContactSubmissionApi(row.id);
    detail.value = res?.data ?? row;
  } catch (error) {
    detail.value = row;
    message(getErrorMessage(error), { type: "error" });
  } finally {
    detailLoading.value = false;
  }
}

async function changeStatus(status: ContactStatus) {
  if (!detail.value || detail.value.status === status) return;
  updating.value = true;
  try {
    const res = await updateContactSubmissionStatusApi(detail.value.id, status);
    if (res?.data) detail.value = res.data;
    message("状态已更新", { type: "success" });
    loadList();
  } catch (error) {
    message(getErrorMessage(error), { type: "error" });
  } finally {
    updating.value = false;
  }
}

onMounted(() => {
  loadList();
});
</script>

<template>
  <div class="flex flex-col gap-4">
    <div class="panel-container">
      <div class="panel-header">
        <span class="panel-title">联系表单收件箱</span>
        <div class="flex items-center gap-2">
          <el-select
            v-model="query.status"
            placeholder="全部状态"
            clearable
            style="width: 160px"
            @change="handleFilter"
          >
            <el-option label="未读" value="new" />
            <el-option label="已读" value="read" />
            <el-option label="已处理" value="handled" />
          </el-select>
          <el-button text :loading="loading" @click="loadList">刷新</el-button>
        </div>
      </div>
      <div class="panel-body table-body">
        <el-table v-loading="loading" :data="list" stripe>
          <el-table-column prop="id" label="ID" width="76" />
          <el-table-column prop="name" label="姓名" min-width="120" />
          <el-table-column prop="email" label="邮箱" min-width="180" />
          <el-table-column prop="company" label="公司" min-width="140" />
          <el-table-column prop="locale" label="语言" width="90" />
          <el-table-column label="状态" width="100">
            <template #default="{ row }">
              <el-tag
                :type="STATUS_TAG[row.status as ContactStatus]"
                size="small"
              >
                {{ STATUS_LABELS[row.status as ContactStatus] }}
              </el-tag>
            </template>
          </el-table-column>
          <el-table-column label="提交时间" width="170">
            <template #default="{ row }">
              {{ formatTime(row.created_at) }}
            </template>
          </el-table-column>
          <el-table-column label="操作" width="120" fixed="right">
            <template #default="{ row }">
              <el-button link type="primary" @click="openDetail(row)">
                查看详情
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

    <el-drawer
      v-model="drawerVisible"
      title="留言详情"
      size="520px"
      destroy-on-close
    >
      <div v-loading="detailLoading" class="detail-body">
        <template v-if="detail">
          <div class="detail-row">
            <span class="detail-label">状态</span>
            <el-tag :type="STATUS_TAG[detail.status as ContactStatus]">
              {{ STATUS_LABELS[detail.status as ContactStatus] }}
            </el-tag>
          </div>
          <div class="detail-row">
            <span class="detail-label">姓名</span>
            <span>{{ detail.name }}</span>
          </div>
          <div class="detail-row">
            <span class="detail-label">邮箱</span>
            <span>{{ detail.email }}</span>
          </div>
          <div class="detail-row">
            <span class="detail-label">公司</span>
            <span>{{ detail.company || "-" }}</span>
          </div>
          <div class="detail-row">
            <span class="detail-label">语言</span>
            <span>{{ detail.locale }}</span>
          </div>
          <div class="detail-row">
            <span class="detail-label">提交时间</span>
            <span>{{ formatTime(detail.created_at) }}</span>
          </div>
          <div class="detail-row">
            <span class="detail-label">来源 IP</span>
            <span>{{ detail.source_ip || "-" }}</span>
          </div>
          <div class="detail-row">
            <span class="detail-label">User-Agent</span>
            <span class="detail-ua">{{ detail.user_agent || "-" }}</span>
          </div>
          <div class="detail-message">
            <span class="detail-label">留言内容</span>
            <pre class="details-pre">{{ detail.message }}</pre>
          </div>

          <el-divider />
          <div class="status-actions">
            <span class="detail-label">状态流转</span>
            <div class="flex gap-2">
              <el-button
                v-perms="['admin.contact.manage']"
                :type="detail.status === 'new' ? 'primary' : 'default'"
                :loading="updating"
                @click="changeStatus('new')"
              >
                未读
              </el-button>
              <el-button
                v-perms="['admin.contact.manage']"
                :type="detail.status === 'read' ? 'primary' : 'default'"
                :loading="updating"
                @click="changeStatus('read')"
              >
                已读
              </el-button>
              <el-button
                v-perms="['admin.contact.manage']"
                :type="detail.status === 'handled' ? 'primary' : 'default'"
                :loading="updating"
                @click="changeStatus('handled')"
              >
                已处理
              </el-button>
            </div>
          </div>
        </template>
      </div>
    </el-drawer>
  </div>
</template>

<style scoped>
@import url("@/style/business.scss");

.detail-body {
  min-height: 200px;
}

.detail-row {
  display: flex;
  gap: 12px;
  padding: 6px 0;
}

.detail-label {
  flex-shrink: 0;
  width: 80px;
  color: var(--el-text-color-secondary);
}

.detail-ua {
  word-break: break-all;
}

.detail-message {
  display: flex;
  gap: 12px;
  margin-top: 8px;
}

.detail-message .details-pre {
  flex: 1;
  margin: 0;
  white-space: pre-wrap;
}

.status-actions {
  display: flex;
  flex-direction: column;
  gap: 8px;
}
</style>
