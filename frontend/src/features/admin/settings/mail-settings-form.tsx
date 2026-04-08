import { Checkbox } from "@/components/ui/checkbox";
import { BasicSelect } from "@/components/ui/basic-select";
import { Input } from "@/components/ui/input";
import { WorkspaceField } from "@/components/layout/workspace-ui";
import type { MailDeliverySettings, MailInboundSettings, MailSMTPSettings } from "./types";

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

export function MailSettingsForm({
  smtp,
  delivery,
  inbound,
  onSMTPChange,
  onDeliveryChange,
  onInboundChange,
  mode = "all",
}: {
  smtp: MailSMTPSettings;
  delivery: MailDeliverySettings;
  inbound: MailInboundSettings;
  onSMTPChange: (next: MailSMTPSettings) => void;
  onDeliveryChange: (next: MailDeliverySettings) => void;
  onInboundChange: (next: MailInboundSettings) => void;
  mode?: "all" | "smtp" | "delivery" | "inbound";
}) {
  const showSMTP = mode === "all" || mode === "smtp";
  const showDelivery = mode === "all" || mode === "delivery";
  const showInbound = mode === "all" || mode === "inbound";

  return (
    <div className="space-y-4">
      {showSMTP ? (
        <div className="grid gap-3 md:grid-cols-2 lg:grid-cols-5">
          <WorkspaceField label="SMTP Hostname / MX Target">
            <Input
              aria-label="SMTP Hostname"
              value={smtp.hostname}
              onChange={(event) =>
                onSMTPChange({ ...smtp, hostname: event.target.value })
              }
            />
          </WorkspaceField>
          <WorkspaceField label="DKIM CNAME Target">
            <Input
              aria-label="DKIM CNAME Target"
              value={smtp.dkimCnameTarget}
              onChange={(event) =>
                onSMTPChange({ ...smtp, dkimCnameTarget: event.target.value })
              }
            />
          </WorkspaceField>
          <WorkspaceField label="监听地址">
            <Input
              aria-label="监听地址"
              value={smtp.listenAddr}
              onChange={(event) =>
                onSMTPChange({ ...smtp, listenAddr: event.target.value })
              }
            />
          </WorkspaceField>
          <WorkspaceField label="最大消息字节">
            <Input
              aria-label="最大消息字节"
              type="number"
              value={String(smtp.maxMessageBytes)}
              onChange={(event) =>
                onSMTPChange({
                  ...smtp,
                  maxMessageBytes: Number(event.target.value || 0),
                })
              }
            />
          </WorkspaceField>
          <div className="flex items-end">
            <CheckboxField
              label="启用 SMTP 收件"
              checked={smtp.enabled}
              onCheckedChange={(enabled) => onSMTPChange({ ...smtp, enabled })}
            />
          </div>
        </div>
      ) : null}

      {showDelivery ? (
        <>
          <div className="grid gap-3 md:grid-cols-2 lg:grid-cols-4">
            <WorkspaceField label="发信 SMTP Host">
              <Input aria-label="发信 SMTP Host" value={delivery.host} onChange={(event) => onDeliveryChange({ ...delivery, host: event.target.value })} />
            </WorkspaceField>
            <WorkspaceField label="发信端口">
              <Input aria-label="发信端口" type="number" value={String(delivery.port)} onChange={(event) => onDeliveryChange({ ...delivery, port: Number(event.target.value || 0) })} />
            </WorkspaceField>
            <WorkspaceField label="传输模式">
              <BasicSelect
                aria-label="传输模式"
                value={delivery.transportMode}
                onChange={(event) => onDeliveryChange({ ...delivery, transportMode: event.target.value })}
              >
                <option value="plain">Plain SMTP</option>
                <option value="starttls">STARTTLS</option>
                <option value="smtps">SMTPS</option>
              </BasicSelect>
            </WorkspaceField>
            <WorkspaceField label="发信账号">
              <Input aria-label="发信账号" value={delivery.username} onChange={(event) => onDeliveryChange({ ...delivery, username: event.target.value })} />
            </WorkspaceField>
            <WorkspaceField label="发件邮箱">
              <Input aria-label="发件邮箱" value={delivery.fromAddress} onChange={(event) => onDeliveryChange({ ...delivery, fromAddress: event.target.value })} />
            </WorkspaceField>
            <WorkspaceField label="发件人名称">
              <Input aria-label="发件人名称" value={delivery.fromName} onChange={(event) => onDeliveryChange({ ...delivery, fromName: event.target.value })} />
            </WorkspaceField>
            <div className="md:col-span-2">
              <WorkspaceField label="SMTP 密码 / App Password">
                <Input aria-label="SMTP 密码 / App Password" type="password" value={delivery.password} onChange={(event) => onDeliveryChange({ ...delivery, password: event.target.value })} />
              </WorkspaceField>
            </div>
            <div className="flex items-end">
              <CheckboxField
                label="启用账户邮件发信"
                checked={delivery.enabled}
                onCheckedChange={(enabled) => onDeliveryChange({ ...delivery, enabled })}
              />
            </div>
            <div className="flex items-end">
              <CheckboxField
                label="跳过 TLS 证书校验"
                checked={delivery.insecureSkipVerify}
                onCheckedChange={(insecureSkipVerify) => onDeliveryChange({ ...delivery, insecureSkipVerify })}
              />
            </div>
          </div>
        </>
      ) : null}

      {showInbound ? (
        <>
          <div className="grid gap-3 md:grid-cols-3">
            <WorkspaceField label="原文保留天数">
              <Input
                aria-label="原文保留天数"
                type="number"
                value={String(inbound.retainRawDays)}
                onChange={(event) =>
                  onInboundChange({
                    ...inbound,
                    retainRawDays: Number(event.target.value || 0),
                  })
                }
              />
            </WorkspaceField>
            <WorkspaceField label="附件大小 MB">
              <Input
                aria-label="附件大小 MB"
                type="number"
                value={String(inbound.maxAttachmentSizeMB)}
                onChange={(event) =>
                  onInboundChange({
                    ...inbound,
                    maxAttachmentSizeMB: Number(event.target.value || 0),
                  })
                }
              />
            </WorkspaceField>
          </div>

          <div className="grid gap-2 md:grid-cols-2">
            <CheckboxField
              label="仅允许已创建邮箱接收"
              checked={inbound.requireExistingMailbox}
              onCheckedChange={(requireExistingMailbox) =>
                onInboundChange({ ...inbound, requireExistingMailbox })
              }
            />
            <CheckboxField
              label="允许 catch-all"
              checked={inbound.allowCatchAll}
              onCheckedChange={(allowCatchAll) =>
                onInboundChange({ ...inbound, allowCatchAll })
              }
            />
            <CheckboxField
              label="拒绝可执行附件"
              checked={inbound.rejectExecutableFiles}
              onCheckedChange={(rejectExecutableFiles) =>
                onInboundChange({ ...inbound, rejectExecutableFiles })
              }
            />
            <CheckboxField
              label="启用垃圾邮件扫描预览"
              checked={inbound.enableSpamScanningPreview}
              onCheckedChange={(enableSpamScanningPreview) =>
                onInboundChange({ ...inbound, enableSpamScanningPreview })
              }
            />
          </div>
        </>
      ) : null}
    </div>
  );
}
