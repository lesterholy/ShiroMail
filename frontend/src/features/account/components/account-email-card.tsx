import { useEffect, useState } from "react";
import { useTranslation } from "react-i18next";
import { WorkspaceField, WorkspacePanel } from "@/components/layout/workspace-ui";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { useAutoDismiss } from "@/hooks/use-auto-dismiss";
import { validateEmailAddress, validateOneTimeCode } from "@/lib/validation";
import type { AccountProfile, VerificationChallenge } from "../api";

export function AccountEmailCard({
  profile,
  isRequestPending,
  isConfirmPending,
  initialChallenge,
  initialCode,
  onRequestChange,
  onConfirmChange,
}: {
  profile: AccountProfile;
  isRequestPending: boolean;
  isConfirmPending: boolean;
  initialChallenge?: VerificationChallenge | null;
  initialCode?: string;
  onRequestChange: (email: string) => Promise<VerificationChallenge>;
  onConfirmChange: (input: { verificationTicket: string; code: string }) => Promise<void>;
}) {
  const { t } = useTranslation();
  const [nextEmail, setNextEmail] = useState("");
  const [code, setCode] = useState("");
  const [challenge, setChallenge] = useState<VerificationChallenge | null>(null);
  const [notice, setNotice] = useState<string | null>(null);
  const [error, setError] = useState<string | null>(null);

  useAutoDismiss(notice, () => setNotice(null));
  useAutoDismiss(error, () => setError(null));

  useEffect(() => {
    setNextEmail(profile.email);
  }, [profile.email]);

  useEffect(() => {
    if (!initialChallenge) {
      return;
    }
    setChallenge(initialChallenge);
    setNextEmail(initialChallenge.email);
    setCode(initialCode ?? "");
    setNotice(t("account.emailVerificationSent", { email: initialChallenge.email }));
  }, [initialChallenge, initialCode, t]);

  async function handleRequest() {
    setError(null);
    const emailError = validateEmailAddress(nextEmail);
    if (emailError) {
      setError(emailError);
      return;
    }
    if (nextEmail.trim().toLowerCase() === profile.email.trim().toLowerCase()) {
      setError(t("account.newEmail") + "不能与当前邮箱相同。");
      return;
    }
    try {
      const result = await onRequestChange(nextEmail.trim());
      setChallenge(result);
      setCode("");
      setNotice(t("account.emailVerificationSent", { email: result.email }));
    } catch (currentError) {
      setError(currentError instanceof Error ? currentError.message : t("account.emailChangeRequestFailed"));
    }
  }

  async function handleConfirm() {
    if (!challenge) {
      return;
    }
    setError(null);
    const codeError = validateOneTimeCode(code);
    if (codeError) {
      setError(codeError);
      return;
    }
    try {
      await onConfirmChange({
        verificationTicket: challenge.verificationTicket,
        code: code.trim(),
      });
      setChallenge(null);
      setCode("");
      setNotice(t("account.emailUpdated"));
    } catch (currentError) {
      setError(currentError instanceof Error ? currentError.message : t("account.emailChangeConfirmFailed"));
    }
  }

  return (
    <WorkspacePanel
      description={t("account.emailDescription")}
      title={t("account.emailTitle")}
    >
      <div className="grid gap-4 xl:grid-cols-2">
        <WorkspaceField label={t("account.currentEmail")}>
          <Input disabled value={profile.email} />
        </WorkspaceField>

        <WorkspaceField label={t("account.newEmail")}>
          <div className="flex gap-2">
            <Input
              onChange={(event) => setNextEmail(event.target.value)}
              placeholder={t("account.newEmailPlaceholder")}
              type="email"
              value={nextEmail}
            />
            <Button disabled={isRequestPending || nextEmail.trim().length === 0} size="sm" onClick={handleRequest}>
              {isRequestPending ? t("account.sendingCode") : t("account.sendVerificationCode")}
            </Button>
          </div>
        </WorkspaceField>
      </div>

      {challenge ? (
        <div className="grid gap-4 rounded-xl border border-border/60 bg-muted/10 p-4 xl:grid-cols-[minmax(0,1fr)_auto]">
          <WorkspaceField label={t("account.verificationCode")}>
            <Input
              inputMode="numeric"
              maxLength={6}
              onChange={(event) => setCode(event.target.value)}
              placeholder={t("account.verificationCodePlaceholder")}
              value={code}
            />
          </WorkspaceField>
          <div className="flex items-end">
            <Button disabled={isConfirmPending || code.trim().length < 6} size="sm" onClick={handleConfirm}>
              {isConfirmPending ? t("account.verifying") : t("account.confirmEmailChange")}
            </Button>
          </div>
        </div>
      ) : null}

      {notice ? <p className="text-xs text-emerald-600 dark:text-emerald-400">{notice}</p> : null}
      {error ? <p className="text-xs text-destructive">{error}</p> : null}
    </WorkspacePanel>
  );
}
