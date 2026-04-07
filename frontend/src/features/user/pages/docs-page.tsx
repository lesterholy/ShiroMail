import { useQuery } from "@tanstack/react-query";
import { Card, CardContent } from "@/components/ui/card";
import { WorkspaceBadge, WorkspaceEmpty, WorkspacePage, WorkspacePanel } from "@/components/layout/workspace-ui";
import { apiReferenceSections } from "../../home/docs-reference";
import { fetchDocs } from "../api";

export function UserDocsPage() {
  const docsQuery = useQuery({ queryKey: ["portal-docs"], queryFn: fetchDocs });

  return (
    <WorkspacePage>
      <WorkspacePanel description="这里同时展示站内文档条目与当前程序已经开放的核心 API 分组。" title="文档中心">
        <div className="space-y-4">
          {apiReferenceSections.map((section) => (
            <Card className="border-border/60 bg-muted/10 shadow-none" key={section.title}>
              <CardContent className="space-y-3 py-4">
                <div className="space-y-1">
                  <div className="text-sm font-medium">{section.title}</div>
                  <p className="text-sm leading-6 text-muted-foreground">{section.description}</p>
                </div>
                <div className="grid gap-2">
                  {section.endpoints.map((endpoint) => (
                    <div className="rounded-lg border border-border/60 bg-card/80 px-3 py-2.5" key={`${endpoint.method}-${endpoint.path}`}>
                      <div className="flex flex-wrap items-center gap-2">
                        <WorkspaceBadge>{endpoint.method}</WorkspaceBadge>
                        <WorkspaceBadge variant="outline">{endpoint.auth}</WorkspaceBadge>
                        <span className="font-mono text-xs text-muted-foreground">{endpoint.path}</span>
                      </div>
                      <p className="mt-2 text-sm leading-6 text-muted-foreground">{endpoint.description}</p>
                    </div>
                  ))}
                </div>
              </CardContent>
            </Card>
          ))}
        </div>

        {docsQuery.data?.length ? (
          <div className="grid gap-4 lg:grid-cols-2">
            {docsQuery.data.map((doc) => (
              <Card className="border-border/60 bg-card/85 shadow-none" key={doc.id}>
                <CardContent className="space-y-3 py-4">
                  <WorkspaceBadge>{doc.category}</WorkspaceBadge>
                  <div className="text-sm font-medium">{doc.title}</div>
                  <p className="text-sm leading-6 text-muted-foreground">{doc.summary}</p>
                  <div className="flex flex-wrap gap-2">
                    {doc.tags.map((tag) => (
                      <WorkspaceBadge key={tag} variant="outline">
                        {tag}
                      </WorkspaceBadge>
                    ))}
                  </div>
                </CardContent>
              </Card>
            ))}
          </div>
        ) : (
          <WorkspaceEmpty description="后续接入说明会显示在这里。" title="暂无文档内容" />
        )}
      </WorkspacePanel>
    </WorkspacePage>
  );
}
