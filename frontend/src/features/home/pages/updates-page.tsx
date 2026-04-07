import { ArrowUpRight, CalendarDays } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Link } from "react-router-dom";
import { PublicBottomCta, PublicPageHero, PublicShell } from "../components/public-shell";
import { PublicSection } from "../components/public-ui";

const updateItems = [
  {
    title: "账户与登录能力已接通",
    category: "Auth",
    date: "2026-04-02",
    body: "账号登录、OAuth、邮箱验证、密码重置与 TOTP 两步验证现已可用。",
  },
  {
    title: "域名管理与 DNS 配置链路已接通",
    category: "Domain",
    date: "2026-04-03",
    body: "根域名、子域名、DNS 服务商绑定、记录预览、变更应用与传播校验现已可用。",
  },
  {
    title: "邮箱与消息操作已切到真实接口",
    category: "Mailbox",
    date: "2026-04-04",
    body: "邮箱创建、续期、释放、消息详情、附件下载和 EML 原文下载均已接入真实接口。",
  },
  {
    title: "账户安全能力已补齐到邮箱验证与 TOTP",
    category: "Security",
    date: "2026-04-05",
    body: "注册验证、重置密码、变更绑定邮箱和两步验证已接入邮件发送链路。",
  },
];

export function UpdatesPage() {
  return (
    <PublicShell>
      <PublicPageHero
        eyebrow="Updates"
        title="功能更新"
        description="这里汇总当前版本的主要更新。"
      />

      <PublicSection description="每条更新都对应当前代码里的实际能力。" title="最近更新">
        <div className="space-y-4">
          {updateItems.map((item) => (
            <Card className="border-border/60 bg-card/92 shadow-none" key={item.title}>
              <CardHeader className="gap-3">
                <div className="flex flex-wrap items-center justify-between gap-2 text-xs text-muted-foreground">
                  <span className="rounded-full border border-border/60 px-2 py-1">{item.category}</span>
                  <span className="inline-flex items-center gap-1.5">
                    <CalendarDays className="size-3.5" />
                    {item.date}
                  </span>
                </div>
                <CardTitle className="text-base">{item.title}</CardTitle>
              </CardHeader>
              <CardContent className="flex flex-wrap items-center justify-between gap-3">
                <p className="max-w-3xl text-sm leading-6 text-muted-foreground">{item.body}</p>
                <Button asChild size="sm" variant="ghost">
                  <Link to="/docs">
                    查看文档
                    <ArrowUpRight className="size-4" />
                  </Link>
                </Button>
              </CardContent>
            </Card>
          ))}
        </div>
      </PublicSection>

      <PublicBottomCta />
    </PublicShell>
  );
}
