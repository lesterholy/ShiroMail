import { useQuery } from "@tanstack/react-query";
import { fetchPublicSiteSettings } from "@/features/home/api";

const DEFAULT_SITE_NAME = "Shiro Email";

export function useSiteName() {
  const siteSettingsQuery = useQuery({
    queryKey: ["public-site-settings"],
    queryFn: fetchPublicSiteSettings,
    staleTime: 60_000,
  });

  return siteSettingsQuery.data?.identity.siteName || DEFAULT_SITE_NAME;
}

export { DEFAULT_SITE_NAME };
