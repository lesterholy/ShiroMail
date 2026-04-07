import { Input } from "@/components/ui/input";
import { WorkspaceField } from "@/components/layout/workspace-ui";
import type { SiteIdentitySettings } from "./types";

export function SiteSettingsForm({
  identity,
  onIdentityChange,
}: {
  identity: SiteIdentitySettings;
  onIdentityChange: (next: SiteIdentitySettings) => void;
}) {
  return (
    <div className="grid gap-3 md:grid-cols-2">
      <WorkspaceField label="站点名称">
        <Input
          aria-label="站点名称"
          value={identity.siteName}
          onChange={(event) =>
            onIdentityChange({ ...identity, siteName: event.target.value })
          }
        />
      </WorkspaceField>

      <WorkspaceField label="支持邮箱">
        <Input
          aria-label="支持邮箱"
          value={identity.supportEmail}
          onChange={(event) =>
            onIdentityChange({
              ...identity,
              supportEmail: event.target.value,
            })
          }
        />
      </WorkspaceField>

      <WorkspaceField label="站点地址">
        <Input
          aria-label="站点地址"
          value={identity.appBaseUrl}
          onChange={(event) =>
            onIdentityChange({
              ...identity,
              appBaseUrl: event.target.value,
            })
          }
        />
      </WorkspaceField>

      <WorkspaceField label="默认语言">
        <Input
          aria-label="默认语言"
          value={identity.defaultLanguage}
          onChange={(event) =>
            onIdentityChange({
              ...identity,
              defaultLanguage: event.target.value,
            })
          }
        />
      </WorkspaceField>

      <WorkspaceField label="默认时区">
        <Input
          aria-label="默认时区"
          value={identity.defaultTimeZone}
          onChange={(event) =>
            onIdentityChange({
              ...identity,
              defaultTimeZone: event.target.value,
            })
          }
        />
      </WorkspaceField>

      <div className="md:col-span-2">
        <WorkspaceField label="站点标语">
          <Input
            aria-label="站点标语"
            value={identity.slogan}
            onChange={(event) =>
              onIdentityChange({ ...identity, slogan: event.target.value })
            }
          />
        </WorkspaceField>
      </div>
    </div>
  );
}
