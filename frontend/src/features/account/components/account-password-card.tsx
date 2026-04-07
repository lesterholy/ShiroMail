import { useState } from "react";
import { useTranslation } from "react-i18next";
import { WorkspaceField, WorkspacePanel } from "@/components/layout/workspace-ui";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { useAutoDismiss } from "@/hooks/use-auto-dismiss";
import { validateRequiredText } from "@/lib/validation";

export function AccountPasswordCard({
  isPending,
  onSubmit,
}: {
  isPending: boolean;
  onSubmit: (input: { currentPassword: string; newPassword: string }) => Promise<void>;
}) {
  const { t } = useTranslation();
  const [currentPassword, setCurrentPassword] = useState("");
  const [newPassword, setNewPassword] = useState("");
  const [notice, setNotice] = useState<string | null>(null);
  const [error, setError] = useState<string | null>(null);

  useAutoDismiss(notice, () => setNotice(null));
  useAutoDismiss(error, () => setError(null));

  async function handleSubmit() {
    setError(null);
    const currentPasswordError = validateRequiredText(t("account.currentPassword"), currentPassword, { minLength: 1, maxLength: 256 });
    if (currentPasswordError) {
      setError(currentPasswordError);
      return;
    }
    const newPasswordError = validateRequiredText(t("account.newPassword"), newPassword, { minLength: 8, maxLength: 256 });
    if (newPasswordError) {
      setError(newPasswordError);
      return;
    }
    if (currentPassword === newPassword) {
      setError("新密码不能与当前密码相同。");
      return;
    }
    try {
      await onSubmit({ currentPassword, newPassword });
      setCurrentPassword("");
      setNewPassword("");
      setNotice(t("account.passwordUpdated"));
    } catch (currentError) {
      setError(currentError instanceof Error ? currentError.message : t("account.passwordUpdateFailed"));
    }
  }

  return (
    <WorkspacePanel
      action={
        <Button
          disabled={isPending || currentPassword.trim().length === 0 || newPassword.trim().length === 0}
          size="sm"
          onClick={handleSubmit}
        >
          {isPending ? t("common.saving") : t("account.updatePassword")}
        </Button>
      }
      description={t("account.passwordDescription")}
      title={t("account.passwordTitle")}
    >
      <div className="grid gap-4 xl:grid-cols-2">
        <WorkspaceField label={t("account.currentPassword")}>
          <Input
            autoComplete="current-password"
            onChange={(event) => setCurrentPassword(event.target.value)}
            type="password"
            value={currentPassword}
          />
        </WorkspaceField>

        <WorkspaceField label={t("account.newPassword")}>
          <Input
            autoComplete="new-password"
            onChange={(event) => setNewPassword(event.target.value)}
            type="password"
            value={newPassword}
          />
        </WorkspaceField>
      </div>

      {notice ? <p className="text-xs text-emerald-600 dark:text-emerald-400">{notice}</p> : null}
      {error ? <p className="text-xs text-destructive">{error}</p> : null}
    </WorkspacePanel>
  );
}
