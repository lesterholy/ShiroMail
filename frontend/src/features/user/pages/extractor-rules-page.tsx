import { useMemo, useState } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { Plus, RefreshCw, Sparkles, Wand2 } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Card, CardContent } from "@/components/ui/card";
import { Checkbox } from "@/components/ui/checkbox";
import { Input } from "@/components/ui/input";
import { Textarea } from "@/components/ui/textarea";
import { BasicSelect } from "@/components/ui/basic-select";
import {
  WorkspaceBadge,
  WorkspaceEmpty,
  WorkspaceField,
  WorkspacePage,
  WorkspacePanel,
} from "@/components/layout/workspace-ui";
import { getAPIErrorMessage } from "@/lib/http";
import {
  copyMailExtractorTemplate,
  createMailExtractorRule,
  deleteMailExtractorRule,
  disableMailExtractorTemplate,
  enableMailExtractorTemplate,
  fetchDashboard,
  fetchMailboxMessages,
  fetchMailExtractorRules,
  type MailExtractorRule,
  testMailExtractorRule,
  updateMailExtractorRule,
} from "../api";
import { emptyRuleDraft, normalizeMailExtractorRule, toRuleDraft, validateRuleDraft, type RuleDraft } from "../extractor-rule-form";

const targetFieldOptions = [
  { value: "subject", label: "标题" },
  { value: "from_addr", label: "发件人" },
  { value: "to_addr", label: "收件人" },
  { value: "text_body", label: "正文" },
  { value: "html_text", label: "HTML 文本" },
  { value: "raw_text", label: "Raw" },
] as const;

const resultModeOptions = [
  { value: "first_match", label: "首个匹配" },
  { value: "all_matches", label: "全部匹配" },
  { value: "capture_group", label: "指定分组" },
] as const;

function targetSummary(rule: Pick<MailExtractorRule, "targetFields">) {
  return targetFieldOptions
    .filter((item) => (rule.targetFields ?? []).includes(item.value))
    .map((item) => item.label)
    .join(" / ");
}

