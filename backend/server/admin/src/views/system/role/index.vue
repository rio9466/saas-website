<script setup lang="ts">
import { computed, onMounted, reactive, ref } from "vue";
import { message } from "@/utils/message";
import { getErrorMessage } from "@/utils/error";
import { hasPerms } from "@/utils/perms";
import {
  listRolesApi,
  createRoleApi,
  updateRoleApi,
  replaceRolePermissionsApi,
  listPermissionsApi
} from "@/api/roles";
import type { Role, Permission } from "@/api/contract";
import type { FormInstance, FormRules } from "element-plus";

defineOptions({
  name: "Role"
});

const loading = ref(false);
const list = ref<Role[]>([]);
const permissionCatalog = ref<Permission[]>([]);

/** 是否拥有角色管理权限（仅 super_admin 持有 admin.role.manage） */
const canManage = computed(() => hasPerms(["admin.role.manage"]));

async function loadList() {
  loading.value = true;
  try {
    const res = await listRolesApi();
    list.value = res?.data?.items ?? [];
  } catch (error) {
    message(getErrorMessage(error), { type: "error" });
  } finally {
    loading.value = false;
  }
}

async function loadPermissions() {
  try {
    const res = await listPermissionsApi();
    permissionCatalog.value = res?.data?.items ?? [];
  } catch {
    permissionCatalog.value = [];
  }
}

/** 按权限码前缀分组（如 admin.user.* / dashboard.* / audit.*） */
const permissionGroups = computed(() => {
  const groups = new Map<string, Permission[]>();
  permissionCatalog.value.forEach(item => {
    const area = item.code.split(".").slice(0, 2).join(".");
    const arr = groups.get(area) ?? [];
    arr.push(item);
    groups.set(area, arr);
  });
  return Array.from(groups.entries()).map(([area, items]) => ({
    area,
    items
  }));
});

function rolePermissionNames(role: Role): string {
  return role.permission_codes.length ? role.permission_codes.join("、") : "—";
}

// ---------- 新建 / 编辑 ----------
const dialogVisible = ref(false);
const dialogTitle = ref("新建角色");
const dialogLoading = ref(false);
const formRef = ref<FormInstance>();
const editingId = ref<string | null>(null);
const form = reactive({
  code: "",
  name: "",
  description: "",
  enabled: true
});

const createRules: FormRules = {
  code: [{ required: true, message: "请输入角色代码", trigger: "blur" }],
  name: [{ required: true, message: "请输入角色名称", trigger: "blur" }]
};

const editRules: FormRules = {
  name: [{ required: true, message: "请输入角色名称", trigger: "blur" }]
};

function openCreate() {
  editingId.value = null;
  dialogTitle.value = "新建角色";
  Object.assign(form, {
    code: "",
    name: "",
    description: "",
    enabled: true
  });
  dialogVisible.value = true;
}

