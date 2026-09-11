import { describe, expect, it, vi } from "vitest";
import systemMenu from "@/router/modules/system";
import userMenu from "@/router/modules/user";

/**
 * 侧边栏只渲染 addIcon 注册过的离线图标（见 offlineIcon.ts 与 useRenderIcon），
 * 路由 meta.icon 若未注册会静默不显示。这里捕获注册表做一致性校验：
 * - 用桩替换 @iconify/vue 离线入口与 @pureadmin/utils，记录 addIcon 的名称；
 * - 用桩替换 unplugin-icons 的 `~icons/*?raw` 导入（vitest 未启用该插件）。
 */
const { registeredIcons } = vi.hoisted(() => ({
  registeredIcons: new Set<string>()
}));

vi.mock("@iconify/vue/dist/offline", () => ({
  addIcon: (name: string) => {
    registeredIcons.add(name);
  }
}));
vi.mock("@pureadmin/utils", () => ({
  getSvgInfo: (raw: string) => ({ body: raw })
}));
vi.mock("~icons/ep/home-filled?raw", () => ({ default: "<svg/>" }));
vi.mock("~icons/ep/setting?raw", () => ({ default: "<svg/>" }));
vi.mock("~icons/ep/user?raw", () => ({ default: "<svg/>" }));
vi.mock("~icons/ep/lock?raw", () => ({ default: "<svg/>" }));
vi.mock("~icons/ep/user-filled?raw", () => ({ default: "<svg/>" }));
vi.mock("~icons/ep/medal?raw", () => ({ default: "<svg/>" }));
vi.mock("~icons/ep/avatar?raw", () => ({ default: "<svg/>" }));
vi.mock("~icons/ri/search-line?raw", () => ({ default: "<svg/>" }));
vi.mock("~icons/ri/information-line?raw", () => ({ default: "<svg/>" }));
vi.mock("~icons/ri/file-list-3-line?raw", () => ({ default: "<svg/>" }));

/** 从系统管理菜单中按路由名取子路由 */
function findChild(name: string) {
  return systemMenu.children?.find(child => child.name === name);
}

/** 从用户管理父菜单中按路由名取子路由 */
function findUserChild(name: string) {
  return userMenu.children?.find(child => child.name === name);
}

describe("system menu routes", () => {
  it("keeps the business-user route under 用户管理 with unchanged semantics", () => {
    // 业务用户已迁移到用户管理父菜单，系统管理中不再直接挂载
    expect(findChild("UserManage")).toBeUndefined();
    expect(findChild("UserLevelManage")).toBeUndefined();
    // 系统管理自己的菜单（管理员/角色/审计/设置）不受影响
    expect(systemMenu.children?.map(child => child.name)).toEqual([
      "Administrator",
      "Role",
      "Audit",
      "SystemSettings"
    ]);
  });

  it("registers every system-menu icon in the offline icon registry", async () => {
    await import("@/components/ReIcon/src/offlineIcon");

    const menuIcons = [
      systemMenu.meta?.icon,
      ...(systemMenu.children ?? []).map(child => child.meta?.icon)
    ].filter((icon): icon is string => typeof icon === "string");

    expect(menuIcons.length).toBeGreaterThan(0);
    for (const icon of menuIcons) {
      expect(registeredIcons.has(icon), `未注册的菜单图标: ${icon}`).toBe(true);
    }
  });
});

describe("user management parent menu", () => {
  it("creates the 用户管理 parent menu containing both business pages", () => {
    expect(userMenu.meta?.title).toBe("用户管理");
    expect(userMenu.name).toBe("UserManagement");
    expect(userMenu.path).toBe("/user");
    expect(findUserChild("UserManage")?.path).toBe("/system/user");
    expect(findUserChild("UserLevelManage")?.path).toBe("/system/userLevel");
  });

  it("renames the business-user submenu to 业务用户 without changing path/name/roles/permissions", () => {
    const user = findUserChild("UserManage");
    expect(user?.meta?.title).toBe("业务用户");
    expect(user?.name).toBe("UserManage");
    expect(user?.path).toBe("/system/user");
    expect(user?.meta?.roles).toEqual(["super_admin", "admin"]);
    expect(user?.meta?.permissions).toEqual(["admin.customer.read"]);
  });

  it("keeps the user-level submenu semantics unchanged", () => {
    const userLevel = findUserChild("UserLevelManage");
    expect(userLevel?.path).toBe("/system/userLevel");
    expect(userLevel?.name).toBe("UserLevelManage");
    expect(userLevel?.meta?.title).toBe("用户等级");
    expect(userLevel?.meta?.roles).toEqual(["super_admin", "admin"]);
    expect(userLevel?.meta?.permissions).toEqual(["admin.user_level.read"]);
  });

  it("registers every user-menu icon in the offline icon registry", async () => {
    await import("@/components/ReIcon/src/offlineIcon");

    const menuIcons = [
      userMenu.meta?.icon,
      ...(userMenu.children ?? []).map(child => child.meta?.icon)
    ].filter((icon): icon is string => typeof icon === "string");

    for (const icon of menuIcons) {
      expect(registeredIcons.has(icon), `未注册的菜单图标: ${icon}`).toBe(true);
    }
    expect(userMenu.meta?.icon).toBe("ep/avatar");
  });
});
