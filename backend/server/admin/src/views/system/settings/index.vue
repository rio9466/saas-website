<script setup lang="ts">
import { onMounted, reactive, ref } from "vue";
import { message } from "@/utils/message";
import { getErrorMessage } from "@/utils/error";
import {
  getSystemSettingsApi,
  updateSystemSettingsApi
} from "@/api/systemSettings";
import { listUserLevelsApi } from "@/api/userLevels";
import { toFourDecimal } from "@/utils/points";
import type { SystemSettings, UserLevel } from "@/api/contract";
import type { FormInstance, FormRules } from "element-plus";

defineOptions({ name: "SystemSettingsPage" });

const loading = ref(false);
const saving = ref(false);
const formRef = ref<FormInstance>();
const levelOptions = ref<UserLevel[]>([]);

const form = reactive({
  platform_name: "",
  public_frontend_url: "",
  public_api_url: "",
  registration_enabled: true,
  username_login_enabled: true,
  email_login_enabled: true,
  email_verification_required: false,
  default_level_id: "",
  default_avatar_url: "",
  registration_points: "0.0000",
  smtp_enabled: false,
  smtp_host: "",
  smtp_port: 587,
  smtp_username: "",
  smtp_password: "",
  smtp_from_email: "",
  smtp_from_name: "",
  smtp_tls_mode: "starttls" as "none" | "starttls" | "ssl",
  password_configured: false,
  version: 0
});

const urlRule = {
  type: "url" as const,
  message: "请输入有效的 http/https URL",
  trigger: "blur"
};

const rules: FormRules = {
  platform_name: [
    { required: true, message: "请输入平台名称", trigger: "blur" }
  ],
  public_frontend_url: [{ required: true, ...urlRule }],
  public_api_url: [{ required: true, ...urlRule }],
  smtp_host: [
    {
      validator: (_, value, callback) => {
        if (form.smtp_enabled && !value) {
          callback(new Error("启用 SMTP 时主机必填"));
        } else {
          callback();
        }
      },
      trigger: "blur"
    }
  ],
  smtp_from_email: [
    {
      validator: (_, value, callback) => {
        if (form.smtp_enabled && !/^[^@\s]+@[^@\s]+$/.test(value || "")) {
          callback(new Error("启用 SMTP 时请填写发件邮箱"));
        } else {
          callback();
        }
      },
      trigger: "blur"
    }
  ]
};

/** 左侧分类导航：与原单页表单的三个分组一一对应 */
const sections = [
  {
    key: "platform",
    title: "平台身份",
    description: "名称与公开 URL"
  },
  {
    key: "registration",
    title: "注册与登录",
    description: "注册开关、登录方式与默认等级"
  },
  {
    key: "email",
    title: "邮件与 SMTP",
    description: "邮箱验证与发信服务器"
  }
] as const;

type SectionKey = (typeof sections)[number]["key"];

const activeSection = ref<SectionKey>("platform");

function goToSection(key: SectionKey) {
  activeSection.value = key;
}

/** 带校验规则的字段所属分类；校验失败时切到对应分类，避免错误提示藏在未展示的区域 */
const sectionOfField: Record<string, SectionKey> = {
  platform_name: "platform",
  public_frontend_url: "platform",
  public_api_url: "platform",
  smtp_host: "email",
  smtp_from_email: "email"
};

function revealInvalidSection(invalidFields?: Record<string, unknown>) {
  const first = Object.keys(invalidFields ?? {})[0];
  const section = first ? sectionOfField[first] : undefined;
  if (section) activeSection.value = section;
}

/** 等级下拉项：与用户等级列表一致的「名称（阈值）」格式 */
function formatLevelOption(level: UserLevel): string {
  return `${level.name}（${level.threshold_points}）`;
}

async function loadLevels() {
  try {
    const res = await listUserLevelsApi();
    levelOptions.value = res?.data?.items ?? [];
  } catch {
    levelOptions.value = [];
  }
}

async function loadSettings() {
  loading.value = true;
  try {
    const res = await getSystemSettingsApi();
    const s = res?.data;
    if (!s) return;
    Object.assign(form, {
      platform_name: s.platform_name,
      public_frontend_url: s.public_frontend_url,
      public_api_url: s.public_api_url,
      registration_enabled: s.registration_enabled,
      username_login_enabled: s.username_login_enabled,
      email_login_enabled: s.email_login_enabled,
      email_verification_required: s.email_verification_required,
      default_level_id: s.default_level_id,
      default_avatar_url: s.default_avatar_url ?? "",
      registration_points: s.registration_points,
      smtp_enabled: s.smtp_enabled,
      smtp_host: s.smtp_host ?? "",
      smtp_port: s.smtp_port ?? 587,
      smtp_username: s.smtp_username ?? "",
      smtp_password: "",
      smtp_from_email: s.smtp_from_email ?? "",
      smtp_from_name: s.smtp_from_name ?? "",
      smtp_tls_mode:
        s.smtp_tls_mode === "none" || s.smtp_tls_mode === "ssl"
          ? s.smtp_tls_mode
          : "starttls",
      password_configured: s.password_configured,
      version: s.version
    });
  } catch (error) {
    message(getErrorMessage(error), { type: "error" });
  } finally {
    loading.value = false;
  }
}

