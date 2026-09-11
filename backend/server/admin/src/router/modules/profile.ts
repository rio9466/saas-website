const Layout = () => import("@/layout/index.vue");

export default {
  path: "/profile",
  name: "Profile",
  component: Layout,
  redirect: "/profile/index",
  meta: {
    icon: "ep/user",
    title: "个人中心",
    showLink: false,
    rank: 8
  },
  children: [
    {
      path: "/profile/index",
      name: "ProfileIndex",
      component: () => import("@/views/profile/index.vue"),
      meta: {
        title: "个人中心"
        // 所有已登录角色可见，通过用户菜单进入（showLink: false）
      }
    }
  ]
} satisfies RouteConfigsTable;