function openEdit(row: Role) {
  editingId.value = row.id;
  dialogTitle.value = "编辑角色";
  Object.assign(form, {
    code: row.code,
    name: row.name,
    description: row.description,
    enabled: row.enabled
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
        await updateRoleApi(editingId.value, {
          name: form.name,
          description: form.description || undefined,
          enabled: form.enabled
        });
      } else {
        await createRoleApi({
          code: form.code,
          name: form.name,
          description: form.description || undefined
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

// ---------- 分配权限 ----------
const permDialogVisible = ref(false);
const permLoading = ref(false);
const permTarget = ref<Role | null>(null);
const permForm = reactive({ permission_codes: [] as string[] });

function openAssignPermissions(row: Role) {
  permTarget.value = row;
  permForm.permission_codes = [...row.permission_codes];
  permDialogVisible.value = true;
}

async function submitPermissions() {
  if (!permTarget.value) return;
  permLoading.value = true;
  try {
    await replaceRolePermissionsApi(permTarget.value.id, {
      permission_codes: permForm.permission_codes
    });
    message("权限已更新", { type: "success" });
    permDialogVisible.value = false;
    loadList();
  } catch (error) {
    message(getErrorMessage(error), { type: "error" });
  } finally {
    permLoading.value = false;
  }
}

onMounted(() => {
  loadList();
  loadPermissions();
});
</script>

<template>
  <div class="flex flex-col gap-4">
    <!-- 操作面板 -->
    <div class="panel-container">
      <div class="panel-body">
        <div class="flex items-center justify-between">
          <div class="text-sm text-secondary">
            {{
              canManage
                ? "可创建、编辑角色并分配权限"
                : "只读：当前账号无角色管理权限"
            }}
          </div>
          <el-button
            v-if="canManage"
            v-perms="['admin.role.manage']"
            type="primary"
            @click="openCreate"
          >
            新建角色
          </el-button>
        </div>
      </div>
    </div>

    <!-- 表格面板 -->
    <div class="panel-container">
      <div class="panel-header">
        <span class="panel-title">角色列表</span>
        <el-button text :loading="loading" @click="loadList">刷新</el-button>
      </div>
      <div class="panel-body table-body">
        <el-table v-loading="loading" :data="list" stripe>
          <el-table-column prop="code" label="角色代码" min-width="130" />
          <el-table-column prop="name" label="名称" min-width="150" />
          <el-table-column
            prop="description"
            label="描述"
            min-width="180"
            show-overflow-tooltip
          />
          <el-table-column label="类型" width="90">
            <template #default="{ row }">
              <el-tag size="small" :type="row.built_in ? 'warning' : 'info'">
                {{ row.built_in ? "内置" : "自定义" }}
              </el-tag>
            </template>
          </el-table-column>
          <el-table-column label="状态" width="80">
            <template #default="{ row }">
              <el-tag size="small" :type="row.enabled ? 'success' : 'danger'">
                {{ row.enabled ? "启用" : "禁用" }}
              </el-tag>
            </template>
          </el-table-column>
          <el-table-column label="权限" min-width="220">
            <template #default="{ row }">
              <span class="text-xs text-secondary">
                {{ rolePermissionNames(row) }}
              </span>
            </template>
          </el-table-column>
          <el-table-column
            v-if="canManage"
            label="操作"
            width="180"
            fixed="right"
          >
            <template #default="{ row }">
              <el-button
                v-perms="['admin.role.manage']"
                link
                type="primary"
                @click="openEdit(row)"
              >
                编辑
              </el-button>
              <el-button
                v-perms="['admin.role.manage']"
                link
                type="primary"
                @click="openAssignPermissions(row)"
              >
                分配权限
              </el-button>
            </template>
          </el-table-column>
        </el-table>
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
        <el-form-item v-if="!editingId" label="角色代码" prop="code">
          <el-input v-model="form.code" placeholder="如 data_analyst" />
        </el-form-item>
        <el-form-item v-if="editingId" label="角色代码">
          <el-input :model-value="form.code" disabled />
        </el-form-item>
        <el-form-item label="名称" prop="name">
          <el-input v-model="form.name" placeholder="角色名称" />
        </el-form-item>
        <el-form-item label="描述" prop="description">
          <el-input
            v-model="form.description"
            type="textarea"
            :rows="2"
            placeholder="角色说明（可选）"
          />
        </el-form-item>
        <el-form-item v-if="editingId" label="启用">
          <el-switch
            v-model="form.enabled"
            :disabled="form.code === 'super_admin'"
          />
          <span class="ml-2 text-xs text-secondary">
            内置 super_admin 角色不可禁用
          </span>
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

    <!-- 分配权限对话框 -->
    <el-dialog
      v-model="permDialogVisible"
      :title="`分配权限：${permTarget?.name ?? ''}`"
      width="640px"
      destroy-on-close
    >
      <div
        v-for="group in permissionGroups"
        :key="group.area"
        class="mb-3 rounded border border-[var(--el-border-color-lighter)] p-3"
      >
        <p
          class="mb-2 text-sm font-medium"
          style="color: var(--el-text-color-primary)"
        >
          {{ group.area }}.*
        </p>
        <el-checkbox-group v-model="permForm.permission_codes">
          <el-checkbox
            v-for="item in group.items"
            :key="item.code"
            :value="item.code"
            class="mr-4"
          >
            {{ item.name }}（{{ item.code }}）
          </el-checkbox>
        </el-checkbox-group>
      </div>
      <el-empty
        v-if="permissionCatalog.length === 0"
        description="权限目录为空"
        :image-size="60"
      />
      <template #footer>
        <el-button @click="permDialogVisible = false">取消</el-button>
        <el-button
          type="primary"
          :loading="permLoading"
          @click="submitPermissions"
        >
          保存
        </el-button>
      </template>
    </el-dialog>
  </div>
</template>

<style scoped>
@import url("@/style/business.scss");
</style>
