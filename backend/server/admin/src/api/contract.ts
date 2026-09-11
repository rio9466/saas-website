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
export type SocialLink = components["schemas"]["SocialLink"];
export type LocaleOption = components["schemas"]["LocaleOption"];

// ---- 内容管理（契约 §6） ----
export type SiteSettingsTranslation =
  components["schemas"]["SiteSettingsTranslation"];
export type SiteSettings = components["schemas"]["SiteSettings"];
export type SiteSettingsWrite = components["schemas"]["SiteSettingsWrite"];

export type NavigationItem = components["schemas"]["NavigationItem"];
export type NavigationItemWrite = components["schemas"]["NavigationItemWrite"];
export type NavigationItemPageData = NonNullable<
  components["schemas"]["NavigationItemPageSuccess"]["data"]
>;

export type HomeSection = components["schemas"]["HomeSection"];
export type HomeSectionWrite = components["schemas"]["HomeSectionWrite"];
export type HomeSectionPageData = NonNullable<
  components["schemas"]["HomeSectionPageSuccess"]["data"]
>;

export type Feature = components["schemas"]["Feature"];
export type FeatureTranslation = components["schemas"]["FeatureTranslation"];
export type FeatureWrite = components["schemas"]["FeatureWrite"];
export type FeaturePageData = NonNullable<
  components["schemas"]["FeaturePageSuccess"]["data"]
>;

export type PricingPlan = components["schemas"]["PricingPlan"];
export type PricingPlanTranslation =
  components["schemas"]["PricingPlanTranslation"];
export type PricingPlanWrite = components["schemas"]["PricingPlanWrite"];
export type PricingPlanPageData = NonNullable<
  components["schemas"]["PricingPlanPageSuccess"]["data"]
>;

export type ContentPage = components["schemas"]["Page"];
export type PageTranslation = components["schemas"]["PageTranslation"];
export type PageWrite = components["schemas"]["PageWrite"];
export type ContentPageListData = NonNullable<
  components["schemas"]["PageListSuccess"]["data"]
>;

export type DocCategory = components["schemas"]["DocCategory"];
export type DocCategoryWrite = components["schemas"]["DocCategoryWrite"];
export type DocCategoryPageData = NonNullable<
  components["schemas"]["DocCategoryPageSuccess"]["data"]
>;

export type DocArticle = components["schemas"]["DocArticle"];
export type DocArticleTranslation =
  components["schemas"]["DocArticleTranslation"];
export type DocArticleWrite = components["schemas"]["DocArticleWrite"];
export type DocArticlePageData = NonNullable<
  components["schemas"]["DocArticlePageSuccess"]["data"]
>;

export type MediaAsset = components["schemas"]["MediaAsset"];
export type MediaAssetPageData = NonNullable<
  components["schemas"]["MediaAssetPageSuccess"]["data"]
>;

export type ContactSubmission = components["schemas"]["ContactSubmission"];
export type ContactSubmissionPageData = NonNullable<
  components["schemas"]["ContactSubmissionPageSuccess"]["data"]
>;
export type ContactStatus =
  components["schemas"]["ContactStatusWrite"]["status"];
