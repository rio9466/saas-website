export { default as dayjs } from "dayjs";
export { useDark, cloneDeep, randomGradient } from "@pureadmin/utils";

/**
 * 官方 welcome 页使用 `Math.random` 生成表格演示数据。
 * easy-admin 要求静态演示数据稳定（刷新、重复进入不变化），
 * 因此这里使用固定种子的 LCG 伪随机序列替代真随机，
 * 保证在相同调用顺序下每次生成的数值完全一致。
 */
let seed = 20260908;

function seededRandom(): number {
  seed = (seed * 1664525 + 1013904223) % 4294967296;
  return seed / 4294967296;
}

export function getRandomIntBetween(min: number, max: number) {
  return Math.floor(seededRandom() * (max - min + 1)) + min;
}
