import { Checkbox } from "@/components/ui/checkbox";
import type { DomainPolicySettings } from "./types";

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

export function DomainPolicyForm({
  value,
  onChange,
}: {
  value: DomainPolicySettings;
  onChange: (next: DomainPolicySettings) => void;
}) {
  return (
    <div className="grid gap-3">
      <CheckboxField
        label="公开域发布需要审核"
        checked={value.requiresReview}
        onCheckedChange={(requiresReview) => onChange({ requiresReview })}
      />
      <div className="rounded-xl border border-border/60 bg-muted/10 px-4 py-3">
        <p className="text-sm font-medium">平台治理说明</p>
        <p className="mt-1 text-sm leading-6 text-muted-foreground">
          这里收口公共域池审核、后续风控开关，以及整站级域名平台治理策略。
        </p>
      </div>
    </div>
  );
}
