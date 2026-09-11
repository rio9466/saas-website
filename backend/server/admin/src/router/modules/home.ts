const { VITE_HIDE_HOME } = import.meta.env;
const Layout = () => import("@/layout/index.vue");

export default {
  path: "/",
  name: "Home",
  component: Layout,
  redirect: "/dashboard",
  meta: {
    icon: "ep/home-filled",
    title: "工作台",
    rank: 0
  },
  children: [
    {
      path: "/dashboard",
      name: "Dashboard",
      component: () => import("@/views/dashboard/index.vue"),
      meta: {
        title: "工作台",
        // 所有已登录角色（super_admin / admin / finance）都可见
        permissions: ["dashboard.view"],
        showLink: VITE_HIDE_HOME === "true" ? false : true
      }
    }
  ]
} satisfies RouteConfigsTable;
