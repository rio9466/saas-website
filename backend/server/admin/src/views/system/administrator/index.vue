<script setup lang="ts">
import { onMounted, reactive, ref } from "vue";
import { message } from "@/utils/message";
import { getErrorMessage } from "@/utils/error";
import { useUserStoreHook } from "@/store/modules/user";
import {
  createPasswordByteValidator,
  PASSWORD_PLACEHOLDER
} from "@/utils/password";
import {
  listAdministratorsApi,
  createAdministratorApi,
  updateAdministratorApi,
  enableAdministratorApi,
  disableAdministratorApi,
  resetAdministratorPasswordApi,
  assignAdministratorRolesApi
} from "@/api/administrators";
import { listRolesApi } from "@/api/roles";
import type { Administrator, Role } from "@/api/contract";
import type { FormInstance, FormRules } from "element-plus";

defineOptions({
  name: "Administrator"
});

const userStore = useUserStoreHook();
const loading = ref(false);
const list = ref<Administrator[]>([]);
const total = ref(0);
const roleOptions = ref<Role[]>([]);

const query = reactive({
  page: 1,
  page_size: 20,
  q: ""
});

/** 表格面板：数据加载 */
async function loadList() {
  loading.value = true;
  try {
    const res = await listAdministratorsApi({
      page: query.page,
      page_size: query.page_size,
      q: query.q || undefined
    });
    list.value = res?.data?.items ?? [];
    total.value = res?.data?.total ?? 0;
  } catch (error) {
    message(getErrorMessage(error), { type: "error" });
  } finally {
    loading.value = false;
  }
}

/** 加载角色选项（用于分配角色；普通管理员只能选非 super 角色，由后端最终校验） */
async function loadRoles() {
  try {
    const res = await listRolesApi();
    roleOptions.value = res?.data?.items ?? [];
  } catch {
    roleOptions.value = [];
  }
}

function handleQuery() {
  query.page = 1;
  loadList();
}

function handleReset() {
  query.q = "";
  query.page = 1;
  loadList();
}

// ---------- 新建 / 编辑 ----------
const dialogVisible = ref(false);
const dialogTitle = ref("新建管理员");
const dialogLoading = ref(false);
const formRef = ref<FormInstance>();
const editingId = ref<string | null>(null);
const form = reactive({
  username: "",
  display_name: "",
  password: "",
  role_codes: [] as string[]
});

const createRules: FormRules = {
  username: [{ required: true, message: "请输入账号", trigger: "blur" }],
  password: [
    { required: true, message: "请输入初始密码", trigger: "blur" },
    createPasswordByteValidator()
  ],
  role_codes: [{ required: true, message: "请选择角色", trigger: "change" }]
};

const editRules: FormRules = {
  display_name: [{ required: true, message: "请输入显示名称", trigger: "blur" }]
};

function openCreate() {
  editingId.value = null;
  dialogTitle.value = "新建管理员";
  Object.assign(form, {
    username: "",
    display_name: "",
    password: "",
    role_codes: []
  });
  dialogVisible.value = true;
}

