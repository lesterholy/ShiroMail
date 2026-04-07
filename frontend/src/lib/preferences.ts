export const THEME_STORAGE_KEY = "shiro-email.theme";
export const LANGUAGE_STORAGE_KEY = "shiro-email.language";

export const themeModes = ["light", "dark", "system"] as const;
export type ThemeMode = (typeof themeModes)[number];
export type ResolvedTheme = Exclude<ThemeMode, "system">;

export const supportedLanguages = ["zh-CN", "en-US"] as const;
export type SupportedLanguage = (typeof supportedLanguages)[number];

export function normalizeLanguage(value?: string | null): SupportedLanguage {
  if (typeof value === "string" && value.toLowerCase().startsWith("en")) {
    return "en-US";
  }

  return "zh-CN";
}

export function readStoredLanguage(): SupportedLanguage {
  if (typeof window === "undefined") {
    return "zh-CN";
  }

  const raw = window.localStorage.getItem(LANGUAGE_STORAGE_KEY);
  if (raw === "zh-CN" || raw === "en-US") {
    return raw;
  }

  return normalizeLanguage(window.navigator.language);
}

export function readStoredThemePreference(): ThemeMode {
  if (typeof window === "undefined") {
    return "system";
  }

  const raw = window.localStorage.getItem(THEME_STORAGE_KEY);
  if (raw === "light" || raw === "dark" || raw === "system") {
    return raw;
  }

  return "system";
}

export function getSystemTheme(): ResolvedTheme {
  if (typeof window === "undefined") {
    return "dark";
  }

  return window.matchMedia("(prefers-color-scheme: dark)").matches ? "dark" : "light";
}

export function resolveThemePreference(theme: ThemeMode, systemTheme = getSystemTheme()): ResolvedTheme {
  return theme === "system" ? systemTheme : theme;
}
