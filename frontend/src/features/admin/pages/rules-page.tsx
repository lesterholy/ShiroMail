import { useEffect, useMemo, useState } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { Button } from "@/components/ui/button";
import { Card, CardContent } from "@/components/ui/card";
import { Checkbox } from "@/components/ui/checkbox";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import {
  WorkspaceField,
  WorkspacePage,
  WorkspacePanel,
} from "@/components/layout/workspace-ui";
import { fetchAdminRules, upsertAdminRule } from "../api";
import { formatDateTime } from "../../user/pages/shared";

export function AdminRulesPage() {
  const queryClient = useQueryClient();
  const rulesQuery = useQuery({ queryKey: ["admin-rules"], queryFn: fetchAdminRules });
  const [selectedRuleId, setSelectedRuleId] = useState<string>("");
  const [draft, setDraft] = useState({ name: "", retentionHours: 24, autoExtend: false });

  const selectedRule = useMemo(
    () => rulesQuery.data?.find((item) => item.id === selectedRuleId) ?? rulesQuery.data?.[0],
    [rulesQuery.data, selectedRuleId],
  );

  useEffect(() => {
    if (!selectedRule) {
      return;
    }
    setSelectedRuleId(selectedRule.id);
    setDraft({
      name: selectedRule.name,
      retentionHours: selectedRule.retentionHours,
      autoExtend: selectedRule.autoExtend,
    });
  }, [selectedRule]);

  const saveMutation = useMutation({
    mutationFn: async () => {
      const targetId = selectedRuleId || selectedRule?.id || "default";
      return upsertAdminRule(targetId, draft);
    },
    onSuccess: async () => {
      await queryClient.invalidateQueries({ queryKey: ["admin-rules"] });
    },
  });

  return (
    <WorkspacePage>
      <WorkspacePanel description="管理邮箱保留时间与自动续期策略。" title="规则中心">
        <div className="grid gap-6 xl:grid-cols-[0.92fr_1.08fr]">
          <div className="space-y-3">
            {(rulesQuery.data ?? []).map((item) => {
              const active = item.id === selectedRuleId;
              return (
                <button
                  className="block w-full text-left"
                  key={item.id}
                  onClick={() => {
                    setSelectedRuleId(item.id);
                    setDraft({
                      name: item.name,
                      retentionHours: item.retentionHours,
                      autoExtend: item.autoExtend,
                    });
                  }}
                  type="button"
                >
                  <Card className={active ? "border-primary/40 bg-muted/25 shadow-none" : "border-border/60 bg-muted/10 shadow-none"}>
                    <CardContent className="space-y-2 py-4">
                      <div className="flex items-start justify-between gap-3">
                        <div className="space-y-1">
                          <div className="text-sm font-medium">{item.name}</div>
                          <div className="text-xs text-muted-foreground">{item.id}</div>
                        </div>
                        <span className="rounded-full border border-border/60 px-2 py-1 text-xs text-muted-foreground">
                          {item.retentionHours}h
                        </span>
                      </div>
                      <div className="text-xs text-muted-foreground">更新于 {formatDateTime(item.updatedAt)}</div>
                    </CardContent>
                  </Card>
                </button>
              );
            })}
          </div>

          <Card className="border-border/60 bg-muted/10 shadow-none">
            <CardContent className="space-y-4 py-4">
              <WorkspaceField label="规则名称">
                <Input
                  className="h-9"
                  onChange={(event) => setDraft((current) => ({ ...current, name: event.target.value }))}
                  placeholder="规则名称"
                  value={draft.name}
                />
              </WorkspaceField>

              <WorkspaceField label="保留时长（小时）">
                <Input
                  className="h-9"
                  min={1}
                  onChange={(event) => setDraft((current) => ({ ...current, retentionHours: Number(event.target.value) }))}
                  type="number"
                  value={draft.retentionHours}
                />
              </WorkspaceField>

              <div className="flex items-center gap-2">
                <Checkbox
                  checked={draft.autoExtend}
                  id="admin-rule-auto-extend"
                  onCheckedChange={(checked) => setDraft((current) => ({ ...current, autoExtend: checked === true }))}
                />
                <Label className="text-sm" htmlFor="admin-rule-auto-extend">
                  启用自动续期
                </Label>
              </div>

              <Button onClick={() => saveMutation.mutate()}>保存规则</Button>
            </CardContent>
          </Card>
        </div>
      </WorkspacePanel>
    </WorkspacePage>
  );
}
