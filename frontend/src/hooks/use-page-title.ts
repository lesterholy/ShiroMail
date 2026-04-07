import { useEffect } from "react";

export function composePageTitle(title: string | null | undefined, siteName: string) {
  const trimmedTitle = title?.trim() ?? "";
  const trimmedSiteName = siteName.trim();
  if (!trimmedTitle) {
    return trimmedSiteName;
  }
  if (!trimmedSiteName || trimmedTitle.includes(trimmedSiteName)) {
    return trimmedTitle;
  }
  return `${trimmedTitle} · ${trimmedSiteName}`;
}

export function usePageTitle(title: string | null | undefined) {
  useEffect(() => {
    if (!title?.trim()) {
      return undefined;
    }
    const previousTitle = document.title;
    document.title = title;
    return () => {
      document.title = previousTitle;
    };
  }, [title]);
}
