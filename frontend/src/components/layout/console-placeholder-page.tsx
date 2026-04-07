import { Button } from "@/components/ui/button";
import { WorkspaceEmpty, WorkspacePanel } from "@/components/layout/workspace-ui";
import { ArrowRight } from "lucide-react";

type ConsolePlaceholderPageProps = {
  eyebrow: string;
  title: string;
  description: string;
  primaryAction: string;
};

export function ConsolePlaceholderPage({
  eyebrow,
  title,
  description,
  primaryAction,
}: ConsolePlaceholderPageProps) {
  return (
    <WorkspacePanel description={eyebrow} title={title}>
      <WorkspaceEmpty
        action={
          <Button size="sm">
            <span>{primaryAction}</span>
            <ArrowRight className="size-4" />
          </Button>
        }
        description={description}
        title="该模块已经预留入口"
      />
    </WorkspacePanel>
  );
}
