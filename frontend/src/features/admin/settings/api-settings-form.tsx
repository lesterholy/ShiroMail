import type { ReactNode } from "react";
import { WorkspaceField } from "@/components/layout/workspace-ui";
import { Checkbox } from "@/components/ui/checkbox";
import { BasicSelect } from "@/components/ui/basic-select";
import { Input } from "@/components/ui/input";
import type { APILimitsSettings } from "./types";

function CheckboxField({
  label,
  checked,
  onCheckedChange,
}: {
  label: string;
  checked: boolean;
  onCheckedChange: (next: boolean) => void;
}) {
  return (
    <label className="flex items-center gap-3 rounded-lg border border-border/60 px-3 py-2 text-sm">
      <Checkbox
        checked={checked}
        onCheckedChange={(next) => onCheckedChange(next === true)}
      />
      <span>{label}</span>
    </label>
  );
}

function NumberField({
  label,
  ariaLabel,
  value,
  onChange,
}: {
  label: string;
  ariaLabel: string;
  value: number;
  onChange: (next: number) => void;
}) {
  return (
    <WorkspaceField label={label}>
      <Input
        aria-label={ariaLabel}
        type="number"
        min={0}
        value={String(value)}
        onChange={(event) => onChange(Number(event.target.value || 0))}
      />
    </WorkspaceField>
  );
}

function SettingsBlock({
  title,
  description,
  children,
}: {
  title: string;
  description: string;
  children: ReactNode;
}) {
  return (
    <section className="space-y-3 rounded-2xl border border-border/60 bg-card/70 p-4">
      <div className="space-y-1">
        <div className="text-sm font-semibold text-foreground">{title}</div>
        <p className="text-sm leading-6 text-muted-foreground">{description}</p>
      </div>
      {children}
    </section>
  );
}

