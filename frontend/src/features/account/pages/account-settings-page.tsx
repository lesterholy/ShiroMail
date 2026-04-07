import { useEffect, useMemo, useState } from "react";
import { RefreshCw } from "lucide-react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { useTranslation } from "react-i18next";
import { useSearchParams } from "react-router-dom";
import { WorkspaceEmpty, WorkspacePage } from "@/components/layout/workspace-ui";
import { Button } from "@/components/ui/button";
import { Skeleton } from "@/components/ui/skeleton";
import { useAuthStore } from "@/lib/auth-store";
import {
  changeAccountPassword,
  confirmAccountEmailChange,
  disableTOTP,
  enableTOTP,
  fetchAccountProfile,
  fetchTOTPStatus,
  getAccountErrorMessage,
  requestAccountEmailChange,
  setupTOTP,
  updateAccountProfile,
  type TOTPSetup,
  type VerificationChallenge,
} from "../api";
import { AccountEmailCard } from "../components/account-email-card";
import { AccountPasswordCard } from "../components/account-password-card";
import { AccountProfileCard } from "../components/account-profile-card";
import { AccountSummaryCard } from "../components/account-summary-card";
import { AccountTOTPCard } from "../components/account-totp-card";

export function AccountSettingsPage({ consoleKind }: { consoleKind: "user" | "admin" }) {
  const { t } = useTranslation();
  const queryClient = useQueryClient();
  const [searchParams] = useSearchParams();
  const authUserId = useAuthStore((state) => state.user?.userId ?? null);
  const [totpSetup, setTotpSetup] = useState<TOTPSetup | null>(null);
  void consoleKind;

  const profileQuery = useQuery({
    queryKey: ["account-profile", authUserId],
    queryFn: fetchAccountProfile,
    staleTime: 60_000,
    enabled: authUserId !== null,
  });
  const totpStatusQuery = useQuery({
    queryKey: ["account-2fa-status", authUserId],
    queryFn: fetchTOTPStatus,
    staleTime: 30_000,
    enabled: authUserId !== null,
  });

  useEffect(() => {
    if (totpStatusQuery.data?.enabled) {
      setTotpSetup(null);
    }
  }, [totpStatusQuery.data?.enabled]);

  useEffect(() => {
    setTotpSetup(null);
  }, [authUserId]);

  const profileMutation = useMutation({
    mutationFn: updateAccountProfile,
    onSuccess: async () => {
      await queryClient.invalidateQueries({ queryKey: ["account-profile"] });
      await queryClient.invalidateQueries({ queryKey: ["portal-settings"] });
      await queryClient.invalidateQueries({ queryKey: ["portal-overview"] });
    },
  });
  const requestEmailMutation = useMutation({
    mutationFn: requestAccountEmailChange,
  });
  const confirmEmailMutation = useMutation({
    mutationFn: confirmAccountEmailChange,
    onSuccess: async () => {
      await queryClient.invalidateQueries({ queryKey: ["account-profile"] });
      await queryClient.invalidateQueries({ queryKey: ["portal-settings"] });
      await queryClient.invalidateQueries({ queryKey: ["portal-overview"] });
    },
  });
  const passwordMutation = useMutation({
    mutationFn: changeAccountPassword,
  });
  const setupTOTPMutation = useMutation({
    mutationFn: setupTOTP,
    onSuccess: (data) => {
      setTotpSetup(data);
    },
  });
  const enableTOTPMutation = useMutation({
    mutationFn: enableTOTP,
    onSuccess: async () => {
      setTotpSetup(null);
      await queryClient.invalidateQueries({ queryKey: ["account-profile"] });
      await queryClient.invalidateQueries({ queryKey: ["account-2fa-status"] });
    },
  });
  const disableTOTPMutation = useMutation({
    mutationFn: disableTOTP,
    onSuccess: async () => {
      setTotpSetup(null);
      await queryClient.invalidateQueries({ queryKey: ["account-profile"] });
      await queryClient.invalidateQueries({ queryKey: ["account-2fa-status"] });
    },
  });

  const twoFactorEnabled = useMemo(
    () => totpStatusQuery.data?.enabled ?? profileQuery.data?.twoFactorEnabled ?? false,
    [profileQuery.data?.twoFactorEnabled, totpStatusQuery.data?.enabled],
  );
  const emailChangePrefill = useMemo(() => {
    if (searchParams.get("action") !== "change-email") {
      return null;
    }
    const verificationTicket = searchParams.get("emailChangeTicket") ?? "";
    const email = searchParams.get("emailChangeEmail") ?? "";
    if (!verificationTicket || !email) {
      return null;
    }
    return {
      challenge: {
        status: "verification_required" as const,
        email,
        verificationTicket,
        expiresInSeconds: 0,
      },
      code: searchParams.get("emailChangeCode") ?? "",
    };
  }, [searchParams]);

  if (profileQuery.isLoading) {
    return (
      <WorkspacePage>
        <div className="grid gap-4">
          <Skeleton className="h-36 rounded-xl" />
          <div className="grid gap-4 xl:grid-cols-2">
            <Skeleton className="h-72 rounded-xl" />
            <Skeleton className="h-72 rounded-xl" />
            <Skeleton className="h-60 rounded-xl" />
            <Skeleton className="h-60 rounded-xl" />
          </div>
        </div>
      </WorkspacePage>
    );
  }

  if (!profileQuery.data) {
    return (
      <WorkspacePage>
        <WorkspaceEmpty
          action={
            <Button size="sm" type="button" variant="outline" onClick={() => void profileQuery.refetch()}>
              <RefreshCw className={profileQuery.isRefetching ? "size-4 animate-spin" : "size-4"} />
              {t("common.refresh")}
            </Button>
          }
          description={t("account.loadFailed")}
          title={t("account.title")}
        />
      </WorkspacePage>
    );
  }

  const profile = profileQuery.data;

  return (
    <WorkspacePage>
      <AccountSummaryCard profile={profile} twoFactorEnabled={twoFactorEnabled} />
      <div className="grid gap-4 xl:grid-cols-2">
        <AccountProfileCard
          isSaving={profileMutation.isPending}
          profile={profile}
          onSave={async (input) => {
            try {
              await profileMutation.mutateAsync(input);
            } catch (error) {
              throw new Error(getAccountErrorMessage(error, t("account.profileSaveFailed")));
            }
          }}
        />
        <AccountEmailCard
          isConfirmPending={confirmEmailMutation.isPending}
          initialChallenge={emailChangePrefill?.challenge}
          initialCode={emailChangePrefill?.code}
          isRequestPending={requestEmailMutation.isPending}
          profile={profile}
          onConfirmChange={async (input) => {
            try {
              await confirmEmailMutation.mutateAsync(input);
            } catch (error) {
              throw new Error(getAccountErrorMessage(error, t("account.emailChangeConfirmFailed")));
            }
          }}
          onRequestChange={async (email): Promise<VerificationChallenge> => {
            try {
              return await requestEmailMutation.mutateAsync(email);
            } catch (error) {
              throw new Error(getAccountErrorMessage(error, t("account.emailChangeRequestFailed")));
            }
          }}
        />
        <AccountPasswordCard
          isPending={passwordMutation.isPending}
          onSubmit={async (input) => {
            try {
              await passwordMutation.mutateAsync(input);
            } catch (error) {
              throw new Error(getAccountErrorMessage(error, t("account.passwordUpdateFailed")));
            }
          }}
        />
        <AccountTOTPCard
          enabled={twoFactorEnabled}
          isDisablePending={disableTOTPMutation.isPending}
          isEnablePending={enableTOTPMutation.isPending}
          isSetupPending={setupTOTPMutation.isPending}
          setupDraft={totpSetup}
          onDisable={async (password) => {
            try {
              await disableTOTPMutation.mutateAsync(password);
            } catch (error) {
              throw new Error(getAccountErrorMessage(error, t("account.totpDisableFailed")));
            }
          }}
          onEnable={async (code) => {
            try {
              await enableTOTPMutation.mutateAsync(code);
            } catch (error) {
              throw new Error(getAccountErrorMessage(error, t("account.totpEnableFailed")));
            }
          }}
          onSetup={async () => {
            try {
              await setupTOTPMutation.mutateAsync();
            } catch (error) {
              throw new Error(getAccountErrorMessage(error, t("account.totpSetupFailed")));
            }
          }}
        />
      </div>
    </WorkspacePage>
  );
}
