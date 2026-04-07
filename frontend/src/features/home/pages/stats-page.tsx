import { Activity, Bell, Database, Globe, Layers, ShieldCheck } from "lucide-react";
import { PublicBottomCta, PublicPageHero, PublicShell } from "../components/public-shell";
import { PublicFeatureCard, PublicInfoCard, PublicSection } from "../components/public-ui";

const statViews = [
  {
    title: "域名",
    body: "管理员总览会统计活跃域名，用户控制台则只展示当前账号自己接入的域名。",
    icon: Globe,
  },
  {
    title: "邮箱",
    body: "邮箱总数、活跃邮箱和到期时间均来自实际邮箱数据。",
    icon: Database,
  },
  {
    title: "消息",
    body: "最新消息、正文摘要、附件与原始邮件都基于真实收件数据读取。",
    icon: Activity,
  },
  {
    title: "公告与反馈",
    body: "公告、反馈线程和站内文档都由后台持久化维护，并直接反映在控制台页面。",
    icon: Bell,
  },
  {
    title: "权限",
    body: "不同权限会看到各自可访问的统计范围与控制台数据。",
    icon: ShieldCheck,
  },
  {
    title: "治理",
    body: "管理员后台还提供规则、任务队列、DNS 变更与系统设置等治理维度。",
    icon: Layers,
  },
];

export function StatsPage() {
  return (
    <PublicShell>
      <PublicPageHero
        eyebrow="Stats"
        title="统计说明"
        description="这里汇总系统提供的统计范围与查看位置。"
      />

      <PublicSection description="各项统计均对应实际控制台页面。" title="统计范围">
        <div className="grid gap-4 md:grid-cols-2 xl:grid-cols-3">
          {statViews.map((item) => (
            <PublicFeatureCard description={item.body} icon={item.icon} key={item.title} title={item.title} />
          ))}
        </div>
      </PublicSection>

      <PublicInfoCard title="查看方式">
        <div className="grid gap-3 md:grid-cols-3">
          <div className="rounded-lg border border-border/60 bg-muted/20 px-3 py-3 text-sm leading-6 text-muted-foreground">
            公开页展示统计范围。
          </div>
          <div className="rounded-lg border border-border/60 bg-muted/20 px-3 py-3 text-sm leading-6 text-muted-foreground">
            用户控制台查看账户数据。
          </div>
          <div className="rounded-lg border border-border/60 bg-muted/20 px-3 py-3 text-sm leading-6 text-muted-foreground">
            管理后台查看整站运行状态。
          </div>
        </div>
      </PublicInfoCard>

      <PublicBottomCta />
    </PublicShell>
  );
}
