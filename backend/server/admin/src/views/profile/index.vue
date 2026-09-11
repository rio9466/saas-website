<script setup lang="ts">
import { reactive, ref } from "vue";
import { useRouter } from "vue-router";
import { storeToRefs } from "pinia";
import { message } from "@/utils/message";
import { getErrorMessage } from "@/utils/error";
import { useUserStoreHook } from "@/store/modules/user";
import {
  createPasswordByteValidator,
  PASSWORD_PLACEHOLDER
} from "@/utils/password";
import { resetRouter } from "@/router";
import { useMultiTagsStoreHook } from "@/store/modules/multiTags";
import { routerArrays } from "@/layout/types";
import type { FormInstance, FormRules } from "element-plus";

defineOptions({
  name: "ProfileIndex"
});

const router = useRouter();
const userStore = useUserStoreHook();
const { username, nickname } = storeToRefs(userStore);

// ---------- 编辑自己的显示名称 ----------
const profileFormRef = ref<FormInstance>();
const profileSaving = ref(false);
const profileForm = reactive({
  display_name: ""
});
// 预填当前显示名称（/me 已在登录/冷启动时加载）
profileForm.display_name = nickname.value ?? "";

const profileRules: FormRules = {
  display_name: [
    {
      validator: (_, value, callback) => {
        if (!String(value ?? "").trim()) {
          callback(new Error("显示名称不能为空"));
        } else {
          callback();
        }
      },
      trigger: "blur"
    }
  ]
};

/** 保存显示名称：调用 PATCH /me，成功后同步 Pinia（导航栏/本页立即更新） */
async function saveDisplayName() {
  if (!profileFormRef.value) return;
  await profileFormRef.value.validate(async valid => {
    if (!valid) return;
    const trimmed = profileForm.display_name.trim();
    if (trimmed === (nickname.value ?? "")) {
      message("显示名称未变化", { type: "info" });
      return;
    }
    profileSaving.value = true;
    try {
      await userStore.updateMyProfile(trimmed);
      message("显示名称已更新", { type: "success" });
    } catch (error) {
      // 失败保留原名称并显示一次中文错误
      profileForm.display_name = nickname.value ?? "";
      message(getErrorMessage(error), { type: "error" });
    } finally {
      profileSaving.value = false;
    }
  });
}

// ---------- 修改密码 ----------
const passwordFormRef = ref<FormInstance>();
const passwordLoading = ref(false);
const passwordForm = reactive({
  current_password: "",
  new_password: "",
  confirm: ""
});

const passwordRules: FormRules = {
  current_password: [
    { required: true, message: "请输入当前密码", trigger: "blur" }
  ],
  new_password: [
    { required: true, message: "请输入新密码", trigger: "blur" },
    createPasswordByteValidator()
  ],
  confirm: [
    {
      validator: (_, value, callback) => {
        if (value !== passwordForm.new_password) {
          callback(new Error("两次输入的密码不一致"));
        } else {
          callback();
        }
      },
      trigger: "blur"
    }
  ]
};

/** 修改密码成功后后端吊销当前会话，需要重新登录 */
async function submitPassword() {
  if (!passwordFormRef.value) return;
  await passwordFormRef.value.validate(async valid => {
    if (!valid) return;
    passwordLoading.value = true;
    try {
      await userStore.changePassword({
        current_password: passwordForm.current_password,
        new_password: passwordForm.new_password
      });
      message("密码已修改，请重新登录", { type: "success" });
      userStore.resetAuthState();
      useMultiTagsStoreHook().handleTags("equal", [...routerArrays]);
      resetRouter();
      router.push("/login");
    } catch (error) {
      message(getErrorMessage(error), { type: "error" });
    } finally {
      passwordLoading.value = false;
    }
  });
}
</script>

<template>
  <div class="flex flex-col gap-4">
    <!-- 个人资料面板 -->
    <div class="panel-container">
      <div class="panel-header">
        <span class="panel-title">个人资料</span>
      </div>
      <div class="panel-body">
        <el-form
          ref="profileFormRef"
          :model="profileForm"
          :rules="profileRules"
          label-width="100px"
          style="max-width: 480px"
        >
          <el-form-item label="账号">
            <el-input :model-value="username || '—'" disabled />
          </el-form-item>
          <el-form-item label="显示名称" prop="display_name">
            <el-input
              v-model="profileForm.display_name"
              :placeholder="nickname || '显示名称'"
              clearable
            />
          </el-form-item>
          <el-form-item>
            <el-button
              type="primary"
              :loading="profileSaving"
              @click="saveDisplayName"
            >
              保存显示名称
            </el-button>
          </el-form-item>
        </el-form>
      </div>
    </div>

    <!-- 修改密码面板 -->
    <div class="panel-container">
      <div class="panel-header">
        <span class="panel-title">修改密码</span>
      </div>
      <div class="panel-body">
        <el-alert
          type="info"
          :closable="false"
          class="mb-3"
          title="修改成功后当前会话将被吊销，需要重新登录。"
        />
        <el-form
          ref="passwordFormRef"
          :model="passwordForm"
          :rules="passwordRules"
          label-width="100px"
          style="max-width: 480px"
        >
          <el-form-item label="当前密码" prop="current_password">
            <el-input
              v-model="passwordForm.current_password"
              type="password"
              show-password
              placeholder="当前登录密码"
            />
          </el-form-item>
          <el-form-item label="新密码" prop="new_password">
            <el-input
              v-model="passwordForm.new_password"
              type="password"
              show-password
              :placeholder="PASSWORD_PLACEHOLDER"
            />
          </el-form-item>
          <el-form-item label="确认新密码" prop="confirm">
            <el-input
              v-model="passwordForm.confirm"
              type="password"
              show-password
              placeholder="再次输入新密码"
            />
          </el-form-item>
          <el-form-item>
            <el-button
              type="primary"
              :loading="passwordLoading"
              @click="submitPassword"
            >
              确认修改
            </el-button>
          </el-form-item>
        </el-form>
      </div>
    </div>
  </div>
</template>

<style scoped>
@import url("@/style/business.scss");
</style>
