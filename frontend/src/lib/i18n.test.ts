import { afterEach, describe, expect, it, vi } from "vitest";

describe("i18n lazy loading", () => {
  afterEach(() => {
    vi.resetModules();
    window.localStorage.clear();
  });

  it("exposes locale bundles through default exports", async () => {
    const module = await import("../locales/en-US");

    expect(module.default.translation.common.refresh).toBe("Refresh");
  });

  it("loads only the active locale at startup and lazy-loads the next locale on demand", async () => {
    window.localStorage.setItem("shiro-email.language", "zh-CN");

    const { default: i18n, changeAppLanguage, initializeI18n } = await import("./i18n");

    await initializeI18n();

    expect(i18n.hasResourceBundle("zh-CN", "translation")).toBe(true);
    expect(i18n.hasResourceBundle("en-US", "translation")).toBe(false);

    await changeAppLanguage("en-US");

    expect(i18n.language).toBe("en-US");
    expect(i18n.t("common.refresh")).toBe("Refresh");
    expect(document.documentElement.lang).toBe("en-US");
    expect(window.localStorage.getItem("shiro-email.language")).toBe("en-US");

    await changeAppLanguage("zh-CN");

    expect(i18n.language).toBe("zh-CN");
    expect(i18n.t("common.refresh")).toBe("刷新数据");
    expect(document.documentElement.lang).toBe("zh-CN");
    expect(window.localStorage.getItem("shiro-email.language")).toBe("zh-CN");
  });
});
