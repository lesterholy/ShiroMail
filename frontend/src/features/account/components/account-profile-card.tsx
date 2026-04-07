import { useEffect, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { WorkspaceField, WorkspacePanel } from "@/components/layout/workspace-ui";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { OptionCombobox, type OptionComboboxOption } from "@/components/ui/option-combobox";
import { useAutoDismiss } from "@/hooks/use-auto-dismiss";
import { changeAppLanguage } from "@/lib/i18n";
import { normalizeLanguage, type SupportedLanguage } from "@/lib/preferences";
import { validateIntegerRange, validateRequiredText, validateSelection } from "@/lib/validation";
import type { AccountProfile, UpdateAccountProfileInput } from "../api";

type ProfileDraft = {
  displayName: string;
  locale: SupportedLanguage;
  timezone: string;
  autoRefreshSeconds: number;
};

export function AccountProfileCard({
  profile,
  isSaving,
  onSave,
}: {
  profile: AccountProfile;
  isSaving: boolean;
  onSave: (input: UpdateAccountProfileInput) => Promise<void>;
}) {
  const { t } = useTranslation();
  const [notice, setNotice] = useState<string | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [draft, setDraft] = useState<ProfileDraft>({
    displayName: "",
    locale: "zh-CN",
    timezone: "Asia/Shanghai",
    autoRefreshSeconds: 30,
  });

  useAutoDismiss(notice, () => setNotice(null));
  useAutoDismiss(error, () => setError(null));

  useEffect(() => {
    setDraft({
      displayName: profile.displayName,
      locale: normalizeLanguage(profile.locale),
      timezone: profile.timezone,
      autoRefreshSeconds: profile.autoRefreshSeconds,
    });
  }, [profile.autoRefreshSeconds, profile.displayName, profile.locale, profile.timezone]);

  const languageOptions = useMemo<OptionComboboxOption[]>(
    () => [
      { value: "zh-CN", label: t("common.simplifiedChinese") },
      { value: "en-US", label: t("common.english") },
    ],
    [t],
  );
  const timezoneOptions = useMemo<OptionComboboxOption[]>(
    () => [
      { value: "Asia/Shanghai", label: "Asia/Shanghai" },
      { value: "UTC", label: "UTC" },
    ],
    [],
  );

  async function handleSave() {
    setError(null);
    const displayNameError = validateRequiredText(t("account.displayName"), draft.displayName, { minLength: 2, maxLength: 64 });
    if (displayNameError) {
      setError(displayNameError);
      return;
    }
    const localeError = validateSelection(t("account.language"), draft.locale, languageOptions.map((item) => item.value));
    if (localeError) {
      setError(localeError);
      return;
    }
    const timezoneError = validateSelection(t("account.timezone"), draft.timezone, timezoneOptions.map((item) => item.value));
    if (timezoneError) {
      setError(timezoneError);
      return;
    }
    const refreshError = validateIntegerRange(t("account.autoRefreshSeconds"), draft.autoRefreshSeconds, { min: 15, max: 3600 });
    if (refreshError) {
      setError(refreshError);
      return;
    }
    try {
      await onSave({
        displayName: draft.displayName.trim(),
        locale: draft.locale,
        timezone: draft.timezone,
        autoRefreshSeconds: draft.autoRefreshSeconds,
      });
      await changeAppLanguage(draft.locale);
      setNotice(t("account.profileSaved"));
    } catch (currentError) {
      setError(currentError instanceof Error ? currentError.message : t("account.profileSaveFailed"));
    }
  }

  return (
    <WorkspacePanel
      action={
        <Button disabled={isSaving} size="sm" onClick={handleSave}>
          {isSaving ? t("common.saving") : t("common.save")}
        </Button>
      }
      description={t("account.profileDescription")}
      title={t("account.profileTitle")}
    >
      <div className="grid gap-4 xl:grid-cols-2">
        <WorkspaceField label={t("account.username")}>
          <Input disabled value={profile.username} />
        </WorkspaceField>

        <WorkspaceField label={t("account.displayName")}>
          <Input
            onChange={(event) => setDraft((current) => ({ ...current, displayName: event.target.value }))}
            placeholder={t("account.displayNamePlaceholder")}
            value={draft.displayName}
          />
        </WorkspaceField>

        <WorkspaceField label={t("account.language")}>
          <OptionCombobox
            ariaLabel={t("account.language")}
            emptyLabel={t("common.noData")}
            options={languageOptions}
            placeholder={t("account.language")}
            searchPlaceholder={t("common.search")}
            value={draft.locale}
            onValueChange={(value) => setDraft((current) => ({ ...current, locale: value as SupportedLanguage }))}
          />
        </WorkspaceField>

        <WorkspaceField label={t("account.timezone")}>
          <OptionCombobox
            ariaLabel={t("account.timezone")}
            emptyLabel={t("common.noData")}
            options={timezoneOptions}
            placeholder={t("account.timezone")}
            searchPlaceholder={t("common.search")}
            value={draft.timezone}
            onValueChange={(value) => setDraft((current) => ({ ...current, timezone: value }))}
          />
        </WorkspaceField>

        <WorkspaceField label={t("account.autoRefreshSeconds")}>
          <Input
            min={15}
            onChange={(event) =>
              setDraft((current) => ({
                ...current,
                autoRefreshSeconds: Number(event.target.value || 0),
              }))
            }
            type="number"
            value={draft.autoRefreshSeconds}
          />
        </WorkspaceField>
      </div>

      {notice ? <p className="text-xs text-emerald-600 dark:text-emerald-400">{notice}</p> : null}
      {error ? <p className="text-xs text-destructive">{error}</p> : null}
    </WorkspacePanel>
  );
}
