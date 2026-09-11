import { describe, it, expect } from "vitest";
import { auditActionLabel, AUDIT_ACTION_LABELS } from "@/utils/auditActions";

describe("审计动作中文化（ADM-003）", () => {
  it("覆盖工单要求的全部动作", () => {
    const required = [
      "auth.login",
      "auth.logout",
      "auth.refresh_replay",
      "auth.change_password",
      "administrator.create",
      "administrator.update",
      "administrator.enable",
      "administrator.disable",
      "administrator.reset_password",
      "administrator.assign_roles",
      "administrator.profile_update",
      "role.create",
      "role.update",
      "role.assign_permissions",
      "user.register",
      "user.login",
      "user.logout",
      "user.refresh_replay",
      "user.verify_email",
      "user.resend_verification",
      "user.create",
      "user.update",
      "user.enable",
      "user.disable",
      "user.reset_password",
      "user.points_adjust",
      "user.level_assign",
      "user_level.create",
      "user_level.update",
      "system_settings.update"
    ];
    for (const action of required) {
      expect(AUDIT_ACTION_LABELS[action], action).toBeTruthy();
    }
  });

  it("已知动作返回中文", () => {
    expect(auditActionLabel("auth.login")).toBe("管理员登录");
    expect(auditActionLabel("administrator.profile_update")).toBe(
      "修改个人资料"
    );
    expect(auditActionLabel("role.assign_permissions")).toBe("分配角色权限");
    expect(auditActionLabel("user.points_adjust")).toBe("调整业务用户积分");
    expect(auditActionLabel("user_level.update")).toBe("修改用户等级");
    expect(auditActionLabel("system_settings.update")).toBe("更新系统设置");
  });

  it("未知动作安全回退为原始英文代码", () => {
    expect(auditActionLabel("future.unknown_action")).toBe(
      "future.unknown_action"
    );
    expect(auditActionLabel("")).toBe("—");
  });
});