function openEdit(row: Administrator) {
  editingId.value = row.id;
  dialogTitle.value = "编辑管理员";
  Object.assign(form, {
    username: row.username,
    display_name: row.display_name,
    password: "",
    role_codes: [...row.role_codes]
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
        await updateAdministratorApi(editingId.value, {
          display_name: form.display_name
        });
      } else {
        await createAdministratorApi({
          username: form.username,
          password: form.password,
          display_name: form.display_name || undefined,
          role_codes: form.role_codes
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

// ---------- 启用 / 禁用（含二次确认） ----------
const enableDialogVisible = ref(false);
const enableLoading = ref(false);
const enableTarget = ref<Administrator | null>(null);
const enableAction = ref<"enable" | "disable">("disable");

function openToggle(row: Administrator) {
  enableTarget.value = row;
  enableAction.value = row.enabled ? "disable" : "enable";
  enableDialogVisible.value = true;
}

async function confirmToggle() {
  if (!enableTarget.value) return;
  const row = enableTarget.value;
  const isDisable = enableAction.value === "disable";
  enableLoading.value = true;
  try {
    if (isDisable) {
      await disableAdministratorApi(row.id);
      message(`已禁用「${row.username}」`, { type: "success" });
    } else {
      await enableAdministratorApi(row.id);
      message(`已启用「${row.username}」`, { type: "success" });
    }
    enableDialogVisible.value = false;
    loadList();
  } catch (error) {
    message(getErrorMessage(error), { type: "error" });
  } finally {
    enableLoading.value = false;
  }
}

// ---------- 重置密码 ----------
const resetDialogVisible = ref(false);
const resetLoading = ref(false);
const resetFormRef = ref<FormInstance>();
const resetTarget = ref<Administrator | null>(null);
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

function openReset(row: Administrator) {
  resetTarget.value = row;
  resetForm.new_password = "";
  resetForm.confirm = "";
  resetDialogVisible.value = true;
}

async function submitReset() {
  if (!resetFormRef.value || !resetTarget.value) return;
  await resetFormRef.value.validate(async valid => {
    if (!valid) return;
    resetLoading.value = true;
    try {
      await resetAdministratorPasswordApi(resetTarget.value!.id, {
        new_password: resetForm.new_password
      });
      message("密码已重置", { type: "success" });
      resetDialogVisible.value = false;
    } catch (error) {
      message(getErrorMessage(error), { type: "error" });
    } finally {
      resetLoading.value = false;
    }
  });
}

// ---------- 分配角色 ----------
const roleDialogVisible = ref(false);
const roleLoading = ref(false);
const roleFormRef = ref<FormInstance>();
const roleTarget = ref<Administrator | null>(null);
const roleForm = reactive({ role_codes: [] as string[] });

function openAssignRoles(row: Administrator) {
  roleTarget.value = row;
  roleForm.role_codes = [...row.role_codes];
  roleDialogVisible.value = true;
}

async function submitRoles() {
  if (!roleTarget.value) return;
  roleLoading.value = true;
  try {
    await assignAdministratorRolesApi(roleTarget.value.id, {
      role_codes: roleForm.role_codes
    });
    message("角色已更新", { type: "success" });
    roleDialogVisible.value = false;
    loadList();
  } catch (error) {
    message(getErrorMessage(error), { type: "error" });
  } finally {
    roleLoading.value = false;
  }
}

/** 是否当前登录账号自身（自身不可禁用/重置/编辑角色，由后端最终校验） */
function isSelf(row: Administrator): boolean {
  return row.id === userStore.adminId;
}

function roleName(code: string): string {
  const found = roleOptions.value.find(item => item.code === code);
  return found ? found.name : code;
}

onMounted(() => {
  loadList();
  loadRoles();
});
</script>

<template>
  <div class="flex flex-col gap-4">
    <!-- 操作/筛选面板 -->
    <div class="panel-container">
      <div class="panel-body">
        <div class="flex flex-wrap items-center justify-between gap-3">
          <div class="flex flex-wrap items-center gap-2">
            <el-input
              v-model="query.q"
              placeholder="账号 / 显示名称"
              clearable
              style="width: 220px"
              @keyup.enter="handleQuery"
            />
          </div>
          <div class="flex items-center gap-2">
            <el-button type="primary" @click="handleQuery">查询</el-button>
            <el-button @click="handleReset">重置</el-button>
            <el-button
              v-perms="['admin.user.create']"
              type="primary"
              plain
              @click="openCreate"
            >
              新建管理员
            </el-button>
          </div>
        </div>
      </div>
    </div>

    <!-- 表格面板 -->
    <div class="panel-container">
      <div class="panel-header">
        <span class="panel-title">管理员列表</span>
        <el-button text :loading="loading" @click="loadList">刷新</el-button>
      </div>
      <div class="panel-body table-body">
        <el-table v-loading="loading" :data="list" stripe>
          <el-table-column prop="id" label="ID" width="80" />
          <el-table-column prop="username" label="账号" min-width="120" />
          <el-table-column
            prop="display_name"
            label="显示名称"
            min-width="140"
          />
          <el-table-column label="状态" width="90">
            <template #default="{ row }">
              <el-tag :type="row.enabled ? 'success' : 'danger'" size="small">
                {{ row.enabled ? "启用" : "禁用" }}
              </el-tag>
            </template>
          </el-table-column>
          <el-table-column label="角色" min-width="180">
            <template #default="{ row }">
              <el-tag
                v-for="code in row.role_codes"
                :key="code"
                size="small"
                class="mr-1"
                :type="code === 'super_admin' ? 'warning' : 'info'"
              >
                {{ roleName(code) }}
              </el-tag>
            </template>
          </el-table-column>
          <el-table-column prop="created_at" label="创建时间" min-width="170" />
          <el-table-column label="操作" width="280" fixed="right">
            <template #default="{ row }">
              <el-button
                v-perms="['admin.user.update']"
                link
                type="primary"
                @click="openEdit(row)"
              >
                编辑
              </el-button>
              <el-button
                v-perms="['admin.user.disable']"
                link
                type="warning"
                :disabled="isSelf(row)"
                @click="openToggle(row)"
              >
                {{ row.enabled ? "禁用" : "启用" }}
              </el-button>
              <el-button
                v-perms="['admin.user.reset_password']"
                link
                type="danger"
                :disabled="isSelf(row)"
                @click="openReset(row)"
              >
                重置密码
              </el-button>
              <el-button
                v-perms="['admin.user.assign_role']"
                link
                type="primary"
                :disabled="isSelf(row)"
                @click="openAssignRoles(row)"
              >
                分配角色
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
      width="520px"
      destroy-on-close
    >
      <el-form
        ref="formRef"
        :model="form"
        :rules="editingId ? editRules : createRules"
        label-width="100px"
      >
        <el-form-item v-if="!editingId" label="账号" prop="username">
          <el-input v-model="form.username" placeholder="登录账号" />
        </el-form-item>
        <el-form-item v-if="editingId" label="账号">
          <el-input :model-value="form.username" disabled />
        </el-form-item>
        <el-form-item label="显示名称" prop="display_name">
          <el-input v-model="form.display_name" placeholder="显示名称" />
        </el-form-item>
        <el-form-item v-if="!editingId" label="初始密码" prop="password">
          <el-input
            v-model="form.password"
            type="password"
            show-password
            :placeholder="PASSWORD_PLACEHOLDER"
          />
        </el-form-item>
        <el-form-item label="角色" prop="role_codes">
          <el-select
            v-model="form.role_codes"
            multiple
            style="width: 100%"
            placeholder="请选择角色"
          >
            <el-option
              v-for="item in roleOptions"
              :key="item.code"
              :label="`${item.name}（${item.code}）`"
              :value="item.code"
              :disabled="!item.enabled"
            />
          </el-select>
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="dialogVisible = false">取消</el-button>
        <el-button
          type="primary"
          :loading="dialogLoading"
          @click="submitDialog"
        >
          保存
        </el-button>
      </template>
    </el-dialog>

    <!-- 启用/禁用确认对话框 -->
    <el-dialog
      v-model="enableDialogVisible"
      :title="enableAction === 'disable' ? '禁用管理员' : '启用管理员'"
      width="440px"
      destroy-on-close
    >
      <el-alert
        :type="enableAction === 'disable' ? 'warning' : 'info'"
        :closable="false"
        :title="
          enableAction === 'disable'
            ? `确定禁用「${enableTarget?.username ?? ''}」吗？禁用后该账号将无法登录。`
            : `确定启用「${enableTarget?.username ?? ''}」吗？`
        "
        class="mb-3"
      />
      <p v-if="enableAction === 'disable'" class="text-sm text-secondary">
        已登录的会话将在其下一次受保护请求时被拒绝。
      </p>
      <template #footer>
        <el-button @click="enableDialogVisible = false">取消</el-button>
        <el-button
          :type="enableAction === 'disable' ? 'danger' : 'primary'"
          :loading="enableLoading"
          @click="confirmToggle"
        >
          {{ enableAction === "disable" ? "确认禁用" : "确认启用" }}
        </el-button>
      </template>
    </el-dialog>

    <!-- 重置密码对话框 -->
    <el-dialog
      v-model="resetDialogVisible"
      title="重置密码"
      width="480px"
      destroy-on-close
    >
      <el-alert
        type="warning"
        :closable="false"
        :title="`将重置「${resetTarget?.username ?? ''}」的密码，重置后需告知对方使用新密码登录。`"
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
        <el-button @click="resetDialogVisible = false">取消</el-button>
        <el-button type="danger" :loading="resetLoading" @click="submitReset">
          确认重置
        </el-button>
      </template>
    </el-dialog>

    <!-- 分配角色对话框 -->
    <el-dialog
      v-model="roleDialogVisible"
      :title="`分配角色：${roleTarget?.username ?? ''}`"
      width="480px"
      destroy-on-close
    >
      <el-form ref="roleFormRef" :model="roleForm" label-width="100px">
        <el-form-item label="角色" prop="role_codes">
          <el-select
            v-model="roleForm.role_codes"
            multiple
            style="width: 100%"
            placeholder="请选择角色"
          >
            <el-option
              v-for="item in roleOptions"
              :key="item.code"
              :label="`${item.name}（${item.code}）`"
              :value="item.code"
              :disabled="!item.enabled"
            />
          </el-select>
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="roleDialogVisible = false">取消</el-button>
        <el-button type="primary" :loading="roleLoading" @click="submitRoles">
          保存
        </el-button>
      </template>
    </el-dialog>
  </div>
</template>

<style scoped>
@import url("@/style/business.scss");
</style>