export function APISettingsForm({
  value,
  onChange,
}: {
  value: APILimitsSettings;
  onChange: (next: APILimitsSettings) => void;
}) {
  return (
    <div className="space-y-4">
      <div className="grid gap-3 lg:grid-cols-3">
        <div className="rounded-2xl border border-border/60 bg-muted/20 p-4">
          <div className="text-xs font-medium uppercase tracking-[0.18em] text-muted-foreground">
            Identity Bucket
          </div>
          <div className="mt-2 text-base font-semibold text-foreground">
            {value.identityMode === "ip" ? "All requests use IP buckets" : "Bearer tokens split authenticated traffic"}
          </div>
          <p className="mt-2 text-sm leading-6 text-muted-foreground">
            匿名请求始终按 IP 统计；已认证请求可按 Bearer Token 分桶，避免多个用户共用出口 IP 时互相抢额度。
          </p>
        </div>
        <div className="rounded-2xl border border-border/60 bg-muted/20 p-4">
          <div className="text-xs font-medium uppercase tracking-[0.18em] text-muted-foreground">
            Hot Reload
          </div>
          <div className="mt-2 text-base font-semibold text-foreground">
            Save to MySQL and refresh automatically
          </div>
          <p className="mt-2 text-sm leading-6 text-muted-foreground">
            保存后后端会自动轮询配置并刷新限流策略，通常几秒内生效，不需要手动重启 `app` 服务。
          </p>
        </div>
        <div className="rounded-2xl border border-border/60 bg-muted/20 p-4">
          <div className="text-xs font-medium uppercase tracking-[0.18em] text-muted-foreground">
            Strict IP Guard
          </div>
          <div className="mt-2 text-base font-semibold text-foreground">
            {value.strictIpEnabled ? "Enabled" : "Disabled"}
          </div>
          <p className="mt-2 text-sm leading-6 text-muted-foreground">
            严格 IP 限流会在主桶之外再套一层纯 IP 限制，适合拦截共享代理、刷子或单源爆破流量。
          </p>
        </div>
      </div>

      <SettingsBlock
        title="全局请求桶"
        description="控制整站 API 的主请求流量，包括匿名访问、已认证访问和用于保护登录注册链路的总认证桶。"
      >
        <div className="grid gap-3 md:grid-cols-2 xl:grid-cols-4">
          <NumberField
            label="匿名请求 RPM"
            ariaLabel="匿名请求 RPM"
            value={value.anonymousRPM}
            onChange={(anonymousRPM) => onChange({ ...value, anonymousRPM })}
          />
          <NumberField
            label="已认证请求 RPM"
            ariaLabel="已认证请求 RPM"
            value={value.authenticatedRPM}
            onChange={(authenticatedRPM) =>
              onChange({ ...value, authenticatedRPM })
            }
          />
          <NumberField
            label="认证接口总 RPM"
            ariaLabel="认证接口总 RPM"
            value={value.authRPM}
            onChange={(authRPM) => onChange({ ...value, authRPM })}
          />
          <WorkspaceField label="身份桶策略">
            <BasicSelect
              aria-label="身份桶策略"
              value={value.identityMode}
              onChange={(event) =>
                onChange({ ...value, identityMode: event.target.value })
              }
            >
              <option value="bearer_or_ip">Bearer / IP 混合</option>
              <option value="ip">仅按 IP</option>
            </BasicSelect>
          </WorkspaceField>
        </div>
      </SettingsBlock>

      <SettingsBlock
        title="认证链路"
        description="分别限制登录、注册、找回密码、邮箱验证、OAuth 和 2FA 验证，便于针对不同攻击面做更严谨的配额。"
      >
        <div className="grid gap-3 md:grid-cols-2 xl:grid-cols-4">
          <NumberField
            label="登录 RPM"
            ariaLabel="登录 RPM"
            value={value.loginRPM}
            onChange={(loginRPM) => onChange({ ...value, loginRPM })}
          />
          <NumberField
            label="注册 RPM"
            ariaLabel="注册 RPM"
            value={value.registerRPM}
            onChange={(registerRPM) => onChange({ ...value, registerRPM })}
          />
          <NumberField
            label="Refresh RPM"
            ariaLabel="Refresh RPM"
            value={value.refreshRPM}
            onChange={(refreshRPM) => onChange({ ...value, refreshRPM })}
          />
          <NumberField
            label="2FA Verify RPM"
            ariaLabel="2FA Verify RPM"
            value={value.login2faVerifyRPM}
            onChange={(login2faVerifyRPM) =>
              onChange({ ...value, login2faVerifyRPM })
            }
          />
          <NumberField
            label="忘记密码 RPM"
            ariaLabel="忘记密码 RPM"
            value={value.forgotPasswordRPM}
            onChange={(forgotPasswordRPM) =>
              onChange({ ...value, forgotPasswordRPM })
            }
          />
          <NumberField
            label="重置密码 RPM"
            ariaLabel="重置密码 RPM"
            value={value.resetPasswordRPM}
            onChange={(resetPasswordRPM) =>
              onChange({ ...value, resetPasswordRPM })
            }
          />
          <NumberField
            label="重发邮箱验证 RPM"
            ariaLabel="重发邮箱验证 RPM"
            value={value.emailVerificationResendRPM}
            onChange={(emailVerificationResendRPM) =>
              onChange({ ...value, emailVerificationResendRPM })
            }
          />
          <NumberField
            label="确认邮箱验证 RPM"
            ariaLabel="确认邮箱验证 RPM"
            value={value.emailVerificationConfirmRPM}
            onChange={(emailVerificationConfirmRPM) =>
              onChange({ ...value, emailVerificationConfirmRPM })
            }
          />
          <NumberField
            label="OAuth Start RPM"
            ariaLabel="OAuth Start RPM"
            value={value.oauthStartRPM}
            onChange={(oauthStartRPM) => onChange({ ...value, oauthStartRPM })}
          />
          <NumberField
            label="OAuth Callback RPM"
            ariaLabel="OAuth Callback RPM"
            value={value.oauthCallbackRPM}
            onChange={(oauthCallbackRPM) =>
              onChange({ ...value, oauthCallbackRPM })
            }
          />
        </div>
      </SettingsBlock>

      <SettingsBlock
        title="邮箱写操作与兜底保护"
        description="针对临时邮箱创建、续期、释放等写操作做单独限制；严格 IP 桶可作为最后一道防线。"
      >
        <div className="grid gap-3 md:grid-cols-2 xl:grid-cols-3">
          <NumberField
            label="邮箱写操作 RPM"
            ariaLabel="邮箱写操作 RPM"
            value={value.mailboxWriteRPM}
            onChange={(mailboxWriteRPM) =>
              onChange({ ...value, mailboxWriteRPM })
            }
          />
          <NumberField
            label="严格 IP RPM"
            ariaLabel="严格 IP RPM"
            value={value.strictIpRPM}
            onChange={(strictIpRPM) => onChange({ ...value, strictIpRPM })}
          />
          <div className="flex items-end">
            <CheckboxField
              label="启用 API 限流"
              checked={value.enabled}
              onCheckedChange={(enabled) => onChange({ ...value, enabled })}
            />
          </div>
          <div className="flex items-end">
            <CheckboxField
              label="启用严格 IP 限流"
              checked={value.strictIpEnabled}
              onCheckedChange={(strictIpEnabled) =>
                onChange({ ...value, strictIpEnabled })
              }
            />
          </div>
        </div>
      </SettingsBlock>

      <div className="rounded-xl border border-border/60 bg-muted/20 p-3 text-sm text-muted-foreground">
        保存后会写入 MySQL 配置表；后端会自动轮询刷新限流配置，通常在数秒内生效，无需手动重启 `app` 服务。
      </div>
    </div>
  );
}
