const Layout = () => import("@/layout/index.vue");

export default {
  path: "/user",
  name: "UserManagement",
  component: Layout,
  redirect: "/system/user",
  meta: {
    icon: "ep/avatar",
    title: "用户管理",
    // 业务用户/用户等级的父级菜单，位于工作台与系统管理之间
    rank: 1
  },
  children: [
    {
      // 子路由 path/name/roles/permissions 保持不变（工单要求），
      // 仅菜单标题调整为“业务用户”以避免与父级“用户管理”同名。
      path: "/system/user",
      name: "UserManage",
      component: () => import("@/views/system/user/index.vue"),
      meta: {
        title: "业务用户",
        icon: "ep/user-filled",
        // 业务用户（前台用户）管理，仅 super_admin/admin 可见
        roles: ["super_admin", "admin"],
        permissions: ["admin.customer.read"]
      }
    },
    {
      path: "/system/userLevel",
      name: "UserLevelManage",
      component: () => import("@/views/system/userLevel/index.vue"),
      meta: {
        title: "用户等级",
        icon: "ep/medal",
        roles: ["super_admin", "admin"],
        permissions: ["admin.user_level.read"]
      }
    }
  ]
} satisfies RouteConfigsTable;
