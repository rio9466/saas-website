import { describe, it, expect } from "vitest";
import {
  filterRoutesByAuth,
  isOneOfArray,
  type RouteLike
} from "@/utils/routeFilter";

/**
 * 与 src/router/modules/* 对齐的本地路由树（不含 remaining.ts）：
 * super_admin / admin / finance 三种角色的可见性必须符合 PRD。
 */
const localRoutes: RouteLike[] = [
  {
    path: "/",
    redirect: "/dashboard",
    children: [
      {
        path: "/dashboard",
        name: "Dashboard",
        meta: { title: "工作台", permissions: ["dashboard.view"] }
      }
    ]
  },
  {
    path: "/system",
    name: "System",
    children: [
      {
        path: "/system/administrator",
        name: "Administrator",
        meta: {
          title: "管理员管理",
          roles: ["super_admin", "admin"],
          permissions: ["admin.user.read"]
        }
      },
      {
        path: "/system/role",
        name: "Role",
        meta: {
          title: "角色权限",
          roles: ["super_admin", "admin"],
          permissions: ["admin.role.read"]
        }
      },
      {
        path: "/system/audit",
        name: "Audit",
        meta: {
          title: "审计日志",
          roles: ["super_admin", "admin"],
          permissions: ["audit.log.read"]
        }
      }
    ]
  },
  {
    path: "/profile",
    name: "Profile",
    children: [
      {
        path: "/profile/index",
        name: "ProfileIndex",
        meta: { title: "个人中心" }
      }
    ]
  }
];

const superAdmin = {
  roles: ["super_admin"],
  permissions: [
    "dashboard.view",
    "admin.user.read",
    "admin.user.create",
    "admin.user.update",
    "admin.user.disable",
    "admin.user.reset_password",
    "admin.user.assign_role",
    "admin.role.read",
    "admin.role.manage",
    "audit.log.read"
  ]
};

const admin = {
  roles: ["admin"],
  permissions: [
    "dashboard.view",
    "admin.user.read",
    "admin.user.create",
    "admin.user.update",
    "admin.user.disable",
    "admin.user.reset_password",
    "admin.user.assign_role",
    "admin.role.read",
    "audit.log.read"
  ]
};

/** 本里程碑 finance 只有 dashboard.view（PRD：无财务占位页） */
const finance = {
  roles: ["finance"],
  permissions: ["dashboard.view"]
};

function flattenNames(routes: RouteLike[], acc: string[] = []): string[] {
  routes.forEach(r => {
    if (r.name) acc.push(r.name);
    if (r.children) flattenNames(r.children, acc);
  });
  return acc;
}

describe("本地路由 /me 权限过滤", () => {
  it("super_admin 可见全部业务菜单", () => {
    const tree = filterRoutesByAuth(
      localRoutes,
      superAdmin.roles,
      superAdmin.permissions
    );
    const names = flattenNames(tree);
    expect(names).toEqual(
      expect.arrayContaining([
        "Dashboard",
        "Administrator",
        "Role",
        "Audit",
        "ProfileIndex"
      ])
    );
  });

  it("admin 可见 dashboard + 系统管理三页（角色管理只读由页面控制）", () => {
    const tree = filterRoutesByAuth(
      localRoutes,
      admin.roles,
      admin.permissions
    );
    const names = flattenNames(tree);
    expect(names).toEqual(
      expect.arrayContaining(["Dashboard", "Administrator", "Role", "Audit"])
    );
  });

  it("finance 只可见 dashboard 与 profile，无系统管理菜单、无财务占位页", () => {
    const tree = filterRoutesByAuth(
      localRoutes,
      finance.roles,
      finance.permissions
    );
    const names = flattenNames(tree);
    expect(names).toContain("Dashboard");
    expect(names).toContain("ProfileIndex");
    expect(names).not.toContain("Administrator");
    expect(names).not.toContain("Role");
    expect(names).not.toContain("Audit");
    // 任何路由名都不允许包含 finance 占位
    expect(names.some(n => /finance/i.test(n))).toBe(false);
  });

  it("缺少 dashboard.view 权限时工作台不可见", () => {
    const tree = filterRoutesByAuth(
      localRoutes,
      ["admin"],
      ["admin.user.read"]
    );
    expect(flattenNames(tree)).not.toContain("Dashboard");
  });

  it("角色不匹配时系统管理目录整体消失（目录裁剪）", () => {
    const tree = filterRoutesByAuth(
      localRoutes,
      ["finance"],
      ["dashboard.view"]
    );
    expect(tree.find(r => r.path === "/system")?.children?.length ?? 0).toBe(0);
  });

  it("fail-closed：用户角色为空时，声明了 roles 的受保护菜单不可见（不视为不限制）", () => {
    const tree = filterRoutesByAuth(
      localRoutes,
      [],
      ["dashboard.view", "admin.user.read"]
    );
    expect(flattenNames(tree)).not.toContain("Administrator");
    expect(flattenNames(tree)).not.toContain("Role");
    expect(flattenNames(tree)).not.toContain("Audit");
    // 未声明 roles 的路由（dashboard/profile）仍然可见
    expect(flattenNames(tree)).toContain("Dashboard");
    expect(flattenNames(tree)).toContain("ProfileIndex");
  });

  it("fail-closed：用户权限为空时，声明了 permissions 的受保护路由不可见", () => {
    const tree = filterRoutesByAuth(localRoutes, ["super_admin"], []);
    expect(flattenNames(tree)).not.toContain("Dashboard");
    expect(flattenNames(tree)).not.toContain("Administrator");
    // 未声明 permissions 的 profile 仍然可见
    expect(flattenNames(tree)).toContain("ProfileIndex");
  });

  it("fail-closed：角色与权限双空时，除未声明要求的 profile 外全部隐藏", () => {
    const tree = filterRoutesByAuth(localRoutes, [], []);
    expect(flattenNames(tree)).toEqual(["Profile", "ProfileIndex"]);
  });

  it("部分权限：只有 admin.role.read 时角色页可见，其余管理员/审计页不可见", () => {
    const tree = filterRoutesByAuth(
      localRoutes,
      ["admin"],
      ["dashboard.view", "admin.role.read"]
    );
    const names = flattenNames(tree);
    expect(names).toContain("Role");
    expect(names).not.toContain("Administrator"); // 无 admin.user.read
    expect(names).not.toContain("Audit"); // 无 audit.log.read
  });
});

describe("isOneOfArray", () => {
  it("有交集返回 true", () => {
    expect(isOneOfArray(["admin"], ["super_admin", "admin"])).toBe(true);
  });
  it("无交集返回 false", () => {
    expect(isOneOfArray(["finance"], ["super_admin", "admin"])).toBe(false);
  });
  it("空数组不视为不限制：空与有值无交集返回 false", () => {
    expect(isOneOfArray([], ["finance"])).toBe(false);
    expect(isOneOfArray(["finance"], [])).toBe(false);
  });
  it("undefined 参数返回 true（未声明要求场景由上层短路处理）", () => {
    expect(isOneOfArray(undefined, ["finance"])).toBe(true);
  });
});
