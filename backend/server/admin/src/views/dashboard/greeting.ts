/**
 * 工作台问候语（参考 pure-admin /welcome 的时间问候）。
 * 纯函数、无副作用，便于单元测试。
 */

export function greetingForHour(hour: number): string {
  const h = ((hour % 24) + 24) % 24;
  if (h < 6) return "凌晨好";
  if (h < 9) return "早上好";
  if (h < 12) return "上午好";
  if (h < 14) return "中午好";
  if (h < 18) return "下午好";
  return "晚上好";
}

export function greetingNow(now: Date = new Date()): string {
  return greetingForHour(now.getHours());
}

/** 今天是 YYYY年M月D日 dddd → 用中文星期，不引入额外依赖 */
export function todayText(now: Date = new Date()): string {
  const weekdays = [
    "星期日",
    "星期一",
    "星期二",
    "星期三",
    "星期四",
    "星期五",
    "星期六"
  ];
  return `${now.getFullYear()}年${now.getMonth() + 1}月${now.getDate()}日 ${
    weekdays[now.getDay()]
  }`;
}
