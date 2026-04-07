const LEGACY_SMTP_HOSTNAME = "mail.shiro.local";
const LEGACY_DKIM_TARGET = "shiro._domainkey.shiro.local";

const COMMON_SECOND_LEVEL_SUFFIXES = new Set([
  "ac",
  "co",
  "com",
  "edu",
  "gov",
  "net",
  "org",
]);

function deriveRootDomainFromAppBaseURL(appBaseUrl: string) {
  try {
    const parsed = new URL(appBaseUrl.trim());
    const hostname = parsed.hostname.trim().toLowerCase();
    if (!hostname || hostname === "localhost") {
      return null;
    }
    if (/^\d{1,3}(\.\d{1,3}){3}$/.test(hostname) || hostname.includes(":")) {
      return null;
    }

    const labels = hostname.split(".");
    if (labels.length < 2) {
      return null;
    }
    if (labels.length === 2) {
      return hostname;
    }

    const last = labels.at(-1) ?? "";
    const secondLast = labels.at(-2) ?? "";
    if (last.length === 2 && COMMON_SECOND_LEVEL_SUFFIXES.has(secondLast) && labels.length >= 3) {
      return labels.slice(-3).join(".");
    }

    return labels.slice(-2).join(".");
  } catch {
    return null;
  }
}

export function deriveMailTargetsFromAppBaseURL(appBaseUrl: string) {
  const rootDomain = deriveRootDomainFromAppBaseURL(appBaseUrl);
  if (!rootDomain) {
    return null;
  }

  return {
    hostname: `smtp.${rootDomain}`,
    dkimCnameTarget: `shiro._domainkey.${rootDomain}`,
  };
}

export function shouldReplaceWithDerivedMailTarget(currentValue: string, previousDerivedValue?: string | null) {
  const normalized = currentValue.trim().toLowerCase();
  return (
    normalized === "" ||
    normalized === LEGACY_SMTP_HOSTNAME ||
    normalized === LEGACY_DKIM_TARGET ||
    (previousDerivedValue != null && normalized === previousDerivedValue.trim().toLowerCase())
  );
}
