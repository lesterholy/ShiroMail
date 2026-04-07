import type { MailExtractorRule } from "./api";

export type RuleDraft = Omit<
  MailExtractorRule,
  "id" | "sourceType" | "enabledForUser" | "createdAt" | "updatedAt" | "ownerUserId" | "templateKey"
>;

export function emptyRuleDraft(): RuleDraft {
  return {
    name: "",
    description: "",
    label: "",
    enabled: true,
    targetFields: ["subject"],
    pattern: "",
    flags: "i",
    resultMode: "capture_group",
    captureGroupIndex: 1,
    mailboxIds: [],
    domainIds: [],
    senderContains: "",
    subjectContains: "",
    sortOrder: 100,
  };
}

export function normalizeMailExtractorRule(rule: Partial<MailExtractorRule>): MailExtractorRule {
  const targetFields = Array.isArray(rule.targetFields)
    ? rule.targetFields.filter((field): field is string => typeof field === "string" && field.length > 0)
    : [];
  const mailboxIds = Array.isArray(rule.mailboxIds)
    ? rule.mailboxIds.filter((id): id is number => typeof id === "number" && Number.isFinite(id))
    : [];
  const domainIds = Array.isArray(rule.domainIds)
    ? rule.domainIds.filter((id): id is number => typeof id === "number" && Number.isFinite(id))
    : [];

  return {
    id: Number(rule.id ?? 0),
    ownerUserId: typeof rule.ownerUserId === "number" ? rule.ownerUserId : undefined,
    sourceType: typeof rule.sourceType === "string" ? rule.sourceType : "user",
    templateKey: typeof rule.templateKey === "string" ? rule.templateKey : undefined,
    name: typeof rule.name === "string" ? rule.name : "",
    description: typeof rule.description === "string" ? rule.description : "",
    label: typeof rule.label === "string" ? rule.label : "",
    enabled: rule.enabled !== false,
    enabledForUser: typeof rule.enabledForUser === "boolean" ? rule.enabledForUser : undefined,
    targetFields: targetFields.length ? targetFields : ["subject"],
    pattern: typeof rule.pattern === "string" ? rule.pattern : "",
    flags: typeof rule.flags === "string" ? rule.flags : "i",
    resultMode: typeof rule.resultMode === "string" ? rule.resultMode : "capture_group",
    captureGroupIndex:
      typeof rule.captureGroupIndex === "number" && Number.isFinite(rule.captureGroupIndex)
        ? rule.captureGroupIndex
        : 1,
    mailboxIds,
    domainIds,
    senderContains: typeof rule.senderContains === "string" ? rule.senderContains : "",
    subjectContains: typeof rule.subjectContains === "string" ? rule.subjectContains : "",
    sortOrder: typeof rule.sortOrder === "number" && Number.isFinite(rule.sortOrder) ? rule.sortOrder : 100,
    createdAt: typeof rule.createdAt === "string" ? rule.createdAt : undefined,
    updatedAt: typeof rule.updatedAt === "string" ? rule.updatedAt : undefined,
  };
}

export function toRuleDraft(rule: Partial<MailExtractorRule>): RuleDraft {
  const normalized = normalizeMailExtractorRule(rule);
  return {
    name: normalized.name,
    description: normalized.description,
    label: normalized.label,
    enabled: normalized.enabled,
    targetFields: normalized.targetFields,
    pattern: normalized.pattern,
    flags: normalized.flags,
    resultMode: normalized.resultMode,
    captureGroupIndex: normalized.captureGroupIndex ?? 1,
    mailboxIds: normalized.mailboxIds,
    domainIds: normalized.domainIds,
    senderContains: normalized.senderContains,
    subjectContains: normalized.subjectContains,
    sortOrder: normalized.sortOrder,
  };
}

export function validateRuleDraft(draft: RuleDraft): string | null {
  if (!draft.name.trim()) {
    return "请输入规则名称。";
  }
  if (!draft.pattern.trim()) {
    return "请输入正则表达式。";
  }
  if (!Array.isArray(draft.targetFields) || draft.targetFields.length === 0) {
    return "至少选择一个提取字段。";
  }
  if (!/^[ims]*$/i.test(draft.flags.trim())) {
    return "Flags 只支持 i、m、s。";
  }
  if (draft.resultMode === "capture_group") {
    const captureGroupIndex = Number(draft.captureGroupIndex ?? 1);
    if (!Number.isInteger(captureGroupIndex) || captureGroupIndex < 0) {
      return "捕获分组必须是大于等于 0 的整数。";
    }
  }
  try {
    // 前端先行验证，避免把非法正则提交到后端。
    // 支持的 flags 与后端保持一致。
    new RegExp(draft.pattern, draft.flags.trim());
  } catch (error) {
    return `正则表达式无效：${error instanceof Error ? error.message : "请检查写法"}`;
  }
  if (looksLikeRegex(draft.senderContains)) {
    return "“发件人包含”只支持普通文本包含，不支持正则，请把正则写到主表达式里。";
  }
  if (looksLikeRegex(draft.subjectContains)) {
    return "“标题包含”只支持普通文本包含，不支持正则，请把正则写到主表达式里。";
  }
  return null;
}

function looksLikeRegex(value: string) {
  const trimmed = value.trim();
  if (!trimmed) {
    return false;
  }
  return /[\\^$.*+?()[\]{}|]/.test(trimmed);
}
