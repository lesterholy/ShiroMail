import { useQuery } from "@tanstack/react-query";
import { Card, CardContent } from "@/components/ui/card";
import { WorkspaceMetric, WorkspacePage, WorkspacePanel } from "@/components/layout/workspace-ui";
import { fetchBilling } from "../api";
import { formatDateTime } from "./shared";

export function UserBillingPage() {
  const billingQuery = useQuery({ queryKey: ["portal-billing"], queryFn: fetchBilling });
  const billing = billingQuery.data;

  return (
    <WorkspacePage>
      <WorkspacePanel description="当前订阅计划、资源配额与续费时间。" title="套餐订阅">
        <div className="grid gap-4 md:grid-cols-3">
          <WorkspaceMetric hint={billing?.planCode ?? "—"} label="当前计划" value={billing?.planName ?? "—"} />
          <WorkspaceMetric hint="可同时保有的邮箱数量" label="邮箱配额" value={billing?.mailboxQuota ?? 0} />
          <WorkspaceMetric hint="API / webhook 总额度" label="每日请求" value={billing?.dailyRequestLimit ?? 0} />
        </div>

        <Card className="border-border/60 bg-muted/10 shadow-none">
          <CardContent className="flex flex-wrap items-center gap-2 py-4 text-xs text-muted-foreground">
            <span className="rounded-full border border-border/60 px-2 py-1">状态：{billing?.status ?? "—"}</span>
            <span>续费时间：{formatDateTime(billing?.renewalAt)}</span>
            <span>域名配额：{billing?.domainQuota ?? 0}</span>
          </CardContent>
        </Card>
      </WorkspacePanel>
    </WorkspacePage>
  );
}
