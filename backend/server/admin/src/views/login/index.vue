<script setup lang="ts">
import Motion from "./utils/motion";
import { useRouter } from "vue-router";
import { message } from "@/utils/message";
import { loginRules } from "./utils/rule";
import { ref, reactive, toRaw } from "vue";
import { getErrorMessage } from "@/utils/error";
import { createLoginSubmit } from "@/utils/loginSubmit";
import { useNav } from "@/layout/hooks/useNav";
import type { FormInstance } from "element-plus";
import { useLayout } from "@/layout/hooks/useLayout";
import { useUserStoreHook } from "@/store/modules/user";
import { initRouter, getTopMenu } from "@/router/utils";
import { bg, illustration } from "./utils/static";
import { useRenderIcon } from "@/components/ReIcon/src/hooks";
import { useDataThemeChange } from "@/layout/hooks/useDataThemeChange";

import dayIcon from "@/assets/svg/day.svg?component";
import darkIcon from "@/assets/svg/dark.svg?component";
import Lock from "~icons/ri/lock-fill";
import User from "~icons/ri/user-3-fill";

defineOptions({
  name: "Login"
});

const router = useRouter();
const loading = ref(false);
const ruleFormRef = ref<FormInstance>();

const { initStorage } = useLayout();
initStorage();

const { dataTheme, overallStyle, dataThemeChange } = useDataThemeChange();
dataThemeChange(overallStyle.value);

const ruleForm = reactive({
  username: "",
  password: ""
});

/**
 * 单一提交入口：el-form 的 submit.prevent（点击/Enter 均走这里）。
 * createLoginSubmit 在第一次异步等待前同步加锁，pending 期间的重复
 * 提交（快速连点/连续 Enter）会被直接忽略，不会产生并发登录请求。
 */
const loginSubmit = createLoginSubmit({
  validate: () => {
    const formEl = ruleFormRef.value;
    if (!formEl) return Promise.resolve(false);
    return formEl.validate().then(
      () => true,
      () => false
    );
  },
  doLogin: async () => {
    await useUserStoreHook().loginByUsername({
      username: ruleForm.username.trim(),
      password: ruleForm.password
    });
    // 本地路由 + /me 角色权限过滤
    await initRouter();
    await router.push(getTopMenu(true).path);
  },
  onSuccess: () => {
    message("登录成功", { type: "success" });
  },
  onError: (error: unknown) => {
    // 后端统一错误信息（不暴露账号是否存在）
    message(getErrorMessage(error), { type: "error" });
  }
});

async function onLoginSubmit() {
  loading.value = true;
  try {
    await loginSubmit.submit();
  } finally {
    loading.value = false;
  }
}
</script>

<template>
  <div class="select-none">
    <img :src="bg" class="wave" />
    <div class="flex-c absolute right-5 top-3">
      <!-- 主题 -->
      <el-switch
        v-model="dataTheme"
        inline-prompt
        :active-icon="dayIcon"
        :inactive-icon="darkIcon"
        @change="dataThemeChange"
      />
    </div>
    <div class="login-container" :class="{ 'dark-mode': dataTheme }">
      <div class="img">
        <component :is="toRaw(illustration)" />
      </div>
      <div class="login-box">
        <div class="login-form">
          <!-- 品牌区域只保留文字 easy-admin，不显示 icon/logo 图片 -->
          <Motion>
            <p class="login-brand">easy-admin</p>
          </Motion>
          <Motion :delay="50">
            <h2 class="outline-hidden">管理后台登录</h2>
          </Motion>

          <el-form
            ref="ruleFormRef"
            :model="ruleForm"
            :rules="loginRules"
            size="large"
            @submit.prevent="onLoginSubmit"
          >
            <Motion :delay="100">
              <el-form-item
                :rules="[
                  {
                    required: true,
                    message: '请输入账号',
                    trigger: 'blur'
                  }
                ]"
                prop="username"
              >
                <el-input
                  v-model="ruleForm.username"
                  clearable
                  placeholder="账号"
                  :prefix-icon="useRenderIcon(User)"
                />
              </el-form-item>
            </Motion>

            <Motion :delay="150">
              <el-form-item prop="password">
                <el-input
                  v-model="ruleForm.password"
                  clearable
                  show-password
                  placeholder="密码"
                  :prefix-icon="useRenderIcon(Lock)"
                />
              </el-form-item>
            </Motion>

            <Motion :delay="250">
              <el-button
                class="w-full mt-4!"
                size="default"
                type="primary"
                native-type="submit"
                :loading="loading"
              >
                登录
              </el-button>
            </Motion>
          </el-form>
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped>
@import url("@/style/login.css");
</style>

<style lang="scss" scoped>
/* 品牌文字：纯文字 easy-admin，无图标 */
.login-brand {
  margin: 0 0 4px;
  font-size: 28px;
  font-weight: 700;
  color: var(--el-text-color-primary);
  letter-spacing: 0.02em;
}

:deep(.el-input-group__append, .el-input-group__prepend) {
  padding: 0;
}
</style>
