/**
 * 文档分类与文档文章管理 API（对应 openapi.yaml 的 admin-content 分组）。
 * 分类提供名称翻译；文章顶层 `category_id` 决定归属。
 */
import { http } from "@/utils/http";
import type {
  DocArticle,
  DocArticlePageData,
  DocArticleWrite,
  DocCategory,
  DocCategoryPageData,
  DocCategoryWrite,
  Envelope
} from "./contract";

export type ListDocParams = {
  page?: number;
  page_size?: number;
};

export type DocCategoryPayload = DocCategoryWrite;
export type DocArticlePayload = DocArticleWrite;

// ---- 文档分类 ----
export const listDocCategoriesApi = (params: ListDocParams = {}) => {
  return http.request<Envelope<DocCategoryPageData>>("get", "/doc-categories", {
    params
  });
};

export const createDocCategoryApi = (data: DocCategoryPayload) => {
  return http.request<Envelope<DocCategory>>("post", "/doc-categories", {
    data
  });
};

export const updateDocCategoryApi = (id: string, data: DocCategoryPayload) => {
  return http.request<Envelope<DocCategory>>("patch", `/doc-categories/${id}`, {
    data
  });
};

export const deleteDocCategoryApi = (id: string) => {
  return http.request<Envelope<Record<string, never>>>(
    "delete",
    `/doc-categories/${id}`
  );
};

// ---- 文档文章 ----
export const listDocArticlesApi = (params: ListDocParams = {}) => {
  return http.request<Envelope<DocArticlePageData>>("get", "/doc-articles", {
    params
  });
};

export const getDocArticleApi = (id: string) => {
  return http.request<Envelope<DocArticle>>("get", `/doc-articles/${id}`);
};

export const createDocArticleApi = (data: DocArticlePayload) => {
  return http.request<Envelope<DocArticle>>("post", "/doc-articles", { data });
};

export const updateDocArticleApi = (id: string, data: DocArticlePayload) => {
  return http.request<Envelope<DocArticle>>("patch", `/doc-articles/${id}`, {
    data
  });
};

export const deleteDocArticleApi = (id: string) => {
  return http.request<Envelope<Record<string, never>>>(
    "delete",
    `/doc-articles/${id}`
  );
};
