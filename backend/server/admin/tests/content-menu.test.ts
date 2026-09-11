import { describe, expect, it, vi } from "vitest";
import contentMenu from "@/router/modules/content";

/**
 * 内容管理菜单为本地定义；图标必须已在 offlineIcon.ts 注册，否则侧边栏静默不显示。
 * 本测试同时锁定权限码，避免菜单显隐与 RBAC 脱节。
 */
const { registeredIcons } = vi.hoisted(() => ({
  registeredIcons: new Set<string>()
}));

vi.mock("@iconify/vue/dist/offline", () => ({
  addIcon: (name: string) => {
    registeredIcons.add(name);
  }
}));
vi.mock("@pureadmin/utils", () => ({
  getSvgInfo: (raw: string) => ({ body: raw })
}));
vi.mock("~icons/ep/home-filled?raw", () => ({ default: "<svg/>" }));
vi.mock("~icons/ep/setting?raw", () => ({ default: "<svg/>" }));
vi.mock("~icons/ep/user?raw", () => ({ default: "<svg/>" }));
vi.mock("~icons/ep/lock?raw", () => ({ default: "<svg/>" }));
vi.mock("~icons/ep/user-filled?raw", () => ({ default: "<svg/>" }));
vi.mock("~icons/ep/medal?raw", () => ({ default: "<svg/>" }));
vi.mock("~icons/ep/avatar?raw", () => ({ default: "<svg/>" }));
vi.mock("~icons/ri/search-line?raw", () => ({ default: "<svg/>" }));
vi.mock("~icons/ri/information-line?raw", () => ({ default: "<svg/>" }));
vi.mock("~icons/ri/file-list-3-line?raw", () => ({ default: "<svg/>" }));

function findChild(name: string) {
  return contentMenu.children?.find(child => child.name === name);
}

describe("content management menu", () => {
  it("mounts a 内容管理 parent containing every content resource", () => {
    expect(contentMenu.path).toBe("/content");
    expect(contentMenu.name).toBe("ContentManagement");
    expect(contentMenu.meta?.title).toBe("内容管理");
    expect(contentMenu.children?.map(child => child.name)).toEqual([
      "ContentNavigation",
      "ContentHome",
      "ContentFeatures",
      "ContentPricing",
      "ContentPages",
      "ContentDocs",
      "ContentMedia",
      "ContactInbox"
    ]);
  });

  it("gates content resources with admin.content.read", () => {
    for (const name of [
      "ContentNavigation",
      "ContentHome",
      "ContentFeatures",
      "ContentPricing",
      "ContentPages",
      "ContentDocs",
      "ContentMedia"
    ]) {
      expect(findChild(name)?.meta?.permissions).toEqual([
        "admin.content.read"
      ]);
    }
  });

  it("gates the contact inbox with admin.contact.read", () => {
    expect(findChild("ContactInbox")?.meta?.permissions).toEqual([
      "admin.contact.read"
    ]);
    expect(findChild("ContactInbox")?.path).toBe("/content/contact");
  });

  it("registers every content-menu icon in the offline icon registry", async () => {
    await import("@/components/ReIcon/src/offlineIcon");

    const menuIcons = [
      contentMenu.meta?.icon,
      ...(contentMenu.children ?? []).map(child => child.meta?.icon)
    ].filter((icon): icon is string => typeof icon === "string");

    expect(menuIcons.length).toBeGreaterThan(0);
    for (const icon of menuIcons) {
      expect(registeredIcons.has(icon), `未注册的菜单图标: ${icon}`).toBe(true);
    }
  });
});
