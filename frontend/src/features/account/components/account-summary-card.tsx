import { WorkspaceBadge, WorkspaceMetric, WorkspacePanel } from "@/components/layout/workspace-ui";
import { Badge } from "@/components/ui/badge";
import { Globe, Mail, ShieldCheck, UserRound } from "lucide-react";
import { useTranslation } from "react-i18next";
import type { AccountProfile } from "../api";

export function AccountSummaryCard({
  profile,
  twoFactorEnabled,
}: {
  profile: AccountProfile;
  twoFactorEnabled: boolean;
}) {
  const { t } = useTranslation();

  return (
    <WorkspacePanel
      description={t("account.summaryDescription")}
      title={t("account.summaryTitle")}
    >
      <div className="grid gap-3 lg:grid-cols-4">
        <WorkspaceMetric
          badge={<WorkspaceBadge variant="outline">{t("account.username")}</WorkspaceBadge>}
          hint={t("account.summaryUsernameHint")}
          icon={UserRound}
          label={t("account.summaryUsername")}
          value={profile.username}
        />
        <WorkspaceMetric
          badge={
            <Badge
              className="rounded-full px-2.5 py-0.5 text-[0.78rem]"
              variant={profile.emailVerified ? "secondary" : "outline"}
            >
              {profile.emailVerified
                ? t("account.emailVerified")
                : t("account.emailPending")}
            </Badge>
          }
          hint={profile.email}
          icon={Mail}
          label={t("account.summaryEmail")}
          value={profile.displayName || profile.email}
        />
        <WorkspaceMetric
          badge={<WorkspaceBadge variant="outline">{profile.roles.join(" / ")}</WorkspaceBadge>}
          hint={t("account.summaryLocaleHint", { locale: profile.locale })}
          icon={Globe}
          label={t("account.summaryLocale")}
          value={profile.timezone}
        />
        <WorkspaceMetric
          badge={
            <Badge
              className="rounded-full px-2.5 py-0.5 text-[0.78rem]"
              variant={twoFactorEnabled ? "secondary" : "outline"}
            >
              {twoFactorEnabled ? t("account.enabled") : t("account.disabled")}
            </Badge>
          }
          hint={t("account.summaryTwoFactorHint")}
          icon={ShieldCheck}
          label={t("account.summaryTwoFactor")}
          value={twoFactorEnabled ? t("account.totpProtected") : t("account.totpInactive")}
        />
      </div>
    </WorkspacePanel>
  );
}
