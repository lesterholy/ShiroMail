export type ApiEndpointReference = {
  method: "GET" | "POST" | "PUT" | "PATCH" | "DELETE";
  path: string;
  auth: string;
  description: string;
};

export type ApiReferenceSection = {
  title: string;
  description: string;
  endpoints: ApiEndpointReference[];
};

export type DocsJSONExample = {
  title: string;
  description: string;
  payload: string;
};

export type DocsFieldGuide = {
  name: string;
  meaning: string;
};

export const apiReferenceSections: ApiReferenceSection[] = [
  {
    title: "认证与账户",
    description: "支持账号登录、OAuth、邮箱验证、密码重置与 TOTP 两步验证。",
    endpoints: [
      { method: "POST", path: "/api/v1/auth/register", auth: "公开", description: "创建账号，是否开放注册由系统设置控制。" },
      { method: "POST", path: "/api/v1/auth/login", auth: "公开", description: "账号密码登录，必要时进入 TOTP 二次校验。" },
      { method: "POST", path: "/api/v1/auth/oauth/:provider/start", auth: "公开", description: "发起 OAuth 登录。" },
      { method: "POST", path: "/api/v1/auth/oauth/:provider/callback", auth: "公开", description: "完成 OAuth 回调换取会话。" },
      { method: "POST", path: "/api/v1/auth/forgot-password", auth: "公开", description: "向账户邮箱发送一次性验证码。" },
      { method: "POST", path: "/api/v1/auth/reset-password", auth: "公开", description: "使用验证码完成密码重置。" },
      { method: "GET", path: "/api/v1/account/profile", auth: "用户", description: "读取当前账户资料与安全状态。" },
      { method: "PATCH", path: "/api/v1/account/profile", auth: "用户", description: "更新显示名称、时区和语言。" },
    ],
  },
  {
    title: "域名与 DNS",
    description: "支持根域名接入、子域名批量生成、DNS 服务商绑定、验证与变更预览。",
    endpoints: [
      { method: "GET", path: "/api/v1/domains", auth: "用户 / API Key", description: "列出当前会话可访问的域名。" },
      { method: "POST", path: "/api/v1/domains", auth: "用户 / API Key", description: "新增当前账号自己的根域名。" },
      { method: "POST", path: "/api/v1/domains/generate", auth: "用户 / API Key", description: "基于根域名批量生成子域名。" },
      { method: "PUT", path: "/api/v1/domains/:id/provider-binding", auth: "用户", description: "为域名绑定或解绑 DNS 服务商账号。" },
      { method: "POST", path: "/api/v1/domains/:id/verify", auth: "用户", description: "检查 DNS 记录传播并更新验证状态。" },
      { method: "GET", path: "/api/v1/portal/domain-providers/:id/zones/:zoneId/records", auth: "用户", description: "读取服务商 Zone 当前记录。" },
      { method: "POST", path: "/api/v1/portal/domain-providers/:id/zones/:zoneId/change-sets/preview", auth: "用户", description: "预览 DNS 变更集，不直接落盘。" },
      { method: "POST", path: "/api/v1/portal/dns-change-sets/:changeSetId/apply", auth: "用户", description: "将预览好的 DNS 变更真正应用到服务商。" },
    ],
  },
  {
    title: "邮箱与消息",
    description: "围绕临时邮箱生命周期、消息列表、正文、附件和 EML 原文下载提供接口。",
    endpoints: [
      { method: "GET", path: "/api/v1/dashboard", auth: "用户 / API Key", description: "返回当前账号仪表盘汇总、域名与邮箱数据。" },
      { method: "GET", path: "/api/v1/mailboxes", auth: "用户 / API Key", description: "列出当前账号邮箱。" },
      { method: "POST", path: "/api/v1/mailboxes", auth: "用户 / API Key", description: "创建临时邮箱，可指定 domainId 与 localPart。" },
      { method: "POST", path: "/api/v1/mailboxes/:id/extend", auth: "用户 / API Key", description: "延长邮箱有效期。" },
      { method: "POST", path: "/api/v1/mailboxes/:id/release", auth: "用户 / API Key", description: "释放邮箱并结束继续收件。" },
      { method: "GET", path: "/api/v1/mailboxes/:mailboxId/messages", auth: "用户 / API Key", description: "读取消息列表。" },
      { method: "GET", path: "/api/v1/mailboxes/:mailboxId/messages/:id", auth: "用户 / API Key", description: "读取消息正文、头部与附件元数据。" },
      { method: "GET", path: "/api/v1/mailboxes/:mailboxId/messages/:id/extractions", auth: "用户 / API Key", description: "返回该邮件按当前提取规则命中的验证码、链接或自定义字段。" },
      { method: "GET", path: "/api/v1/mailboxes/:mailboxId/messages/:id/raw", auth: "用户 / API Key", description: "下载原始 EML。" },
      { method: "GET", path: "/api/v1/mailboxes/:mailboxId/messages/:id/raw/parsed", auth: "用户 / API Key", description: "解析原始 EML 并返回结构化正文、头部与附件信息。" },
      { method: "POST", path: "/api/v1/mailboxes/:mailboxId/messages/receive", auth: "用户 / API Key", description: "向指定邮箱注入 RFC822 原文，走完整收件入库链路。" },
    ],
  },
  {
    title: "SMTP 投递与诊断",
    description: "覆盖后台 SMTP 测试发信、入站 spool 队列、拒收原因与实时指标，便于排查真实收发链路。",
    endpoints: [
      { method: "POST", path: "/api/v1/admin/configs/mail.delivery/test", auth: "管理员", description: "发送一封 SMTP 测试邮件；失败时会返回 stage、code、hint、retryable 等结构化诊断字段。" },
      { method: "GET", path: "/api/v1/admin/jobs/inbound-spool", auth: "管理员", description: "分页读取入站 spool 队列、状态汇总和失败原因聚合。" },
      { method: "POST", path: "/api/v1/admin/jobs/inbound-spool/:id/retry", auth: "管理员", description: "将单条 failed spool 项重新放回待消费队列。" },
      { method: "GET", path: "/api/v1/admin/jobs/smtp-metrics", auth: "管理员", description: "读取 SMTP 会话、拒收、accepted、spool worker 的实时内存快照。" },
    ],
  },
  {
    title: "API Key、Webhook 与控制台补充接口",
    description: "已登录用户可直接在控制台管理 API Key、Webhook、文档、余额与设置。",
    endpoints: [
      { method: "GET", path: "/api/v1/portal/api-keys", auth: "用户", description: "读取当前账号自己的 API Key。" },
      { method: "POST", path: "/api/v1/portal/api-keys", auth: "用户", description: "创建带 scope 与域名绑定策略的 API Key。" },
      { method: "GET", path: "/api/v1/portal/webhooks", auth: "用户", description: "读取当前账号 Webhook。" },
      { method: "POST", path: "/api/v1/portal/webhooks", auth: "用户", description: "新增 Webhook 目标与事件。" },
      { method: "GET", path: "/api/v1/portal/docs", auth: "用户", description: "读取站内文档条目。" },
      { method: "GET", path: "/api/v1/portal/billing", auth: "用户", description: "读取套餐、域名配额与请求限制。" },
      { method: "GET", path: "/api/v1/portal/balance", auth: "用户", description: "读取余额与流水。" },
      { method: "GET", path: "/api/v1/admin/overview", auth: "管理员", description: "读取管理员控制台总览。" },
    ],
  },
];

