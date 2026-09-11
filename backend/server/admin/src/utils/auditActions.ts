/**
 * 审计动作中文化映射。
 *
 * 审计数据库始终保存稳定的英文动作代码（不可变更历史数据）；这里只做
 * 前端展示翻译。列表、详情、筛选下拉都使用本映射；未知动作回退为原始代码。
 */
export const AUDIT_ACTION_LABELS: Record<string, string> = {
  "auth.login": "管理员登录",
  "auth.logout": "管理员退出",
  "auth.refresh_replay": "刷新令牌重放",
  "auth.change_password": "修改个人密码",
  "administrator.create": "新增管理员",
  "administrator.update": "修改管理员",
  "administrator.enable": "启用管理员",
  "administrator.disable": "禁用管理员",
  "administrator.reset_password": "重置管理员密码",
  "administrator.assign_roles": "分配管理员角色",
  "administrator.profile_update": "修改个人资料",
  "role.create": "新增角色",
  "role.update": "修改角色",
  "role.assign_permissions": "分配角色权限",
  "user.register": "用户注册",
  "user.login": "用户登录",
  "user.logout": "用户退出",
  "user.refresh_replay": "用户刷新令牌重放",
  "user.verify_email": "用户邮箱验证",
  "user.resend_verification": "重发验证邮件",
  "user.create": "新建业务用户",
  "user.update": "修改业务用户",
  "user.enable": "启用业务用户",
  "user.disable": "禁用业务用户",
  "user.reset_password": "重置业务用户密码",
  "user.points_adjust": "调整业务用户积分",
  "user.level_assign": "设置业务用户等级",
  "user_level.create": "新增用户等级",
  "user_level.update": "修改用户等级",
  "system_settings.update": "更新系统设置"
};

/** 返回动作的中文标签；未知动作安全回退为原始代码。 */
export function auditActionLabel(action: string): string {
  if (!action) return "—";
  return AUDIT_ACTION_LABELS[action] ?? action;
}

/** 筛选项：中文标签 → 后端原始动作代码 */
export const AUDIT_ACTION_FILTER_OPTIONS = Object.entries(
  AUDIT_ACTION_LABELS
).map(([value, label]) => ({ value, label }));