async function save() {
  if (!formRef.value) return;
  await formRef.value.validate(async (valid, invalidFields) => {
    if (!valid) {
      revealInvalidSection(invalidFields);
      return;
    }
    saving.value = true;
    try {
      const res = await updateSystemSettingsApi({
        platform_name: form.platform_name,
        public_frontend_url: form.public_frontend_url,
        public_api_url: form.public_api_url,
        registration_enabled: form.registration_enabled,
        username_login_enabled: form.username_login_enabled,
        email_login_enabled: form.email_login_enabled,
        email_verification_required: form.email_verification_required,
        default_level_id: form.default_level_id,
        default_avatar_url: form.default_avatar_url || undefined,
        registration_points: toFourDecimal(form.registration_points),
        smtp_enabled: form.smtp_enabled,
        smtp_host: form.smtp_host,
        smtp_port: form.smtp_port,
        smtp_username: form.smtp_username,
        smtp_password: form.smtp_password || undefined,
        smtp_from_email: form.smtp_from_email,
        smtp_from_name: form.smtp_from_name,
        smtp_tls_mode: form.smtp_tls_mode,
        version: form.version
      });
      const s = res?.data;
      if (s) {
        form.version = s.version;
        form.password_configured = s.password_configured;
        form.smtp_password = "";
      }
      message("设置已保存", { type: "success" });
    } catch (error) {
      message(getErrorMessage(error), { type: "error" });
    } finally {
      saving.value = false;
    }
  });
}

onMounted(async () => {
  await loadLevels();
  await loadSettings();
});
</script>

<template>
  <div class="flex flex-col gap-4">
    <div class="panel-container">
      <div class="panel-header">
        <span class="panel-title">系统设置</span>
        <el-button text :loading="loading" @click="loadSettings"
          >重新载入</el-button
        >
      </div>
      <div class="panel-body settings-layout">
        <!-- 左侧：分类导航 -->
        <nav class="settings-nav" aria-label="设置分类">
          <button
            v-for="section in sections"
            :key="section.key"
            type="button"
            class="settings-nav-item"
            :class="{ 'is-active': section.key === activeSection }"
            @click="goToSection(section.key)"
          >
            <span class="settings-nav-title">{{ section.title }}</span>
            <span class="settings-nav-description">
              {{ section.description }}
            </span>
          </button>
        </nav>

        <!-- 右侧：当前分类表单。各分类用 v-show 切换，字段始终挂载，
             因此校验覆盖与改造前的单页表单完全一致。 -->
        <el-form
          ref="formRef"
          v-loading="loading"
          :model="form"
          :rules="rules"
          label-width="150px"
          class="settings-form"
        >
          <div v-show="activeSection === 'platform'" class="settings-section">
            <h4 class="settings-section-title">平台身份</h4>
            <el-form-item label="平台名称" prop="platform_name">
              <el-input
                v-model="form.platform_name"
                placeholder="用于注册/验证邮件抬头"
              />
            </el-form-item>
            <el-form-item label="前台 URL" prop="public_frontend_url">
              <el-input
                v-model="form.public_frontend_url"
                placeholder="如 https://app.example.com"
              />
              <div class="text-xs text-secondary mt-1">
                仅作为链接与邮件生成的公开元数据，不会改变服务监听地址
              </div>
            </el-form-item>
            <el-form-item label="公开 API URL" prop="public_api_url">
              <el-input
                v-model="form.public_api_url"
                placeholder="如 https://api.example.com"
              />
            </el-form-item>
          </div>

          <div
            v-show="activeSection === 'registration'"
            class="settings-section"
          >
            <h4 class="settings-section-title">注册与登录策略</h4>
            <el-form-item label="允许注册">
              <el-switch v-model="form.registration_enabled" />
            </el-form-item>
            <el-form-item label="用户名登录">
              <el-switch v-model="form.username_login_enabled" />
            </el-form-item>
            <el-form-item label="邮箱登录">
              <el-switch v-model="form.email_login_enabled" />
              <div class="text-xs text-secondary ml-2">
                两种登录方式不能同时关闭
              </div>
            </el-form-item>
            <el-form-item label="新用户默认等级">
              <el-select
                v-model="form.default_level_id"
                filterable
                style="width: 260px"
              >
                <el-option
                  v-for="l in levelOptions.filter(item => item.enabled)"
                  :key="l.id"
                  :label="formatLevelOption(l)"
                  :value="l.id"
                />
              </el-select>
            </el-form-item>
            <el-form-item label="默认头像 URL">
              <el-input
                v-model="form.default_avatar_url"
                placeholder="可留空"
              />
            </el-form-item>
            <el-form-item label="注册赠送积分">
              <el-input
                v-model="form.registration_points"
                style="width: 180px"
              />
              <div class="text-xs text-secondary mt-1">
                注册成功即按此值写入不可变账本
              </div>
            </el-form-item>
          </div>

          <div v-show="activeSection === 'email'" class="settings-section">
            <h4 class="settings-section-title">邮件验证与 SMTP</h4>
            <el-form-item label="要求邮箱验证">
              <el-switch v-model="form.email_verification_required" />
              <div class="text-xs text-secondary ml-2">
                开启要求完整、可解密且启用的 SMTP 配置
              </div>
            </el-form-item>
            <el-form-item label="启用 SMTP">
              <el-switch v-model="form.smtp_enabled" />
            </el-form-item>
            <template v-if="form.smtp_enabled">
              <el-form-item label="SMTP 主机" prop="smtp_host">
                <el-input
                  v-model="form.smtp_host"
                  placeholder="smtp.example.com"
                />
              </el-form-item>
              <el-form-item label="端口">
                <el-input-number
                  v-model="form.smtp_port"
                  :min="1"
                  :max="65535"
                />
              </el-form-item>
              <el-form-item label="TLS 模式">
                <el-select v-model="form.smtp_tls_mode" style="width: 200px">
                  <el-option label="STARTTLS（推荐）" value="starttls" />
                  <el-option label="隐式 TLS（SSL）" value="ssl" />
                  <el-option label="无加密（仅测试）" value="none" />
                </el-select>
              </el-form-item>
              <el-form-item label="用户名">
                <el-input v-model="form.smtp_username" autocomplete="off" />
              </el-form-item>
              <el-form-item label="密码">
                <el-input
                  v-model="form.smtp_password"
                  type="password"
                  show-password
                  autocomplete="new-password"
                  :placeholder="
                    form.password_configured
                      ? '已配置（留空保持不变，仅写入不回读）'
                      : '未配置'
                  "
                />
                <div class="text-xs text-secondary mt-1">
                  SMTP
                  密码仅可写入：留空表示保留现有密钥；接口永远不返回明文或密文。
                </div>
              </el-form-item>
              <el-form-item label="发件邮箱" prop="smtp_from_email">
                <el-input
                  v-model="form.smtp_from_email"
                  placeholder="no-reply@example.com"
                />
              </el-form-item>
              <el-form-item label="发件人名称">
                <el-input
                  v-model="form.smtp_from_name"
                  placeholder="如 easy-admin"
                />
              </el-form-item>
            </template>
          </div>

          <el-form-item class="settings-actions">
            <el-button type="primary" :loading="saving" @click="save"
              >保存设置</el-button
            >
            <el-button @click="loadSettings">放弃修改</el-button>
          </el-form-item>
        </el-form>
      </div>
    </div>
  </div>
