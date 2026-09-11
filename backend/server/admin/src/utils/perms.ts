/**
 * 按钮级权限判断（从 /me 返回的 permission_codes 判断）。
 */
import { isString, isIncludeAllChildren } from "@pureadmin/utils";
import { useUserStoreHook } from "@/store/modules/user";

/** 是否有按钮级别的权限 */
export const hasPerms = (value: string | Array<string>): boolean => {
  if (!value) return false;
  const allPerms = "*:*:*";
  const { permissions } = useUserStoreHook();
  if (!permissions) return false;
  if (permissions.length === 1 && permissions[0] === allPerms) return true;
  const isAuths = isString(value)
    ? permissions.includes(value)
    : isIncludeAllChildren(value, permissions);
  return isAuths ? true : false;
};
