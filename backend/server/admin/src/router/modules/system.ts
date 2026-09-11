const Layout = () => import("@/layout/index.vue");

export default {
  path: "/system",
  name: "System",
  component: Layout,
  redirect: "/system/administrator",
  meta: {
    icon: "ep/setting",
    title: "系统管理",
    rank: 2
  },
  children: [
    {
      path: "/system/administrator",
      name: "Administrator",
      component: () => import("@/views/system/administrator/index.vue"),
      meta: {
        title: "管理员管理",
        icon: "ep/user",
        // 普通管理员可管理非 super 账号；finance 无此菜单
        roles: ["super_admin", "admin"],
        permissions: ["admin.user.read"]
      }
    },
    {
      path: "/system/role",
      name: "Role",
      component: () => import("@/views/system/role/index.vue"),
      meta: {
        title: "角色权限",
        icon: "ep/lock",
        roles: ["super_admin", "admin"],
        permissions: ["admin.role.read"]
      }
    },
    {
      path: "/system/audit",
      name: "Audit",
      component: () => import("@/views/system/audit/index.vue"),
      meta: {
        title: "审计日志",
        icon: "ri/file-list-3-line",
        roles: ["super_admin", "admin"],
        permissions: ["audit.log.read"]
      }
    },
    {
      path: "/system/settings",
      name: "SystemSettings",
      component: () => import("@/views/system/settings/index.vue"),
      meta: {
        title: "系统设置",
        icon: "ep/setting",
        roles: ["super_admin", "admin"],
        permissions: ["admin.system_settings.read"]
      }
    }
  ]
} satisfies RouteConfigsTable;
