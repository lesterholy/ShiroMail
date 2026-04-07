import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { Button } from "@/components/ui/button";
import { Card, CardContent } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { OptionCombobox } from "@/components/ui/option-combobox";
import { Textarea } from "@/components/ui/textarea";
import { WorkspaceBadge, WorkspaceEmpty, WorkspaceField, WorkspacePage, WorkspacePanel } from "@/components/layout/workspace-ui";
import { getAPIErrorMessage } from "@/lib/http";
import { validateRequiredText, validateSelection } from "@/lib/validation";
import { useState } from "react";
import { createFeedback, fetchFeedback } from "../api";
import { formatDateTime } from "./shared";

export function UserFeedbackPage() {
  const queryClient = useQueryClient();
  const [draft, setDraft] = useState({ category: "product", subject: "", content: "" });
  const [formError, setFormError] = useState<string | null>(null);
  const feedbackQuery = useQuery({ queryKey: ["portal-feedback"], queryFn: fetchFeedback });
  const categoryOptions = [
    { value: "product", label: "产品体验", keywords: ["体验", "product"] },
    { value: "bug", label: "Bug 报告", keywords: ["问题", "bug"] },
    { value: "billing", label: "计费问题", keywords: ["支付", "billing"] },
  ];

  const createMutation = useMutation({
    mutationFn: createFeedback,
    onSuccess: async () => {
      setFormError(null);
      setDraft({ category: "product", subject: "", content: "" });
      await queryClient.invalidateQueries({ queryKey: ["portal-feedback"] });
      await queryClient.invalidateQueries({ queryKey: ["portal-overview"] });
    },
    onError: (error) => {
      setFormError(getAPIErrorMessage(error, "提交反馈失败，请稍后重试。"));
    },
  });

  function handleSubmit() {
    const categoryError = validateSelection("反馈类型", draft.category, categoryOptions.map((item) => item.value));
    if (categoryError) {
      setFormError(categoryError);
      return;
    }
    const subjectError = validateRequiredText("反馈标题", draft.subject, { minLength: 2, maxLength: 120 });
    if (subjectError) {
      setFormError(subjectError);
      return;
    }
    const contentError = validateRequiredText("问题描述", draft.content, { minLength: 5, maxLength: 5000 });
    if (contentError) {
      setFormError(contentError);
      return;
    }
    setFormError(null);
    createMutation.mutate({
      category: draft.category,
      subject: draft.subject.trim(),
      content: draft.content.trim(),
    });
  }

  return (
    <WorkspacePage>
      <WorkspacePanel description="提交问题、体验建议或功能需求，并查看处理状态。" title="反馈">
        <div className="grid gap-4 xl:grid-cols-[0.92fr_1.08fr]">
          <Card className="border-border/60 bg-muted/10 shadow-none">
            <CardContent className="flex flex-col gap-4 py-4">
              <WorkspaceField label="反馈类型">
                <OptionCombobox
                  ariaLabel="反馈类型"
                  emptyLabel="没有匹配的反馈类型"
                  value={draft.category}
                  onValueChange={(value) => setDraft((current) => ({ ...current, category: value }))}
                  options={categoryOptions}
                  placeholder="选择反馈类型"
                  searchPlaceholder="搜索反馈类型"
                />
              </WorkspaceField>

              <WorkspaceField label="反馈标题">
                <Input
                  onChange={(event) => setDraft((current) => ({ ...current, subject: event.target.value }))}
                  placeholder="反馈标题"
                  value={draft.subject}
                />
              </WorkspaceField>

              <WorkspaceField label="问题描述">
                <Textarea
                  onChange={(event) => setDraft((current) => ({ ...current, content: event.target.value }))}
                  placeholder="描述你的问题或需求"
                  rows={6}
                  value={draft.content}
                />
              </WorkspaceField>

              {formError ? <p className="text-xs text-destructive">{formError}</p> : null}

              <Button disabled={createMutation.isPending} onClick={handleSubmit}>
                {createMutation.isPending ? "提交中..." : "提交反馈"}
              </Button>
            </CardContent>
          </Card>

          <div className="flex flex-col gap-3">
            {feedbackQuery.data?.length ? (
              feedbackQuery.data.map((item) => (
                <Card className="border-border/60 bg-card/85 shadow-none" key={item.id}>
                  <CardContent className="flex flex-col gap-3 py-4">
                    <div className="flex flex-wrap items-center justify-between gap-2">
                      <WorkspaceBadge>{item.category}</WorkspaceBadge>
                      <span className="text-xs text-muted-foreground">{formatDateTime(item.updatedAt)}</span>
                    </div>
                    <div className="text-sm font-medium">{item.subject}</div>
                    <p className="text-sm leading-6 text-muted-foreground">{item.content}</p>
                    <div className="text-xs text-muted-foreground">状态：{item.status}</div>
                  </CardContent>
                </Card>
              ))
            ) : (
              <WorkspaceEmpty description="还没有历史反馈记录，提交一条后就会在这里显示。" title="暂无反馈记录" />
            )}
          </div>
        </div>
      </WorkspacePanel>
    </WorkspacePage>
  );
}
