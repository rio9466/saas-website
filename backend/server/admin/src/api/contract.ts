/**
 * 手写的 API 契约适配层（禁止直接修改 types/api.generated.ts）。
 * 从这里引用生成类型，并补充通用响应信封类型。
 */
import type { components } from "../../types/api.generated";

/** 后端统一响应信封 */
export interface Envelope<T = unknown> {
  code: number;
  message: string;
  data: T;
  request_id: string;
}

export type AdministratorProfile =
  components["schemas"]["AdministratorProfile"];
export type Administrator = components["schemas"]["Administrator"];
export type Role = components["schemas"]["Role"];
export type Permission = components["schemas"]["Permission"];
export type AuditEvent = components["schemas"]["AuditEvent"];
export type AuditOutcome = components["schemas"]["AuditEvent"]["outcome"];

export type LoginData = NonNullable<
  components["schemas"]["LoginSuccess"]["data"]
>;
export type AdministratorPageData = NonNullable<
  components["schemas"]["AdministratorPageSuccess"]["data"]
>;
export type RoleListData = NonNullable<
  components["schemas"]["RoleListSuccess"]["data"]
>;
export type PermissionListData = NonNullable<
  components["schemas"]["PermissionListSuccess"]["data"]
>;
export type AuditPageData = NonNullable<
  components["schemas"]["AuditPageSuccess"]["data"]
>;

// ---- 业务用户平台类型 ----
export type UserProfile = components["schemas"]["UserProfile"];
export type UserLoginData = NonNullable<
  components["schemas"]["UserLoginSuccess"]["data"]
>;
export type BusinessUser = components["schemas"]["BusinessUser"];
export type BusinessUserPageData = NonNullable<
  components["schemas"]["BusinessUserPageSuccess"]["data"]
>;
export type PointTransaction = components["schemas"]["PointTransaction"];
export type PointTransactionPageData = NonNullable<
  components["schemas"]["PointTransactionPageSuccess"]["data"]
>;
export type UserLevel = components["schemas"]["UserLevel"];
export type UserLevelListData = NonNullable<
  components["schemas"]["UserLevelListSuccess"]["data"]
>;
export type SystemSettings = components["schemas"]["SystemSettings"];
export type PublicSettings = components["schemas"]["PublicSettings"];
