import { useQuery } from "@tanstack/react-query";
import { WorkspaceBadge, WorkspaceEmpty, WorkspacePage, WorkspacePanel } from "@/components/layout/workspace-ui";
import { Card, CardContent } from "@/components/ui/card";
import { fetchNotices } from "../api";
import { formatDateTime } from "./shared";

export function UserNoticesPage() {
  const noticesQuery = useQuery({ queryKey: ["portal-notices"], queryFn: fetchNotices });

  return (
    <WorkspacePage>
      <WorkspacePanel description="平台维护、版本更新和投递策略变更都会在这里同步。" title="公告">
        {noticesQuery.data?.length ? (
          <div className="space-y-3">
            {noticesQuery.data.map((notice) => (
              <Card className="border-border/60 bg-card/85 shadow-none" key={notice.id}>
                <CardContent className="space-y-3 py-4">
                  <div className="flex flex-wrap items-center justify-between gap-2">
                    <WorkspaceBadge>{notice.category}</WorkspaceBadge>
                    <span className="text-xs text-muted-foreground">{formatDateTime(notice.publishedAt)}</span>
                  </div>
                  <div className="text-sm font-medium">{notice.title}</div>
                  <p className="text-sm leading-6 text-muted-foreground">{notice.body}</p>
                </CardContent>
              </Card>
            ))}
          </div>
        ) : (
          <WorkspaceEmpty description="当前还没有发布内容，后续平台更新会出现在这里。" title="暂无公告" />
        )}
      </WorkspacePanel>
    </WorkspacePage>
  );
}
