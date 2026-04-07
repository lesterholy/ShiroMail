import { useQuery } from "@tanstack/react-query";
import { ArrowRight, Coins, Gift, ShieldCheck } from "lucide-react";
import { Link } from "react-router-dom";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Card, CardContent } from "@/components/ui/card";
import {
  WorkspaceEmpty,
  WorkspaceListRow,
  WorkspaceMetric,
  WorkspacePage,
  WorkspacePanel,
} from "@/components/layout/workspace-ui";
import { fetchBalance } from "../api";
import { formatCurrency, formatDateTime } from "./shared";

export function UserRewardsPage() {
  const balanceQuery = useQuery({ queryKey: ["portal-balance"], queryFn: fetchBalance });
  const balance = balanceQuery.data;

  return (
    <WorkspacePage>
      <WorkspacePanel
        action={
          <Badge className="rounded-full" variant="outline">
            <Gift className="mr-1 size-3.5" />
            {balanceQuery.isLoading ? "正在同步奖励数据" : "奖励余额已同步"}
          </Badge>
        }
        description="集中查看可用余额、最近流水和兑换入口，作为当前账户的奖励与余额总览。"
        title="兑换中心"
      >
        <div className="grid gap-4 md:grid-cols-3">
          <WorkspaceMetric
            hint="当前账户所有奖励、赠送和扣减共用这笔可用额度。"
            label="可用余额"
            value={formatCurrency(balance?.balanceCents ?? 0)}
          />
          <WorkspaceMetric
            hint="展示最近产生的奖励与扣减记录，作为兑换前的余额依据。"
            label="最近流水"
            value={balance?.entries.length ?? 0}
          />
          <WorkspaceMetric hint="余额、兑换与套餐资源都在当前控制台内查看和处理。" label="兑换路径" value="统一控制台" />
        </div>
      </WorkspacePanel>

      <WorkspacePanel description="通过余额明细和套餐资源两条路径，快速确认当前账户的兑换依据与可用额度。" title="奖励说明">
        <div className="grid gap-4 md:grid-cols-2">
          <Card className="border-border/60 bg-muted/10 shadow-none">
            <CardContent className="flex items-start justify-between gap-3 py-4">
              <div className="flex gap-3">
                <div className="flex size-9 items-center justify-center rounded-lg border border-border/60 bg-muted/35 text-muted-foreground">
                  <Coins className="size-4" />
                </div>
                <div className="space-y-1">
                  <p className="text-sm font-medium">查看余额明细</p>
                  <p className="text-xs leading-6 text-muted-foreground">跳转到余额页，查看所有增减记录与说明。</p>
                </div>
              </div>
              <Button asChild size="icon-sm" variant="ghost">
                <Link to="/dashboard/balance">
                  <ArrowRight className="size-4" />
                </Link>
              </Button>
            </CardContent>
          </Card>

          <Card className="border-border/60 bg-muted/10 shadow-none">
            <CardContent className="flex items-start justify-between gap-3 py-4">
              <div className="flex gap-3">
                <div className="flex size-9 items-center justify-center rounded-lg border border-border/60 bg-muted/35 text-muted-foreground">
                  <ShieldCheck className="size-4" />
                </div>
                <div className="space-y-1">
                  <p className="text-sm font-medium">查看套餐资源</p>
                  <p className="text-xs leading-6 text-muted-foreground">对照当前订阅计划与资源配额，决定后续兑换方向。</p>
                </div>
              </div>
              <Button asChild size="icon-sm" variant="ghost">
                <Link to="/dashboard/billing">
                  <ArrowRight className="size-4" />
                </Link>
              </Button>
            </CardContent>
          </Card>
        </div>
      </WorkspacePanel>

      <WorkspacePanel description="最近奖励、扣减和调整记录都会直接展示在这里，便于核对余额变化。" title="最近奖励流水">
        {balance?.entries.length ? (
          <div className="space-y-3">
            {balance.entries.map((entry) => (
              <WorkspaceListRow
                description={entry.entryType}
                key={entry.id}
                meta={
                  <>
                    <span className="rounded-full border border-border/60 px-2 py-1">{formatCurrency(entry.amount)}</span>
                    <span>{formatDateTime(entry.createdAt)}</span>
                  </>
                }
                title={entry.description}
              />
            ))}
          </div>
        ) : (
          <WorkspaceEmpty description="当前还没有奖励流水记录。" title="暂无奖励流水" />
        )}
      </WorkspacePanel>
    </WorkspacePage>
  );
}
