import { createContext, type PropsWithChildren, useContext, useEffect, useLayoutEffect, useMemo, useRef, useState } from "react";
import {
  THEME_STORAGE_KEY,
  readStoredThemePreference,
  type ResolvedTheme,
  type ThemeMode,
} from "@/lib/preferences";

type ThemeContextValue = {
  theme: ThemeMode;
  resolvedTheme: ResolvedTheme;
  setTheme: (theme: ThemeMode) => void;
};

const ThemeContext = createContext<ThemeContextValue | null>(null);

function getSystemThemeFromMediaQuery(mediaQuery?: MediaQueryList) {
  return (mediaQuery ?? window.matchMedia("(prefers-color-scheme: dark)")).matches ? "dark" : "light";
}

function readBootstrappedThemeState() {
  const theme = readStoredThemePreference();

  if (typeof document === "undefined") {
    return {
      theme,
      systemTheme: "dark" as ResolvedTheme,
    };
  }

  const root = document.documentElement;
  const bootstrappedTheme: ResolvedTheme =
    root.dataset.theme === "dark" || root.classList.contains("dark") ? "dark" : "light";

  return {
    theme,
    systemTheme: theme === "system" ? bootstrappedTheme : getSystemThemeFromMediaQuery(),
  };
}

function applyTheme(theme: ThemeMode, systemTheme: ResolvedTheme) {
  const resolvedTheme = theme === "system" ? systemTheme : theme;
  const root = document.documentElement;

  root.classList.toggle("dark", resolvedTheme === "dark");
  root.style.colorScheme = resolvedTheme;
  root.dataset.theme = resolvedTheme;
  root.dataset.themePreference = theme;
  window.localStorage.setItem(THEME_STORAGE_KEY, theme);

  return resolvedTheme;
}

export function ThemeProvider({ children }: PropsWithChildren) {
  const [theme, setThemeState] = useState<ThemeMode>(() => readBootstrappedThemeState().theme);
  const [systemTheme, setSystemTheme] = useState<ResolvedTheme>(() => readBootstrappedThemeState().systemTheme);
  const initializedSystemThemeRef = useRef(theme !== "system");
  const resolvedTheme = theme === "system" ? systemTheme : theme;

  useEffect(() => {
    const mediaQuery = window.matchMedia("(prefers-color-scheme: dark)");
    const handleChange = () => {
      if (theme !== "system") {
        return;
      }

      setSystemTheme(getSystemThemeFromMediaQuery(mediaQuery));
    };

    if (theme === "system") {
      if (initializedSystemThemeRef.current) {
        handleChange();
      } else {
        initializedSystemThemeRef.current = true;
      }
    }

    if (typeof mediaQuery.addEventListener === "function") {
      mediaQuery.addEventListener("change", handleChange);
      return () => mediaQuery.removeEventListener("change", handleChange);
    }

    mediaQuery.addListener(handleChange);
    return () => mediaQuery.removeListener(handleChange);
  }, [theme]);

  useLayoutEffect(() => {
    applyTheme(theme, systemTheme);
  }, [systemTheme, theme]);

  const value = useMemo<ThemeContextValue>(
    () => ({
      theme,
      resolvedTheme,
      setTheme: setThemeState,
    }),
    [resolvedTheme, theme],
  );

  return <ThemeContext.Provider value={value}>{children}</ThemeContext.Provider>;
}

export function useTheme() {
  const context = useContext(ThemeContext);

  if (!context) {
    throw new Error("useTheme must be used within ThemeProvider.");
  }

  return context;
}
