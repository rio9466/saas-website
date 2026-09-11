<script setup lang="ts">
import { onMounted, reactive, ref } from "vue";
import { message } from "@/utils/message";
import { getErrorMessage } from "@/utils/error";
import {
  listUserLevelsApi,
  createUserLevelApi,
  updateUserLevelApi
} from "@/api/userLevels";
import { toFourDecimal } from "@/utils/points";
import type { UserLevel } from "@/api/contract";
import type { FormInstance, FormRules } from "element-plus";

defineOptions({ name: "UserLevelManage" });

const loading = ref(false);
const list = ref<UserLevel[]>([]);

async function loadList() {
  loading.value = true;
  try {
    const res = await listUserLevelsApi();
    list.value = res?.data?.items ?? [];
  } catch (error) {
    message(getErrorMessage(error), { type: "error" });
  } finally {
    loading.value = false;
  }
}

const dialogVisible = ref(false);
const dialogTitle = ref("新建等级");
const dialogLoading = ref(false);
const formRef = ref<FormInstance>();
const editingId = ref<string | null>(null);
const form = reactive({
  code: "",
  name: "",
  icon_url: "",
  threshold_points: "0.0000",
  sort_order: 0,
  enabled: true
});

const codeRule = {
  pattern: /^[A-Za-z0-9._-]{1,64}$/,
  message: "1-64 位字母/数字/._-",
  trigger: "blur"
} as const;

const thresholdRule = {
  pattern: /^\d+(\.\d{1,4})?$/,
  message: "请输入非负且最多四位小数的阈值，如 100.0000",
  trigger: "blur"
} as const;

const rules: FormRules = {
  code: [{ required: true, ...codeRule }],
  name: [{ required: true, message: "请输入等级名称", trigger: "blur" }],
  threshold_points: [
    { required: true, message: "请输入累积消费阈值", trigger: "blur" },
    thresholdRule
  ]
};

function openCreate() {
  editingId.value = null;
  dialogTitle.value = "新建用户等级";
  Object.assign(form, {
    code: "",
    name: "",
    icon_url: "",
    threshold_points: "0.0000",
    sort_order: 0,
    enabled: true
  });
  dialogVisible.value = true;
}

function openEdit(row: UserLevel) {
  editingId.value = row.id;
  dialogTitle.value = `编辑等级：${row.name}`;
  Object.assign(form, {
    code: row.code,
    name: row.name,
    icon_url: row.icon_url,
    threshold_points: row.threshold_points,
    sort_order: row.sort_order,
    enabled: row.enabled
  });
  dialogVisible.value = true;
}

async function submit() {
  if (!formRef.value) return;
  await formRef.value.validate(async valid => {
    if (!valid) return;
    dialogLoading.value = true;
    try {
      const payload = {
        name: form.name,
        icon_url: form.icon_url || undefined,
        threshold_points: toFourDecimal(form.threshold_points),
        sort_order: form.sort_order,
        enabled: form.enabled
      };
      if (editingId.value) {
        await updateUserLevelApi(editingId.value, payload);
      } else {
        await createUserLevelApi({
          code: form.code,
          ...payload
        });
      }
      message(editingId.value ? "保存成功" : "创建成功", { type: "success" });
      dialogVisible.value = false;
      loadList();
    } catch (error) {
      message(getErrorMessage(error), { type: "error" });
    } finally {
      dialogLoading.value = false;
    }
  });
}

/** 停用/启用确认（默认等级不可停用，由后端最终校验） */
const toggleVisible = ref(false);
const toggleLoading = ref(false);
const toggleTarget = ref<UserLevel | null>(null);
const toggleEnable = ref(true);

function openToggle(row: UserLevel) {
  toggleTarget.value = row;
  toggleEnable.value = !row.enabled;
  toggleVisible.value = true;
}

