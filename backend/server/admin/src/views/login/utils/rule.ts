import { reactive } from "vue";
import type { FormRules } from "element-plus";

/**
 * 登录校验：密码格式与强度由后端统一约束（新建/重置密码 8-72 字节），
 * 登录页只做必填校验，避免前端正则与后端策略不一致。
 */
const loginRules = reactive<FormRules>({
  username: [
    {
      required: true,
      message: "请输入账号",
      trigger: "blur"
    }
  ],
  password: [
    {
      required: true,
      message: "请输入密码",
      trigger: "blur"
    }
  ]
});

export { loginRules };