export function UserExtractorRulesPage() {
  const queryClient = useQueryClient();
  const [selectedRuleId, setSelectedRuleId] = useState<number | "new">("new");
  const [draft, setDraft] = useState<RuleDraft>(emptyRuleDraft);
  const [feedback, setFeedback] = useState<string | null>(null);
  const [sampleMailboxId, setSampleMailboxId] = useState("");
  const [sampleMessageId, setSampleMessageId] = useState("");

  const rulesQuery = useQuery({
    queryKey: ["mail-extractor-rules"],
    queryFn: fetchMailExtractorRules,
  });
  const dashboardQuery = useQuery({
    queryKey: ["user-dashboard"],
    queryFn: fetchDashboard,
  });

  const mailboxes = useMemo(
    () => (dashboardQuery.data?.mailboxes ?? []).filter((mailbox) => mailbox.status === "active"),
    [dashboardQuery.data?.mailboxes],
  );
  const domains = useMemo(() => dashboardQuery.data?.availableDomains ?? [], [dashboardQuery.data?.availableDomains]);
  const selectedMailboxId = sampleMailboxId ? Number(sampleMailboxId) : mailboxes[0]?.id;
  const messagesQuery = useQuery({
    queryKey: ["extractor-rule-sample-messages", selectedMailboxId],
    queryFn: () => fetchMailboxMessages(selectedMailboxId!),
    enabled: Boolean(selectedMailboxId),
    staleTime: 10_000,
  });

  const userRules = (rulesQuery.data?.rules ?? []).map(normalizeMailExtractorRule);
  const templates = (rulesQuery.data?.templates ?? []).map(normalizeMailExtractorRule);
  const sampleMessages = messagesQuery.data ?? [];

  const saveMutation = useMutation({
    mutationFn: async () => {
      const payload = {
        ...draft,
        captureGroupIndex: draft.resultMode === "capture_group" ? Number(draft.captureGroupIndex ?? 1) : undefined,
      };
      if (selectedRuleId === "new") {
        return createMailExtractorRule(payload);
      }
      return updateMailExtractorRule(selectedRuleId, payload);
    },
    onSuccess: async (savedRule) => {
      setFeedback("提取规则已保存。");
      setSelectedRuleId(savedRule.id);
      setDraft(toRuleDraft(savedRule));
      await queryClient.invalidateQueries({ queryKey: ["mail-extractor-rules"] });
    },
    onError: (error) => setFeedback(getAPIErrorMessage(error, "保存提取规则失败。")),
  });

  const deleteMutation = useMutation({
    mutationFn: async (ruleId: number) => deleteMailExtractorRule(ruleId),
    onSuccess: async () => {
      setFeedback("提取规则已删除。");
      setSelectedRuleId("new");
      setDraft(emptyRuleDraft());
      await queryClient.invalidateQueries({ queryKey: ["mail-extractor-rules"] });
    },
    onError: (error) => setFeedback(getAPIErrorMessage(error, "删除提取规则失败。")),
  });

  const testMutation = useMutation({
    mutationFn: async () =>
      testMailExtractorRule(
        {
          ...draft,
          captureGroupIndex: draft.resultMode === "capture_group" ? Number(draft.captureGroupIndex ?? 1) : undefined,
        },
        selectedMailboxId && sampleMessageId
          ? { mailboxId: Number(selectedMailboxId), messageId: Number(sampleMessageId) }
          : {},
      ),
    onError: (error) => setFeedback(getAPIErrorMessage(error, "测试提取规则失败。")),
  });

  const toggleTemplateMutation = useMutation({
    mutationFn: async (template: MailExtractorRule) => {
      if (template.enabledForUser) {
        return disableMailExtractorTemplate(template.id);
      }
      return enableMailExtractorTemplate(template.id);
    },
    onSuccess: async (_, template) => {
      setFeedback(template.enabledForUser ? "默认模板已停用。" : "默认模板已启用。");
      await queryClient.invalidateQueries({ queryKey: ["mail-extractor-rules"] });
    },
    onError: (error) => setFeedback(getAPIErrorMessage(error, "切换默认模板失败。")),
  });

  const copyTemplateMutation = useMutation({
    mutationFn: async (templateId: number) => copyMailExtractorTemplate(templateId),
    onSuccess: async () => {
      setFeedback("默认模板已复制到我的规则。");
      await queryClient.invalidateQueries({ queryKey: ["mail-extractor-rules"] });
    },
    onError: (error) => setFeedback(getAPIErrorMessage(error, "复制默认模板失败。")),
  });

  function selectRule(rule: MailExtractorRule) {
    setSelectedRuleId(rule.id);
    setDraft(toRuleDraft(rule));
    setFeedback(null);
  }

  function handleSave() {
    const error = validateRuleDraft(draft);
    if (error) {
      setFeedback(error);
      return;
    }
    saveMutation.mutate();
  }

  function handleTest() {
    const error = validateRuleDraft(draft);
    if (error) {
      setFeedback(error);
      return;
    }
    testMutation.mutate();
  }

  return (
    <WorkspacePage>
      <WorkspacePanel
        title="提取规则"
        description="为自己的邮箱维护可视化正则提取器，支持标题、正文、HTML 文本和 Raw 灵活提取。"
        action={
          <Button
            size="sm"
            variant="outline"
            onClick={() => {
              setSelectedRuleId("new");
              setDraft(emptyRuleDraft());
              setFeedback(null);
            }}
          >
            <Plus className="size-4" />
            新建规则
          </Button>
        }
      >
        {feedback ? (
          <div className="rounded-xl border border-border/60 bg-muted/10 px-3 py-2 text-sm text-muted-foreground">
            {feedback}
          </div>
        ) : null}

        <div className="grid gap-4 xl:grid-cols-[0.9fr_1.1fr]">
          <div className="space-y-4">
            <section className="space-y-3">
              <div className="flex items-center gap-2 text-sm font-medium">
                <Wand2 className="size-4" />
                我的规则
              </div>
              {userRules.length ? (
                userRules.map((rule) => (
                  <button
                    key={rule.id}
                    type="button"
                    className="block w-full text-left"
                    onClick={() => selectRule(rule)}
                  >
                    <Card className={selectedRuleId === rule.id ? "border-primary/40 bg-muted/20 shadow-none" : "border-border/60 bg-muted/10 shadow-none"}>
                      <CardContent className="space-y-2 py-4">
                        <div className="flex items-start justify-between gap-3">
                          <div className="space-y-1">
                            <div className="text-sm font-medium">{rule.name}</div>
                            <div className="text-xs text-muted-foreground">{targetSummary(rule) || "未选择字段"}</div>
                          </div>
                          <WorkspaceBadge variant={rule.enabled ? "secondary" : "outline"}>
                            {rule.enabled ? "启用" : "停用"}
                          </WorkspaceBadge>
                        </div>
                        <div className="text-xs text-muted-foreground">{rule.label || "未设置标签"}</div>
                      </CardContent>
                    </Card>
                  </button>
                ))
              ) : (
                <WorkspaceEmpty title="还没有自定义规则" description="从右侧编辑器创建自己的第一条提取规则。" />
              )}
            </section>

            <section className="space-y-3">
              <div className="flex items-center gap-2 text-sm font-medium">
                <Sparkles className="size-4" />
                默认模板
              </div>
              {templates.length ? (
                templates.map((template) => (
                  <Card className="border-border/60 bg-muted/10 shadow-none" key={template.id}>
                    <CardContent className="space-y-3 py-4">
                      <div className="flex items-start justify-between gap-3">
                        <div className="space-y-1">
                          <div className="text-sm font-medium">{template.name}</div>
                          <div className="text-xs text-muted-foreground">{targetSummary(template)}</div>
                        </div>
                        <WorkspaceBadge variant={template.enabledForUser ? "secondary" : "outline"}>
                          {template.enabledForUser ? "已启用" : "未启用"}
                        </WorkspaceBadge>
                      </div>
                      <div className="text-xs text-muted-foreground">{template.description || template.label || "管理员提供的默认提取模板。"}</div>
                      <div className="flex flex-wrap gap-2">
                        <Button size="sm" variant="outline" onClick={() => toggleTemplateMutation.mutate(template)}>
                          {template.enabledForUser ? "停用" : "启用"}
                        </Button>
                        <Button size="sm" variant="outline" onClick={() => copyTemplateMutation.mutate(template.id)}>
                          复制到我的规则
                        </Button>
                      </div>
                    </CardContent>
                  </Card>
                ))
              ) : (
                <WorkspaceEmpty title="暂无默认模板" description="管理员还没有提供默认提取规则模板。" />
              )}
            </section>
          </div>

          <div className="space-y-4">
            <Card className="border-border/60 bg-muted/10 shadow-none">
              <CardContent className="space-y-4 py-4">
                <div className="text-sm font-medium">{selectedRuleId === "new" ? "新建规则" : "编辑规则"}</div>

                <div className="grid gap-3 md:grid-cols-2">
                  <WorkspaceField label="规则名称">
                    <Input aria-label="规则名称" value={draft.name} onChange={(event) => setDraft((current) => ({ ...current, name: event.target.value }))} />
                  </WorkspaceField>
                  <WorkspaceField label="结果标签">
                    <Input aria-label="结果标签" value={draft.label} onChange={(event) => setDraft((current) => ({ ...current, label: event.target.value }))} />
                  </WorkspaceField>
                </div>

                <WorkspaceField label="描述">
                  <Textarea aria-label="规则描述" rows={3} value={draft.description} onChange={(event) => setDraft((current) => ({ ...current, description: event.target.value }))} />
                </WorkspaceField>

                <WorkspaceField label="提取字段">
                  <div className="grid gap-2 md:grid-cols-2">
                    {targetFieldOptions.map((option) => (
                      <label className="flex items-center gap-3 rounded-lg border border-border/60 px-3 py-2 text-sm" key={option.value}>
                        <Checkbox
                          checked={draft.targetFields.includes(option.value)}
                          onCheckedChange={(checked) =>
                            setDraft((current) => ({
                              ...current,
                              targetFields: checked === true
                                ? [...current.targetFields, option.value]
                                : current.targetFields.filter((item) => item !== option.value),
                            }))
                          }
                        />
                        <span>{option.label}</span>
                      </label>
                    ))}
                  </div>
                </WorkspaceField>

                <div className="grid gap-3 md:grid-cols-3">
                  <WorkspaceField label="Flags">
                    <Input aria-label="正则 Flags" value={draft.flags} onChange={(event) => setDraft((current) => ({ ...current, flags: event.target.value }))} placeholder="如 i / im / ims" />
                  </WorkspaceField>
                  <WorkspaceField label="结果模式">
                    <BasicSelect value={draft.resultMode} onChange={(event) => setDraft((current) => ({ ...current, resultMode: event.target.value }))}>
                      {resultModeOptions.map((option) => (
                        <option key={option.value} value={option.value}>
                          {option.label}
                        </option>
                      ))}
                    </BasicSelect>
                  </WorkspaceField>
                  <WorkspaceField label="捕获分组">
                    <Input
                      type="number"
                      aria-label="捕获分组"
                      value={String(draft.captureGroupIndex ?? 1)}
                      onChange={(event) => setDraft((current) => ({ ...current, captureGroupIndex: Number(event.target.value || 0) }))}
                      disabled={draft.resultMode !== "capture_group"}
                    />
                  </WorkspaceField>
                </div>

                <WorkspaceField label="正则表达式">
                  <Textarea aria-label="正则表达式" rows={5} value={draft.pattern} onChange={(event) => setDraft((current) => ({ ...current, pattern: event.target.value }))} placeholder="例如：\\b(\\d{6})\\b" />
                </WorkspaceField>

                <div className="grid gap-3 md:grid-cols-2">
                  <WorkspaceField label="发件人包含">
                    <Input aria-label="发件人包含" value={draft.senderContains} onChange={(event) => setDraft((current) => ({ ...current, senderContains: event.target.value }))} placeholder="仅支持普通文本包含，如 noreply@x.ai" />
                  </WorkspaceField>
                  <WorkspaceField label="标题包含">
                    <Input aria-label="标题包含" value={draft.subjectContains} onChange={(event) => setDraft((current) => ({ ...current, subjectContains: event.target.value }))} placeholder="仅支持普通文本包含，如 verification code" />
                  </WorkspaceField>
                </div>

                <div className="grid gap-3 md:grid-cols-3">
                  <WorkspaceField label="邮箱作用域">
                    <BasicSelect
                      value={draft.mailboxIds[0] ? String(draft.mailboxIds[0]) : ""}
                      onChange={(event) =>
                        setDraft((current) => ({
                          ...current,
                          mailboxIds: event.target.value ? [Number(event.target.value)] : [],
                        }))
                      }
                    >
                      <option value="">全部邮箱</option>
                      {mailboxes.map((mailbox) => (
                        <option key={mailbox.id} value={mailbox.id}>
                          {mailbox.address}
                        </option>
                      ))}
                    </BasicSelect>
                  </WorkspaceField>
                  <WorkspaceField label="域名作用域">
                    <BasicSelect
                      value={draft.domainIds[0] ? String(draft.domainIds[0]) : ""}
                      onChange={(event) =>
                        setDraft((current) => ({
                          ...current,
                          domainIds: event.target.value ? [Number(event.target.value)] : [],
                        }))
                      }
                    >
                      <option value="">全部域名</option>
                      {domains.map((domain) => (
                        <option key={domain.id} value={domain.id}>
                          {domain.domain}
                        </option>
                      ))}
                    </BasicSelect>
                  </WorkspaceField>
                  <WorkspaceField label="排序权重">
                    <Input aria-label="排序权重" type="number" value={String(draft.sortOrder)} onChange={(event) => setDraft((current) => ({ ...current, sortOrder: Number(event.target.value || 0) }))} />
                  </WorkspaceField>
                </div>

                <label className="flex items-center gap-3 rounded-lg border border-border/60 px-3 py-2 text-sm">
                  <Checkbox checked={draft.enabled} onCheckedChange={(checked) => setDraft((current) => ({ ...current, enabled: checked === true }))} />
                  <span>启用此规则</span>
                </label>

                <div className="flex flex-wrap gap-2">
                  <Button onClick={handleSave} disabled={saveMutation.isPending}>
                    {saveMutation.isPending ? "保存中…" : "保存规则"}
                  </Button>
                  {selectedRuleId !== "new" ? (
                    <Button variant="outline" onClick={() => deleteMutation.mutate(Number(selectedRuleId))} disabled={deleteMutation.isPending}>
                      删除规则
                    </Button>
                  ) : null}
                </div>
              </CardContent>
            </Card>

            <Card className="border-border/60 bg-muted/10 shadow-none">
              <CardContent className="space-y-4 py-4">
                <div className="flex items-center justify-between gap-3">
                  <div className="text-sm font-medium">实时测试</div>
                  <Button size="sm" variant="outline" onClick={() => void rulesQuery.refetch()}>
                    <RefreshCw className={`size-4 ${rulesQuery.isFetching ? "animate-spin" : ""}`} />
                    刷新
                  </Button>
                </div>

                <div className="grid gap-3 md:grid-cols-2">
                  <WorkspaceField label="测试邮箱">
                    <BasicSelect value={sampleMailboxId || (selectedMailboxId ? String(selectedMailboxId) : "")} onChange={(event) => {
                      setSampleMailboxId(event.target.value);
                      setSampleMessageId("");
                    }}>
                      {mailboxes.map((mailbox) => (
                        <option key={mailbox.id} value={mailbox.id}>
                          {mailbox.address}
                        </option>
                      ))}
                    </BasicSelect>
                  </WorkspaceField>
                  <WorkspaceField label="测试邮件">
                    <BasicSelect value={sampleMessageId} onChange={(event) => setSampleMessageId(event.target.value)}>
                      <option value="">选择一封邮件</option>
                      {sampleMessages.map((message) => (
                        <option key={message.id} value={message.id}>
                          {(message.subject || "(无主题)").slice(0, 40)}
                        </option>
                      ))}
                    </BasicSelect>
                  </WorkspaceField>
                </div>

                <div className="flex gap-2">
                  <Button onClick={handleTest} disabled={testMutation.isPending || !sampleMessageId}>
                    {testMutation.isPending ? "测试中…" : "运行测试"}
                  </Button>
                </div>

                {testMutation.data?.items.length ? (
                  <div className="space-y-2">
                    {testMutation.data.items.map((item, index) => (
                      <div className="rounded-xl border border-border/60 bg-background/60 px-3 py-3" key={`${item.ruleId}-${index}`}>
                        <div className="flex flex-wrap items-center gap-2">
                          <WorkspaceBadge variant="outline">{item.label || item.ruleName}</WorkspaceBadge>
                          <span className="text-xs text-muted-foreground">{item.sourceField}</span>
                        </div>
                        <div className="mt-2 whitespace-pre-wrap break-all text-sm leading-6">{item.values?.join("\n") || item.value}</div>
                      </div>
                    ))}
                  </div>
                ) : testMutation.isSuccess ? (
                  <WorkspaceEmpty title="没有命中结果" description="当前测试邮件没有匹配到这条规则。" />
                ) : (
                  <WorkspaceEmpty title="选择一封邮件开始测试" description="保存前先对最近收到的邮件跑一次，确认提取结果是否符合预期。" />
                )}
              </CardContent>
            </Card>
          </div>
        </div>
      </WorkspacePanel>
    </WorkspacePage>
  );
}
