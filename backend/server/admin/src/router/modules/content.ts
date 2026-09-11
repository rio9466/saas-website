const Layout = () => import("@/layout/index.vue");

/**
 * 「内容管理」父级菜单（本地定义，不从后端拉菜单树）。
 * 子项分别以 admin.content.read / admin.contact.read 权限显隐；
 * 父级不设权限，由子项裁剪决定其是否显示。
 * 图标仅使用 offlineIcon.ts 已注册的离线图标（本任务不修改图标注册表）。
 */
export default {
  path: "/content",
  name: "ContentManagement",
  component: Layout,
  redirect: "/content/navigation",
  meta: {
    icon: "ri/file-list-3-line",
    title: "内容管理",
    rank: 3
  },
  children: [
    {
      path: "/content/navigation",
      name: "ContentNavigation",
      component: () => import("@/views/content/navigation/index.vue"),
      meta: {
        title: "导航管理",
        icon: "ri/search-line",
        roles: ["super_admin", "admin"],
        permissions: ["admin.content.read"]
      }
    },
    {
      path: "/content/home",
      name: "ContentHome",
      component: () => import("@/views/content/home/index.vue"),
      meta: {
        title: "首页区块",
        icon: "ep/home-filled",
        roles: ["super_admin", "admin"],
        permissions: ["admin.content.read"]
      }
    },
    {
      path: "/content/features",
      name: "ContentFeatures",
      component: () => import("@/views/content/features/index.vue"),
      meta: {
        title: "功能条目",
        icon: "ep/medal",
        roles: ["super_admin", "admin"],
        permissions: ["admin.content.read"]
      }
    },
    {
      path: "/content/pricing",
      name: "ContentPricing",
      component: () => import("@/views/content/pricing/index.vue"),
      meta: {
        title: "价格方案",
        icon: "ep/lock",
        roles: ["super_admin", "admin"],
        permissions: ["admin.content.read"]
      }
    },
    {
      path: "/content/pages",
      name: "ContentPages",
      component: () => import("@/views/content/pages/index.vue"),
      meta: {
        title: "页面",
        icon: "ri/file-list-3-line",
        roles: ["super_admin", "admin"],
        permissions: ["admin.content.read"]
      }
    },
    {
      path: "/content/docs",
      name: "ContentDocs",
      component: () => import("@/views/content/docs/index.vue"),
      meta: {
        title: "文档中心",
        icon: "ri/information-line",
        roles: ["super_admin", "admin"],
        permissions: ["admin.content.read"]
      }
    },
    {
      path: "/content/media",
      name: "ContentMedia",
      component: () => import("@/views/content/media/index.vue"),
      meta: {
        title: "媒体库",
        icon: "ep/avatar",
        roles: ["super_admin", "admin"],
        permissions: ["admin.content.read"]
      }
    },
    {
      path: "/content/contact",
      name: "ContactInbox",
      component: () => import("@/views/contact/index.vue"),
      meta: {
        title: "联系表单",
        icon: "ep/user-filled",
        roles: ["super_admin", "admin"],
        permissions: ["admin.contact.read"]
      }
    }
  ]
} satisfies RouteConfigsTable;
