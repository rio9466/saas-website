/** 内容管理共用的翻译字段描述与映射类型。 */

export type TranslationFieldType =
  | "input"
  | "textarea"
  | "markdown"
  | "string-list"
  | "json"
  | "kv-list"
  | "media";

export interface TranslationFieldDef {
  key: string;
  label: string;
  type?: TranslationFieldType;
  rows?: number;
  placeholder?: string;
  /** 字段下方的灰色说明 */
  tip?: string;
}

/** locale -> 该语言的字段映射 */
export type TranslationMap = Record<string, Record<string, unknown>>;
