import { useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { WorkspaceBadge, WorkspaceField, WorkspaceListRow, WorkspacePanel } from "@/components/layout/workspace-ui";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { useAutoDismiss } from "@/hooks/use-auto-dismiss";
import { validateOneTimeCode, validateRequiredText } from "@/lib/validation";
import { Copy } from "lucide-react";
import type { TOTPSetup } from "../api";

export function AccountTOTPCard({
  enabled,
  setupDraft,
  isSetupPending,
  isEnablePending,
  isDisablePending,
  onSetup,
  onEnable,
  onDisable,
}: {
  enabled: boolean;
  setupDraft: TOTPSetup | null;
  isSetupPending: boolean;
  isEnablePending: boolean;
  isDisablePending: boolean;
  onSetup: () => Promise<void>;
  onEnable: (code: string) => Promise<void>;
  onDisable: (password: string) => Promise<void>;
}) {
  const { t } = useTranslation();
  const [setupCode, setSetupCode] = useState("");
  const [disablePassword, setDisablePassword] = useState("");
  const [copyState, setCopyState] = useState<"idle" | "manual" | "url" | "failed">("idle");
  const [notice, setNotice] = useState<string | null>(null);
  const [error, setError] = useState<string | null>(null);
  const manualSteps = useMemo(
    () => [t("account.totpStep1"), t("account.totpStep2"), t("account.totpStep3")],
    [t],
  );

  useAutoDismiss(notice, () => setNotice(null));
  useAutoDismiss(error, () => setError(null));
  useAutoDismiss(copyState === "idle" ? null : copyState, () => setCopyState("idle"));

  async function handleCopy(value: string, type: "manual" | "url") {
    if (typeof navigator === "undefined" || !navigator.clipboard) {
      setCopyState("failed");
      return;
    }
    try {
      await navigator.clipboard.writeText(value);
      setCopyState(type);
    } catch {
      setCopyState("failed");
    }
  }

  async function handleSetup() {
    setError(null);
    try {
      await onSetup();
      setNotice(t("account.totpSetupReady"));
    } catch (currentError) {
      setError(currentError instanceof Error ? currentError.message : t("account.totpSetupFailed"));
    }
  }

  async function handleEnable() {
    setError(null);
    const codeError = validateOneTimeCode(setupCode);
    if (codeError) {
      setError(codeError);
      return;
    }
    try {
      await onEnable(setupCode.trim());
      setSetupCode("");
      setNotice(t("account.totpEnabledNotice"));
    } catch (currentError) {
      setError(currentError instanceof Error ? currentError.message : t("account.totpEnableFailed"));
    }
  }

  async function handleDisable() {
    setError(null);
    const passwordError = validateRequiredText(t("account.currentPassword"), disablePassword, { minLength: 1, maxLength: 256 });
    if (passwordError) {
      setError(passwordError);
      return;
    }
    try {
      await onDisable(disablePassword);
      setDisablePassword("");
      setNotice(t("account.totpDisabledNotice"));
    } catch (currentError) {
      setError(currentError instanceof Error ? currentError.message : t("account.totpDisableFailed"));
    }
  }

  return (
    <WorkspacePanel
      action={
        !enabled ? (
          <Button disabled={isSetupPending} size="sm" onClick={handleSetup}>
            {isSetupPending ? t("account.preparing2FA") : t("account.prepare2FA")}
          </Button>
        ) : null
      }
      description={t("account.totpDescription")}
      title={t("account.totpTitle")}
    >
      <WorkspaceListRow
        description={enabled ? t("account.totpEnabledDescription") : t("account.totpDisabledDescription")}
        meta={
          <WorkspaceBadge variant={enabled ? "secondary" : "outline"}>
            {enabled ? t("account.enabled") : t("account.disabled")}
          </WorkspaceBadge>
        }
        title={t("account.totpStatus")}
      />

      {!enabled && setupDraft ? (
        <div className="grid gap-4 rounded-xl border border-border/60 bg-muted/10 p-4 xl:grid-cols-[minmax(0,1fr)_minmax(0,1fr)]">
          <div className="space-y-3">
            <WorkspaceField label={t("account.manualEntryKey")}>
              <div className="flex gap-2">
                <Input readOnly value={setupDraft.manualEntryKey} />
                <Button
                  aria-label={t("account.copyManualKey")}
                  size="sm"
                  type="button"
                  variant="outline"
                  onClick={() => handleCopy(setupDraft.manualEntryKey, "manual")}
                >
                  <Copy className="size-4" />
                  {copyState === "manual" ? t("account.copied") : t("account.copy")}
                </Button>
              </div>
            </WorkspaceField>
            <WorkspaceField label={t("account.otpauthUrl")}>
              <div className="flex gap-2">
                <Input readOnly value={setupDraft.otpauthUrl} />
                <Button
                  aria-label={t("account.copyOtpAuthUrl")}
                  size="sm"
                  type="button"
                  variant="outline"
                  onClick={() => handleCopy(setupDraft.otpauthUrl, "url")}
                >
                  <Copy className="size-4" />
                  {copyState === "url" ? t("account.copied") : t("account.copy")}
                </Button>
              </div>
            </WorkspaceField>
          </div>
          <div className="space-y-3">
            <div className="space-y-2">
              {manualSteps.map((step) => (
                <p className="text-sm leading-6 text-muted-foreground" key={step}>
                  {step}
                </p>
              ))}
            </div>
            <WorkspaceField label={t("account.verificationCode")}>
              <Input
                inputMode="numeric"
                maxLength={6}
                onChange={(event) => setSetupCode(event.target.value)}
                placeholder={t("account.verificationCodePlaceholder")}
                value={setupCode}
              />
            </WorkspaceField>
            <div className="flex justify-end">
              <Button disabled={isEnablePending || setupCode.trim().length < 6} size="sm" onClick={handleEnable}>
                {isEnablePending ? t("account.verifying") : t("account.enable2FA")}
              </Button>
            </div>
          </div>
        </div>
      ) : null}

      {enabled ? (
        <div className="grid gap-4 rounded-xl border border-border/60 bg-muted/10 p-4 xl:grid-cols-[minmax(0,1fr)_auto]">
          <WorkspaceField label={t("account.currentPassword")}>
            <Input
              autoComplete="current-password"
              onChange={(event) => setDisablePassword(event.target.value)}
              type="password"
              value={disablePassword}
            />
          </WorkspaceField>
          <div className="flex items-end">
            <Button
              disabled={isDisablePending || disablePassword.trim().length === 0}
              size="sm"
              variant="outline"
              onClick={handleDisable}
            >
              {isDisablePending ? t("account.disabling") : t("account.disable2FA")}
            </Button>
          </div>
        </div>
      ) : null}

      {copyState === "failed" ? <p className="text-xs text-destructive">{t("account.copyFailed")}</p> : null}
      {notice ? <p className="text-xs text-emerald-600 dark:text-emerald-400">{notice}</p> : null}
      {error ? <p className="text-xs text-destructive">{error}</p> : null}
    </WorkspacePanel>
  );
}