export const runtimeCapabilities = [
  "账户体系支持账号密码、OAuth 与 TOTP 两步验证。",
  "域名管理支持根域名、子域名、DNS 服务商、校验与变更应用。",
  "邮箱链路支持创建、续期、释放、消息正文、附件和 EML 下载。",
  "SMTP 后台支持测试发信、结构化诊断、reject 统计与 inbound spool 观察。",
  "控制台支持 API Key、Webhook、文档、余额、公告和管理员治理接口。",
];

export const smtpDiagnosticExamples: DocsJSONExample[] = [
  {
    title: "SMTP test failure payload",
    description: "管理员在“账户邮件发信”里点测试发信失败时，接口会直接返回结构化诊断字段。",
    payload: `{
  "message": "mail delivery TLS handshake failed: server does not advertise STARTTLS",
  "stage": "tls",
  "code": "starttls_unavailable",
  "hint": "The upstream server does not advertise STARTTLS. Switch to SMTPS or plain mode only if your provider explicitly supports it.",
  "retryable": false
}`,
  },
  {
    title: "Inbound spool item with diagnostic",
    description: "任务队列和失败原因聚合现在都会带同一套可读诊断结构，前端无需再自己猜错误文案。",
    payload: `{
  "id": 18,
  "mailFrom": "sender@example.com",
  "status": "failed",
  "errorMessage": "temporary parse failure",
  "diagnostic": {
    "code": "temporary_parse_failure",
    "title": "Temporary Parse Failure",
    "description": "The worker failed while parsing MIME content or message structure. This is often retryable after transient input or runtime issues clear.",
    "retryable": true
  }
}`,
  },
  {
    title: "SMTP metrics rejectedDetails",
    description: "实时指标除了原始 reject 计数，也会返回按原因展开后的标题、说明与 retryable 元数据。",
    payload: `{
  "sessionsStarted": 14,
  "recipientsAccepted": 21,
  "rejected": {
    "attachment_too_large": 2
  },
  "rejectedDetails": [
    {
      "key": "attachment_too_large",
      "count": 2,
      "diagnostic": {
        "code": "attachment_too_large",
        "title": "Attachment Too Large",
        "description": "The message was rejected because at least one attachment exceeded the active inbound size limit.",
        "retryable": false
      }
    }
  ]
}`,
  },
];

export const smtpDiagnosticFieldGuides: DocsFieldGuide[] = [
  {
    name: "diagnostic",
    meaning: "Human-readable diagnostic object attached to failed spool items and failed background jobs. Includes `title`, `description`, and `retryable`.",
  },
  {
    name: "failureMode",
    meaning: "Query filter for `/api/v1/admin/jobs/inbound-spool`. Supports `all`, `retryable`, and `non_retryable` to narrow failed spool items.",
  },
  {
    name: "rejectedDetails",
    meaning: "Expanded view of SMTP reject counters. Each entry keeps the raw key and count, plus a normalized diagnostic payload for UI or automation.",
  },
  {
    name: "retryable",
    meaning: "Boolean hint for operators and clients. `true` means a retry may succeed after transient conditions clear; `false` usually means the config or target must be fixed first.",
  },
  {
    name: "stage / code / hint",
    meaning: "Structured fields returned by SMTP test delivery failures. `stage` shows where the failure occurred, `code` is stable for logic, and `hint` is the operator-facing remediation note.",
  },
];
