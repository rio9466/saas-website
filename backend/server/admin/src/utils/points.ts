/**
 * 积分显示工具：后端始终以四位小数字符串传输（如 "123.4567"），前端只做
 * half-up 舍入到两位小数用于展示，绝不回写、绝不变更存储的原始字符串。
 * 手工字符串舍入避免浮点误差（禁止对积分做浮点运算）。
 */

/** 纯字符串十进制 +1（不使用 Number/parseInt 处理大整数，避免 2^53 精度丢失）。 */
function addOneToInteger(intDigits: string): string {
  const digits = intDigits.split("");
  let i = digits.length - 1;
  for (; i >= 0; i--) {
    if (digits[i] === "9") {
      digits[i] = "0";
    } else {
      digits[i] = String.fromCharCode(digits[i].charCodeAt(0) + 1);
      break;
    }
  }
  if (i < 0) digits.unshift("1");
  return digits.join("");
}

/** 四舍五入到两位小数（half-up），输入必须为数值字符串（可带负号）。 */
export function roundHalfUpTwo(input?: string): string {
  if (!input) return "0.00";
  const raw = String(input).trim();
  if (raw === "") return "0.00";
  if (!/^-?\d+(\.\d+)?$/.test(raw)) return raw;

  const negative = raw.startsWith("-");
  const abs = negative ? raw.slice(1) : raw;
  const [intPart = "0", fracRaw = ""] = abs.split(".");
  const frac4 = (fracRaw + "0000").slice(0, 4);

  // frac4 的第 3 位决定是否向两位小数进位（half-up）。
  let intDigits = intPart || "0";
  let two = frac4.slice(0, 2);
  if ((frac4[2] ?? "0") >= "5") {
    if (two === "99") {
      two = "00";
      intDigits = addOneToInteger(intDigits);
    } else if (two[1] === "9") {
      // 十位最多从 8 进到 9（99 已在上面处理）。
      two = String.fromCharCode(two.charCodeAt(0) + 1) + "0";
    } else {
      two = two[0] + String.fromCharCode(two.charCodeAt(1) + 1);
    }
  }
  return `${negative ? "-" : ""}${intDigits}.${two}`;
}

/** 补全为固定四位小数字符串（用于提交前规范化）。 */
export function toFourDecimal(input: string): string {
  const raw = (input ?? "").trim();
  if (raw === "") return "";
  if (!/^-?\d+(\.\d+)?$/.test(raw)) return raw;
  const negative = raw.startsWith("-");
  const abs = negative ? raw.slice(1) : raw;
  const [intPart = "0", fracRaw = ""] = abs.split(".");
  const frac = (fracRaw + "0000").slice(0, 4);
  return `${negative ? "-" : ""}${intPart}.${frac}`;
}
