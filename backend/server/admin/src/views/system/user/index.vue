<script setup lang="ts">
import { onMounted, reactive, ref } from "vue";
import { message } from "@/utils/message";
import { getErrorMessage } from "@/utils/error";
import {
  createPasswordByteValidator,
  PASSWORD_PLACEHOLDER
} from "@/utils/password";
import {
  listUsersApi,
  createUserApi,
  updateUserApi,
  enableUserApi,
  disableUserApi,
  resetUserPasswordApi,
  adjustUserPointsApi,
  listUserPointTransactionsApi,
  assignUserLevelApi
} from "@/api/users";
import { listUserLevelsApi } from "@/api/userLevels";
import { roundHalfUpTwo as formatPoints } from "@/utils/points";
import type { BusinessUser, UserLevel, PointTransaction } from "@/api/contract";
import type { FormInstance, FormRules } from "element-plus";

defineOptions({ name: "UserManage" });

const loading = ref(false);
const list = ref<BusinessUser[]>([]);
const total = ref(0);
const levelOptions = ref<UserLevel[]>([]);

const query = reactive({
  page: 1,
  page_size: 20,
  q: "",
  status: "" as "" | "pending_verification" | "active" | "disabled",
  level_id: ""
});

async function loadList() {
  loading.value = true;
  try {
    const res = await listUsersApi({
      page: query.page,
      page_size: query.page_size,
      q: query.q || undefined,
      status: query.status || undefined,
      level_id: query.level_id || undefined
    });
    list.value = res?.data?.items ?? [];
    total.value = res?.data?.total ?? 0;
  } catch (error) {
    message(getErrorMessage(error), { type: "error" });
  } finally {
    loading.value = false;
  }
}

async function loadLevels() {
  try {
    const res = await listUserLevelsApi();
    levelOptions.value = res?.data?.items ?? [];
  } catch {
    levelOptions.value = [];
  }
}

function handleQuery() {
  query.page = 1;
  loadList();
}

function handleReset() {
  query.q = "";
  query.status = "";
  query.level_id = "";
  query.page = 1;
  loadList();
}

const statusText: Record<string, string> = {
  pending_verification: "待验证",
  active: "启用",
  disabled: "禁用"
};
const statusType: Record<string, "info" | "success" | "danger"> = {
  pending_verification: "info",
  active: "success",
  disabled: "danger"
};

function levelNameOf(row: BusinessUser): string {
  return row.level?.name || row.level?.code || "-";
}

// ---------- 新建 / 编辑 ----------
const dialogVisible = ref(false);
const dialogTitle = ref("新建用户");
const dialogLoading = ref(false);
const formRef = ref<FormInstance>();
const editingId = ref<string | null>(null);
const form = reactive({
  username: "",
  email: "",
  password: "",
  nickname: "",
  avatar_url: "",
  remark: ""
});

const emailRule = {
  pattern: /^[^@\s]+@[^@\s]+\.[^@\s]+$/,
  message: "请输入有效的邮箱地址",
  trigger: "blur"
} as const;

const createRules: FormRules = {
  username: [
    { required: true, message: "请输入用户名", trigger: "blur" },
    {
      pattern: /^[A-Za-z0-9._-]{3,64}$/,
      message: "3-64 位字母/数字/._-，且不能包含 @",
      trigger: "blur"
    }
  ],
  email: [{ required: true, ...emailRule }],
  password: [
    { required: true, message: "请输入初始密码", trigger: "blur" },
    createPasswordByteValidator()
  ],
  nickname: [{ max: 64, message: "昵称最多 64 字符", trigger: "blur" }]
};

const editRules: FormRules = {
  email: [{ required: true, ...emailRule }],
  nickname: [{ required: true, message: "昵称不能为空", trigger: "blur" }]
};

function openCreate() {
  editingId.value = null;
  dialogTitle.value = "新建用户";
  Object.assign(form, {
    username: "",
    email: "",
    password: "",
    nickname: "",
    avatar_url: "",
    remark: ""
  });
  dialogVisible.value = true;
}

function openEdit(row: BusinessUser) {
  editingId.value = row.id;
  dialogTitle.value = `编辑用户：${row.username}`;
  Object.assign(form, {
    username: row.username,
    email: row.email,
    password: "",
    nickname: row.nickname,
    avatar_url: row.avatar_url,
    remark: row.remark
  });
  dialogVisible.value = true;
}

