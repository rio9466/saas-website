/** 值是否为空（null/undefined、空串/空数组/空对象） */
export function isEmptyValue(value: unknown): boolean {
  if (value == null) return true;
  if (typeof value === "string") return value.trim() === "";
  if (Array.isArray(value)) return value.length === 0;
  if (typeof value === "object") return Object.keys(value).length === 0;
  return false;
}

/**
 * 只保留「填写的语言」：整条翻译所有字段都为空则丢弃。
 * 后端在写入时会整体替换翻译集合，因此未填写的语言不应提交。
 */
export function compactTranslations<T extends Record<string, unknown>>(
  translations: Record<string, T> | undefined
): Record<string, T> {
  const out: Record<string, T> = {};
  for (const [locale, value] of Object.entries(translations ?? {})) {
    if (!value) continue;
    if (Object.values(value).every(isEmptyValue)) continue;
    out[locale] = value;
  }
  return out;
}

/** 把一行文本拆成去空的字符串数组（价格功能清单等） */
export function listToText(list: string[] | undefined): string {
  return (list ?? []).join("\n");
}

export function textToList(text: string): string[] {
  return text
    .split("\n")
    .map(item => item.trim())
    .filter(Boolean);
}
