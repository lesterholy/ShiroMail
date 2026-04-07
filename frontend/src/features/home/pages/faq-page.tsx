import { PublicBottomCta, PublicPageHero, PublicShell } from "../components/public-shell";
import { PublicFeatureCard, PublicSection } from "../components/public-ui";

const faqItems = [
  {
    title: "支持哪些账户入口？",
    body: "支持账号密码登录、OAuth 登录、邮箱验证、密码重置与 TOTP 两步验证。",
  },
  {
    title: "现在的域名能力包含哪些？",
    body: "支持根域名接入、子域名批量生成、DNS 服务商绑定、校验连接、记录预览与变更应用。",
  },
  {
    title: "临时邮箱包含哪些操作？",
    body: "支持邮箱创建、续期、释放、消息列表、正文、附件与 EML 原文下载。",
  },
  {
    title: "API Key 可以限制权限和域名吗？",
    body: "可以。API Key 支持 scope、域名绑定和资源策略，实际请求会按 scope 与域名约束执行。",
  },
  {
    title: "Webhook 现在能做什么？",
    body: "当前用户可以在控制台创建、更新、启停 Webhook，并接收消息事件回调。",
  },
  {
    title: "公开文档和站内文档有什么区别？",
    body: "公开文档说明实际能力边界与接口分组；站内文档中心则给已登录用户和管理员提供操作说明与维护内容。",
  },
];

export function FaqPage() {
  return (
    <PublicShell>
      <PublicPageHero
        eyebrow="FAQ"
        title="常见问题"
        description="这里汇总部署、接入和使用中的常见问题。"
      />

      <PublicSection description="以下内容对应当前可用功能。" title="常见问题">
        <div className="grid gap-4 md:grid-cols-2 xl:grid-cols-3">
          {faqItems.map((item) => (
            <PublicFeatureCard description={item.body} key={item.title} title={item.title} />
          ))}
        </div>
      </PublicSection>

      <PublicBottomCta />
    </PublicShell>
  );
}