async function confirmToggle() {
  if (!toggleTarget.value) return;
  toggleLoading.value = true;
  try {
    await updateUserLevelApi(toggleTarget.value.id, {
      enabled: toggleEnable.value
    });
    message(toggleEnable.value ? "已启用" : "已停用", { type: "success" });
    toggleVisible.value = false;
    loadList();
  } catch (error) {
    message(getErrorMessage(error), { type: "error" });
  } finally {
    toggleLoading.value = false;
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
        <span class="panel-title">用户等级</span>
        <div class="flex items-center gap-2">
          <el-button text :loading="loading" @click="loadList">刷新</el-button>
          <el-button
            v-perms="['admin.user_level.manage']"
            type="primary"
            plain
            @click="openCreate"
          >
            新建等级
          </el-button>
        </div>
      </div>
      <div class="panel-body table-body">
        <el-alert
          type="info"
          :closable="false"
          class="mb-3"
          title="自动等级按累积消费积分选取“阈值 ≤ 消费”的最高已启用等级；两个已启用等级不可共用同一阈值。停用等级不会影响已有手动分配记录。"
        />
        <el-table v-loading="loading" :data="list" stripe>
          <el-table-column prop="id" label="ID" width="76" />
          <el-table-column prop="code" label="编码" min-width="110" />
          <el-table-column prop="name" label="名称" min-width="120" />
          <el-table-column label="阈值（累积消费）" width="160" align="right">
            <template #default="{ row }">
              <span>{{ row.threshold_points }}</span>
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
              <el-tag :type="row.enabled ? 'success' : 'danger'" size="small">
                {{ row.enabled ? "启用" : "停用" }}
              </el-tag>
            </template>
          </el-table-column>
          <el-table-column
            prop="icon_url"
            label="图标 URL"
            min-width="160"
            show-overflow-tooltip
          />
          <el-table-column label="操作" width="150" fixed="right">
            <template #default="{ row }">
              <el-button
                v-perms="['admin.user_level.manage']"
                link
                type="primary"
                @click="openEdit(row)"
              >
                编辑
              </el-button>
              <el-button
                v-perms="['admin.user_level.manage']"
                link
                :type="row.enabled ? 'warning' : 'success'"
                @click="openToggle(row)"
              >
                {{ row.enabled ? "停用" : "启用" }}
              </el-button>
            </template>
          </el-table-column>
        </el-table>
      </div>
    </div>

    <el-dialog
      v-model="dialogVisible"
      :title="dialogTitle"
      width="520px"
      destroy-on-close
    >
      <el-form ref="formRef" :model="form" :rules="rules" label-width="120px">
        <el-form-item v-if="!editingId" label="编码" prop="code">
          <el-input v-model="form.code" placeholder="唯一编码，如 gold" />
        </el-form-item>
        <el-form-item v-else label="编码">
          <el-input :model-value="form.code" disabled />
        </el-form-item>
        <el-form-item label="名称" prop="name">
          <el-input v-model="form.name" placeholder="如 黄金会员" />
        </el-form-item>
        <el-form-item label="阈值" prop="threshold_points">
          <el-input v-model="form.threshold_points" placeholder="如 100.0000" />
          <div class="text-xs text-secondary mt-1">
            累积消费积分达到该值（含）即达标
          </div>
        </el-form-item>
        <el-form-item label="排序">
          <el-input-number v-model="form.sort_order" :min="0" />
        </el-form-item>
        <el-form-item label="图标 URL">
          <el-input v-model="form.icon_url" placeholder="可留空" />
        </el-form-item>
        <el-form-item label="启用">
          <el-switch v-model="form.enabled" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="dialogVisible = false">取消</el-button>
        <el-button type="primary" :loading="dialogLoading" @click="submit"
          >保存</el-button
        >
      </template>
    </el-dialog>

    <el-dialog
      v-model="toggleVisible"
      :title="toggleEnable ? '启用等级' : '停用等级'"
      width="440px"
      destroy-on-close
    >
      <el-alert
        :type="toggleEnable ? 'info' : 'warning'"
        :closable="false"
        :title="
          toggleEnable
            ? `确定启用「${toggleTarget?.name ?? ''}」吗？`
            : `确定停用「${toggleTarget?.name ?? ''}」吗？停用后新分配/自动计算不再选择它。`
        "
      />
      <template #footer>
        <el-button @click="toggleVisible = false">取消</el-button>
        <el-button
          :type="toggleEnable ? 'primary' : 'danger'"
          :loading="toggleLoading"
          @click="confirmToggle"
        >
          {{ toggleEnable ? "确认启用" : "确认停用" }}
        </el-button>
      </template>
    </el-dialog>
  </div>
</template>

<style scoped>
@import url("@/style/business.scss");
</style>
