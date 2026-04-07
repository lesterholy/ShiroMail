import { useQuery } from "@tanstack/react-query";
import {
  WorkspaceEmpty,
  WorkspaceListRow,
  WorkspacePage,
  WorkspacePanel,
} from "@/components/layout/workspace-ui";
import { fetchAdminJobs } from "../api";
import { formatDateTime } from "../../user/pages/shared";

export function AdminJobsPage() {
  const jobsQuery = useQuery({ queryKey: ["admin-jobs"], queryFn: fetchAdminJobs });
  const jobs = jobsQuery.data ?? [];

  return (
    <WorkspacePage>
      <WorkspacePanel description="查看任务状态与最近异常信息。" title="任务队列">
        {jobs.length ? (
          <div className="space-y-3">
            {jobs.map((item) => (
              <WorkspaceListRow
                description={item.errorMessage || "无异常"}
                key={item.id}
                meta={
                  <>
                    <span className="rounded-full border border-border/60 px-2 py-1">{item.status}</span>
                    <span>{formatDateTime(item.createdAt)}</span>
                  </>
                }
                title={item.jobType}
              />
            ))}
          </div>
        ) : (
          <WorkspaceEmpty description="任务队列当前没有待观察记录。" title="暂无任务记录" />
        )}
      </WorkspacePanel>
    </WorkspacePage>
  );
}
