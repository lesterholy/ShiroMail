import { http } from "../../lib/http";

export type PublicSiteSettings = {
  identity: {
    siteName: string;
    slogan: string;
    supportEmail: string;
    appBaseUrl: string;
    defaultLanguage: string;
    defaultTimeZone: string;
  };
  mailDns: {
    mxTarget: string;
    dkimCnameTarget: string;
  };
};

export async function fetchPublicSiteSettings() {
  const { data } = await http.get<PublicSiteSettings>("/site/settings");
  return data;
}
