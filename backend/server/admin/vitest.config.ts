import { defineConfig } from "vitest/config";
import { fileURLToPath, URL } from "node:url";

export default defineConfig({
  resolve: {
    alias: {
      "@": fileURLToPath(new URL("./src", import.meta.url))
    }
  },
  test: {
    // forks + singleFork：避免关闭时遗留进程句柄，保证 CI 可干净退出
    pool: "forks",
    poolOptions: {
      forks: {
        singleFork: true
      }
    },
    environment: "node"
  }
});
