import { render, screen, waitFor } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { ThemeProvider, useTheme } from "./theme-provider";

function ThemeProbe() {
  const { theme, resolvedTheme } = useTheme();
  return <div data-resolved-theme={resolvedTheme} data-theme={theme}>theme-probe</div>;
}

describe("ThemeProvider", () => {
  const originalMatchMedia = window.matchMedia;

  beforeEach(() => {
    window.localStorage.clear();
    document.documentElement.className = "";
    document.documentElement.dataset.theme = "";
    document.documentElement.dataset.themePreference = "";
    document.documentElement.style.colorScheme = "";
  });

  afterEach(() => {
    window.matchMedia = originalMatchMedia;
    vi.restoreAllMocks();
  });

  it("keeps the bootstrapped dark system theme from the document root on first mount", async () => {
    window.localStorage.setItem("shiro-email.theme", "system");
    document.documentElement.classList.add("dark");
    document.documentElement.dataset.theme = "dark";
    document.documentElement.dataset.themePreference = "system";
    document.documentElement.style.colorScheme = "dark";

    window.matchMedia = vi.fn().mockImplementation(() => ({
      matches: false,
      media: "(prefers-color-scheme: dark)",
      onchange: null,
      addEventListener: vi.fn(),
      removeEventListener: vi.fn(),
      addListener: vi.fn(),
      removeListener: vi.fn(),
      dispatchEvent: vi.fn(),
    })) as typeof window.matchMedia;

    render(
      <ThemeProvider>
        <ThemeProbe />
      </ThemeProvider>,
    );

    await waitFor(() => {
      expect(document.documentElement.classList.contains("dark")).toBe(true);
      expect(document.documentElement.dataset.theme).toBe("dark");
      expect(document.documentElement.dataset.themePreference).toBe("system");
      expect(document.documentElement.style.colorScheme).toBe("dark");
      expect(screen.getByText("theme-probe")).toHaveAttribute("data-resolved-theme", "dark");
    });
  });
});
