/**
 * 本地路由按 /me 角色/权限过滤的纯函数（无 Pinia / router 依赖，便于单测）。
 *
 * 路由 meta 支持 roles（角色代码）与 permissions（权限代码）两种声明：
 * - 未声明 roles/permissions → 所有已登录用户可见；
 * - 声明任一 → 与用户角色/权限有交集才可见（两种声明同时存在时取交集语义，
 *   即两类都必须命中）。
 *
 * 安全约束：用户角色/权限数组为空时，对已声明的路由要求**拒绝**（fail-closed），
 * 不允许把空权限误当成"不限制"。
 */
export interface RouteLikeMeta {
  roles?: string[];
  permissions?: string[];
  showLink?: boolean;
  title?: string;
}

export interface RouteLike {
  path?: string;
  name?: string;
  redirect?: string;
  meta?: RouteLikeMeta;
  children?: RouteLike[];
}

/** 判断两个数组彼此是否存在相同值（空数组不视为"不限制"） */
export function isOneOfArray(
  a: string[] | undefined,
  b: string[] | undefined
): boolean {
  if (!Array.isArray(a) || !Array.isArray(b)) return true;
  return a.some(item => b.includes(item));
}

/** 单个路由对当前角色/权限是否可见（fail-closed：已声明要求且用户数组为空则不可见） */
export function routeVisibleByAuth(
  route: RouteLike,
  roles: string[] | undefined,
  permissions: string[] | undefined
): boolean {
  const routeRoles = route.meta?.roles;
  const routePerms = route.meta?.permissions;
  // 路由声明了 roles / permissions 时，用户对应数组必须有交集；
  // 用户数组为空 → 无交集 → 不可见（不允许空权限放行受保护路由）
  const roleOk = routeRoles?.length ? isOneOfArray(routeRoles, roles) : true;
  const permOk = routePerms?.length
    ? isOneOfArray(routePerms, permissions)
    : true;
  return roleOk && permOk;
}

/** 递归过滤路由树；目录下没有任何可见子路由时目录也被过滤 */
export function filterRoutesByAuth(
  data: RouteLike[],
  roles: string[] | undefined,
  permissions: string[] | undefined
): RouteLike[] {
  const visible = (route: RouteLike): boolean =>
    routeVisibleByAuth(route, roles, permissions);

  const filterLevel = (level: RouteLike[]): RouteLike[] => {
    const result: RouteLike[] = [];
    level.forEach(route => {
      if (!visible(route)) return;
      const next = { ...route };
      if (next.children?.length) {
        next.children = filterLevel(next.children);
        if (!next.children.length) return;
      }
      result.push(next);
    });
    return result;
  };

  return filterLevel(data);
}