async function submitDialog() {
  if (!formRef.value) return;
  await formRef.value.validate(async valid => {
    if (!valid) return;
    dialogLoading.value = true;
    try {
      if (editingId.value) {
        await updateUserApi(editingId.value, {
          email: form.email,
          nickname: form.nickname,
          avatar_url: form.avatar_url || undefined,
          remark: form.remark || undefined
        });
      } else {
        await createUserApi({
          username: form.username,
          email: form.email,
          password: form.password,
          nickname: form.nickname || undefined
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

// ---------- 启用 / 禁用 ----------
const toggleDialogVisible = ref(false);
const toggleLoading = ref(false);
const toggleTarget = ref<BusinessUser | null>(null);
const toggleAction = ref<"enable" | "disable">("disable");

function openToggle(row: BusinessUser) {
  toggleTarget.value = row;
  toggleAction.value = row.status === "active" ? "disable" : "enable";
  toggleDialogVisible.value = true;
}

async function confirmToggle() {
  if (!toggleTarget.value) return;
  const row = toggleTarget.value;
  const isDisable = toggleAction.value === "disable";
  toggleLoading.value = true;
  try {
    if (isDisable) {
      await disableUserApi(row.id);
      message(`已禁用「${row.username}」`, { type: "success" });
    } else {
      await enableUserApi(row.id);
      message(`已启用「${row.username}」`, { type: "success" });
    }
    toggleDialogVisible.value = false;
    loadList();
  } catch (error) {
    message(getErrorMessage(error), { type: "error" });
  } finally {
    toggleLoading.value = false;
  }
}

// ---------- 重置密码 ----------
const resetVisible = ref(false);
const resetLoading = ref(false);
const resetFormRef = ref<FormInstance>();
const resetTarget = ref<BusinessUser | null>(null);
const resetForm = reactive({ new_password: "", confirm: "" });
const resetRules: FormRules = {
  new_password: [
    { required: true, message: "请输入新密码", trigger: "blur" },
    createPasswordByteValidator()
  ],
  confirm: [
    {
      validator: (_, value, callback) => {
        if (value !== resetForm.new_password) {
          callback(new Error("两次输入的密码不一致"));
        } else {
          callback();
        }
      },
      trigger: "blur"
    }
  ]
};

function openReset(row: BusinessUser) {
  resetTarget.value = row;
  resetForm.new_password = "";
  resetForm.confirm = "";
  resetVisible.value = true;
}

async function submitReset() {
  if (!resetFormRef.value || !resetTarget.value) return;
  await resetFormRef.value.validate(async valid => {
    if (!valid) return;
    resetLoading.value = true;
    try {
      await resetUserPasswordApi(resetTarget.value!.id, {
        new_password: resetForm.new_password
      });
      message("密码已重置，旧会话将全部失效", { type: "success" });
      resetVisible.value = false;
    } catch (error) {
      message(getErrorMessage(error), { type: "error" });
    } finally {
      resetLoading.value = false;
    }
  });
}

// ---------- 积分调整（原子账本） ----------
const pointsVisible = ref(false);
const pointsLoading = ref(false);
const pointsFormRef = ref<FormInstance>();
const pointsTarget = ref<BusinessUser | null>(null);
const pointsForm = reactive({
  points_delta: "",
  consumption_delta: "0.0000",
  reason: "",
  idempotency_key: ""
});

const fourDecimalRule = {
  pattern: /^-?\d+(\.\d{1,4})?$/,
  message: "请输入最多四位小数的数值，例如 5.0000",
  trigger: "blur"
} as const;

const pointsRules: FormRules = {
  points_delta: [
    {
      required: true,
      message: "请输入可用积分变动（可正可负）",
      trigger: "blur"
    },
    fourDecimalRule
  ],
  consumption_delta: [
    {
      required: true,
      message: "请输入累积消费积分变动（不可为负）",
      trigger: "blur"
    },
    fourDecimalRule
  ],
  reason: [
    { required: true, message: "必须填写调整原因", trigger: "blur" },
    { max: 200, message: "原因最多 200 字符", trigger: "blur" }
  ]
};

function openAdjustPoints(row: BusinessUser) {
  pointsTarget.value = row;
  Object.assign(pointsForm, {
    points_delta: "",
    consumption_delta: "0.0000",
    reason: "",
    idempotency_key:
      "admin-" +
      Date.now().toString(36) +
      "-" +
      Math.random().toString(36).slice(2, 10)
  });
  pointsVisible.value = true;
}

function submitPoints() {
  if (!pointsFormRef.value || !pointsTarget.value) return;
  pointsFormRef.value.validate(async valid => {
    if (!valid) return;
    pointsLoading.value = true;
    try {
      await adjustUserPointsApi(pointsTarget.value!.id, {
        points_delta: pointsForm.points_delta,
        consumption_delta: pointsForm.consumption_delta || "0.0000",
        reason: pointsForm.reason,
        idempotency_key: pointsForm.idempotency_key
      });
      message("积分调整成功（已写入不可变账本）", { type: "success" });
      pointsVisible.value = false;
      loadList();
    } catch (error) {
      message(getErrorMessage(error), { type: "error" });
    } finally {
      pointsLoading.value = false;
    }
  });
}

// ---------- 积分账本 ----------
const ledgerVisible = ref(false);
const ledgerLoading = ref(false);
const ledgerTarget = ref<BusinessUser | null>(null);
const ledgerItems = ref<PointTransaction[]>([]);
const ledgerTotal = ref(0);
const ledgerQuery = reactive({ page: 1, page_size: 10 });

async function openLedger(row: BusinessUser) {
  ledgerTarget.value = row;
  ledgerItems.value = [];
  ledgerTotal.value = 0;
  ledgerQuery.page = 1;
  ledgerVisible.value = true;
  await loadLedger();
}

async function loadLedger() {
  if (!ledgerTarget.value) return;
  ledgerLoading.value = true;
  try {
    const res = await listUserPointTransactionsApi(ledgerTarget.value.id, {
      page: ledgerQuery.page,
      page_size: ledgerQuery.page_size
    });
    ledgerItems.value = res?.data?.items ?? [];
    ledgerTotal.value = res?.data?.total ?? 0;
  } catch (error) {
    message(getErrorMessage(error), { type: "error" });
  } finally {
    ledgerLoading.value = false;
  }
}

// ---------- 等级设置 ----------
const levelVisible = ref(false);
const levelLoading = ref(false);
const levelTarget = ref<BusinessUser | null>(null);
const levelForm = reactive({
  mode: "auto" as "auto" | "manual",
  level_id: ""
});

function openAssignLevel(row: BusinessUser) {
  levelTarget.value = row;
  levelForm.mode = row.level?.mode === "manual" ? "manual" : "auto";
  levelForm.level_id = row.level?.mode === "manual" ? row.level.id : "";
  levelVisible.value = true;
}

async function submitLevel() {
  if (!levelTarget.value) return;
  if (levelForm.mode === "manual" && !levelForm.level_id) {
    message("手动等级请选择一个等级", { type: "warning" });
    return;
  }
  levelLoading.value = true;
  try {
    await assignUserLevelApi(levelTarget.value.id, {
      level_mode: levelForm.mode,
      level_id: levelForm.mode === "manual" ? levelForm.level_id : undefined
    });
    message(
      levelForm.mode === "auto" ? "已切回自动等级并重新计算" : "已设置手动等级",
      {
        type: "success"
      }
    );
    levelVisible.value = false;
    loadList();
  } catch (error) {
    message(getErrorMessage(error), { type: "error" });
  } finally {
    levelLoading.value = false;
  }
}

onMounted(() => {
  loadList();
  loadLevels();
});
</script>

<template>
  <div class="flex flex-col gap-4">
    <!-- 筛选/操作面板 -->
    <div class="panel-container">
      <div class="panel-body">
        <div class="flex flex-wrap items-center justify-between gap-3">
          <div class="flex flex-wrap items-center gap-2">
            <el-input
              v-model="query.q"
              placeholder="用户名 / 邮箱 / 昵称"
              clearable
              style="width: 220px"
              @keyup.enter="handleQuery"
            />
            <el-select
              v-model="query.status"
              placeholder="状态"
              clearable
              style="width: 130px"
            >
              <el-option label="待验证" value="pending_verification" />
              <el-option label="启用" value="active" />
              <el-option label="禁用" value="disabled" />
            </el-select>
            <el-select
              v-model="query.level_id"
              placeholder="等级"
              clearable
              filterable
              style="width: 150px"
            >
              <el-option
                v-for="l in levelOptions"
                :key="l.id"
                :label="l.name"
                :value="l.id"
              />
            </el-select>
          </div>
          <div class="flex items-center gap-2">
            <el-button type="primary" @click="handleQuery">查询</el-button>
            <el-button @click="handleReset">重置</el-button>
            <el-button
              v-perms="['admin.customer.create']"
              type="primary"
              plain
              @click="openCreate"
            >
              新建用户
            </el-button>
          </div>
        </div>
      </div>
    </div>

    <!-- 用户表格 -->
    <div class="panel-container">
      <div class="panel-header">
        <span class="panel-title">业务用户列表</span>
        <el-button text :loading="loading" @click="loadList">刷新</el-button>
      </div>
      <div class="panel-body table-body">
        <el-table v-loading="loading" :data="list" stripe>
          <el-table-column prop="id" label="ID" width="76" />
          <el-table-column prop="username" label="用户名" min-width="120" />
          <el-table-column
            prop="email"
            label="邮箱"
            min-width="190"
            show-overflow-tooltip
          />
          <el-table-column
            prop="nickname"
            label="昵称"
            min-width="110"
            show-overflow-tooltip
          />
          <el-table-column label="状态" width="92">
            <template #default="{ row }">
              <el-tag :type="statusType[row.status] ?? 'info'" size="small">
                {{ statusText[row.status] ?? row.status }}
              </el-tag>
            </template>
          </el-table-column>
          <el-table-column label="可用积分" width="100" align="right">
            <template #default="{ row }">
              <el-tooltip :content="row.points_balance" placement="top">
                <span>{{ formatPoints(row.points_balance) }}</span>
              </el-tooltip>
            </template>
          </el-table-column>
          <el-table-column label="累积消费" width="100" align="right">
            <template #default="{ row }">
              <el-tooltip :content="row.consumption_points" placement="top">
                <span>{{ formatPoints(row.consumption_points) }}</span>
              </el-tooltip>
            </template>
          </el-table-column>
          <el-table-column label="等级" min-width="120">
            <template #default="{ row }">
              <span>{{ levelNameOf(row) }}</span>
              <el-tag
                v-if="row.level?.mode === 'manual'"
                size="small"
                type="warning"
                class="ml-1"
              >
                手动
              </el-tag>
            </template>
          </el-table-column>
          <el-table-column prop="registration_ip" label="注册 IP" width="120" />
          <el-table-column label="最后登录" min-width="160">
            <template #default="{ row }">
              {{
                row.last_login_at
                  ? new Date(row.last_login_at).toLocaleString()
                  : "—"
              }}
            </template>
          </el-table-column>
          <el-table-column label="操作" width="372" fixed="right">
            <template #default="{ row }">
              <el-button
                v-perms="['admin.customer.update']"
                link
                type="primary"
                @click="openEdit(row)"
              >
                编辑
              </el-button>
              <el-button
                v-perms="['admin.customer.disable']"
                link
                :type="row.status === 'active' ? 'warning' : 'success'"
                @click="openToggle(row)"
              >
                {{ row.status === "active" ? "禁用" : "启用" }}
              </el-button>
              <el-button
                v-perms="['admin.customer.reset_password']"
                link
                type="danger"
                @click="openReset(row)"
              >
                重置密码
              </el-button>
              <el-button
                v-perms="['admin.customer.points']"
                link
                type="primary"
                @click="openAdjustPoints(row)"
              >
                调积分
              </el-button>
              <el-button
                v-perms="['admin.customer.points']"
                link
                @click="openLedger(row)"
              >
                账本
              </el-button>
              <el-button
                v-perms="['admin.customer.level_assign']"
                link
                @click="openAssignLevel(row)"
              >
                设等级
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

    <!-- 新建/编辑对话框 -->
    <el-dialog
      v-model="dialogVisible"
      :title="dialogTitle"
      width="540px"
      destroy-on-close
    >
      <el-form
        ref="formRef"
        :model="form"
        :rules="editingId ? editRules : createRules"
        label-width="110px"
      >
        <el-form-item v-if="!editingId" label="用户名" prop="username">
          <el-input
            v-model="form.username"
            placeholder="字母/数字/._-，不含 @（创建后不可修改）"
          />
        </el-form-item>
        <el-form-item v-if="editingId" label="用户名">
          <el-input :model-value="form.username" disabled />
        </el-form-item>
        <el-form-item label="邮箱" prop="email">
          <el-input v-model="form.email" placeholder="登录邮箱" />
        </el-form-item>
        <el-form-item v-if="!editingId" label="初始密码" prop="password">
          <el-input
            v-model="form.password"
            type="password"
            show-password
            :placeholder="PASSWORD_PLACEHOLDER"
          />
        </el-form-item>
        <el-form-item label="昵称" prop="nickname">
          <el-input
            v-model="form.nickname"
            placeholder="管理员可填昵称，不填则默认用户名"
          />
        </el-form-item>
        <el-form-item label="头像 URL">
          <el-input
            v-model="form.avatar_url"
            placeholder="可留空使用系统默认头像"
          />
        </el-form-item>
        <el-form-item v-if="editingId" label="备注">
          <el-input
            v-model="form.remark"
            type="textarea"
            :rows="2"
            placeholder="管理员可见备注"
          />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="dialogVisible = false">取消</el-button>
        <el-button type="primary" :loading="dialogLoading" @click="submitDialog"
          >保存</el-button
        >
      </template>
    </el-dialog>

    <!-- 启用/禁用确认 -->
    <el-dialog
      v-model="toggleDialogVisible"
      :title="toggleAction === 'disable' ? '禁用用户' : '启用用户'"
      width="440px"
      destroy-on-close
    >
      <el-alert
        :type="toggleAction === 'disable' ? 'warning' : 'info'"
        :closable="false"
        :title="
          toggleAction === 'disable'
            ? `确定禁用「${toggleTarget?.username ?? ''}」吗？禁用后该账号无法登录。`
            : `确定启用「${toggleTarget?.username ?? ''}」吗？`
        "
      />
      <template #footer>
        <el-button @click="toggleDialogVisible = false">取消</el-button>
        <el-button
          :type="toggleAction === 'disable' ? 'danger' : 'primary'"
          :loading="toggleLoading"
          @click="confirmToggle"
        >
          {{ toggleAction === "disable" ? "确认禁用" : "确认启用" }}
        </el-button>
      </template>
    </el-dialog>

    <!-- 重置密码 -->
    <el-dialog
      v-model="resetVisible"
      title="重置用户密码"
      width="480px"
      destroy-on-close
    >
      <el-alert
        type="warning"
        :closable="false"
        :title="`将重置「${resetTarget?.username ?? ''}」的密码，所有已登录会话将失效。`"
        class="mb-3"
      />
      <el-form
        ref="resetFormRef"
        :model="resetForm"
        :rules="resetRules"
        label-width="100px"
      >
        <el-form-item label="新密码" prop="new_password">
          <el-input
            v-model="resetForm.new_password"
            type="password"
            show-password
            :placeholder="PASSWORD_PLACEHOLDER"
          />
        </el-form-item>
        <el-form-item label="确认密码" prop="confirm">
          <el-input
            v-model="resetForm.confirm"
            type="password"
            show-password
            placeholder="再次输入新密码"
          />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="resetVisible = false">取消</el-button>
        <el-button type="danger" :loading="resetLoading" @click="submitReset"
          >确认重置</el-button
        >
      </template>
    </el-dialog>

    <!-- 积分调整 -->
    <el-dialog
      v-model="pointsVisible"
      :title="`调整积分：${pointsTarget?.username ?? ''}`"
      width="540px"
      destroy-on-close
    >
      <el-alert
        type="info"
        :closable="false"
        class="mb-3"
        :title="`当前可用 ${formatPoints(pointsTarget?.points_balance)}，累积消费 ${formatPoints(pointsTarget?.consumption_points)}。调整将原子写入不可变账本；同一 idempotency_key 重试不会重复生效。`"
      />
      <el-form
        ref="pointsFormRef"
        :model="pointsForm"
        :rules="pointsRules"
        label-width="130px"
      >
        <el-form-item label="可用积分变动" prop="points_delta">
          <el-input
            v-model="pointsForm.points_delta"
            placeholder="如 10.0000 或 -5.0000（余额不可为负）"
          />
        </el-form-item>
        <el-form-item label="累积消费变动" prop="consumption_delta">
          <el-input
            v-model="pointsForm.consumption_delta"
            placeholder="如 150.0000（不可为负）"
          />
          <div class="text-xs text-secondary mt-1">
            增加累积消费会自动重算“自动”模式的等级
          </div>
        </el-form-item>
        <el-form-item label="原因（必填）" prop="reason">
          <el-input
            v-model="pointsForm.reason"
            placeholder="调整原因，将写入审计日志"
          />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="pointsVisible = false">取消</el-button>
        <el-button type="primary" :loading="pointsLoading" @click="submitPoints"
          >提交调整</el-button
        >
      </template>
    </el-dialog>

    <!-- 积分账本 -->
    <el-dialog
      v-model="ledgerVisible"
      :title="`积分账本：${ledgerTarget?.username ?? ''}`"
      width="860px"
      destroy-on-close
    >
      <el-table
        v-loading="ledgerLoading"
        :data="ledgerItems"
        stripe
        max-height="460"
      >
        <el-table-column prop="id" label="ID" width="76" />
        <el-table-column label="可用变动" width="110" align="right">
          <template #default="{ row }">
            <el-tooltip :content="row.points_delta" placement="top">
              <span>{{ formatPoints(row.points_delta) }}</span>
            </el-tooltip>
          </template>
        </el-table-column>
        <el-table-column label="消费变动" width="110" align="right">
          <template #default="{ row }">
            <el-tooltip :content="row.consumption_delta" placement="top">
              <span>{{ formatPoints(row.consumption_delta) }}</span>
            </el-tooltip>
          </template>
        </el-table-column>
        <el-table-column label="变动后余额" width="120" align="right">
          <template #default="{ row }">
            <el-tooltip :content="row.balance_after" placement="top">
              <span>{{ formatPoints(row.balance_after) }}</span>
            </el-tooltip>
          </template>
        </el-table-column>
        <el-table-column
          prop="reason"
          label="原因"
          min-width="150"
          show-overflow-tooltip
        />
        <el-table-column prop="actor_id" label="操作人ID" width="100" />
        <el-table-column label="时间" min-width="160">
          <template #default="{ row }">
            {{ new Date(row.created_at).toLocaleString() }}
          </template>
        </el-table-column>
      </el-table>
      <div class="flex justify-end pt-3">
        <el-pagination
          background
          layout="total, prev, pager, next"
          :total="ledgerTotal"
          :page-size="ledgerQuery.page_size"
          :current-page="ledgerQuery.page"
          @current-change="
            page => {
              ledgerQuery.page = page;
              loadLedger();
            }
          "
        />
      </div>
    </el-dialog>

    <!-- 等级设置 -->
    <el-dialog
      v-model="levelVisible"
      :title="`设置等级：${levelTarget?.username ?? ''}`"
      width="480px"
      destroy-on-close
    >
      <el-form :model="levelForm" label-width="110px">
        <el-form-item label="等级模式">
          <el-radio-group v-model="levelForm.mode">
            <el-radio value="auto">自动（按累积消费计算）</el-radio>
            <el-radio value="manual">手动（固定等级）</el-radio>
          </el-radio-group>
        </el-form-item>
        <el-form-item v-if="levelForm.mode === 'manual'" label="手动等级">
          <el-select
            v-model="levelForm.level_id"
            filterable
            style="width: 100%"
          >
            <el-option
              v-for="l in levelOptions.filter(item => item.enabled)"
              :key="l.id"
              :label="`${l.name}（${l.threshold_points}）`"
              :value="l.id"
            />
          </el-select>
          <div class="text-xs text-secondary mt-1">
            已禁用的等级不可选；切回自动后会立即重算
          </div>
        </el-form-item>
        <el-form-item v-else label="说明">
          <div class="text-sm text-secondary">
            自动模式取“累积消费积分 ≥
            阈值”的最高已启用等级，并在消费变动时自动重算。
          </div>
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="levelVisible = false">取消</el-button>
        <el-button type="primary" :loading="levelLoading" @click="submitLevel"
          >保存</el-button
        >
      </template>
    </el-dialog>
  </div>
</template>

<style scoped>
@import url("@/style/business.scss");
</style>