</template>

<style scoped>
@import url("@/style/business.scss");

/* 窄屏：分类导航移到顶部横向排列，表单单列铺满 */
@media (width <= 768px) {
  .settings-layout {
    flex-direction: column;
    gap: 16px;
  }

  .settings-nav {
    flex-flow: row wrap;
    gap: 8px;
    width: 100%;
  }

  .settings-nav-item {
    padding: 6px 10px;
    border: 1px solid var(--el-border-color-lighter);
  }

  .settings-nav-description {
    display: none;
  }

  .settings-form {
    max-width: none;
  }

  :deep(.el-form-item) {
    display: block;
  }

  :deep(.el-form-item__label) {
    justify-content: flex-start;
    width: auto !important;
  }

  :deep(.el-form-item__content) {
    margin-left: 0 !important;
  }
}

/* 左分类导航 + 右表单 */
.settings-layout {
  display: flex;
  gap: 24px;
  align-items: stretch;
}

.settings-nav {
  display: flex;
  flex-shrink: 0;
  flex-direction: column;
  gap: 4px;
  width: 200px;
}

.settings-nav-item {
  display: flex;
  flex-direction: column;
  gap: 2px;
  padding: 10px 12px;
  color: var(--el-text-color-regular);
  text-align: left;
  cursor: pointer;
  background: transparent;
  border: none;
  border-radius: 4px;
}

.settings-nav-item:hover {
  color: var(--el-color-primary);
  background: var(--el-fill-color-light);
}

.settings-nav-item.is-active {
  color: var(--el-color-primary);
  background: var(--el-color-primary-light-9);
}

.settings-nav-title {
  font-size: 14px;
  font-weight: 500;
  line-height: 20px;
}

.settings-nav-description {
  font-size: 12px;
  line-height: 17px;
  color: var(--el-text-color-secondary);
}

.settings-form {
  flex: 1;
  min-width: 0;
  max-width: 640px;
}

.settings-section-title {
  margin: 0 0 16px;
  font-size: 15px;
  font-weight: 500;
  color: var(--el-text-color-primary);
}

.settings-actions {
  margin-top: 8px;
  margin-bottom: 0;
}
</style>
