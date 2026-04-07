import type { LucideIcon } from "lucide-react";
import type { ReactNode } from "react";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import {
  Card,
  CardAction,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import { cn } from "@/lib/utils";

export function PublicSection({
  eyebrow,
  title,
  description,
  action,
  children,
  className,
}: {
  eyebrow?: string;
  title: string;
  description?: string;
  action?: ReactNode;
  children: ReactNode;
  className?: string;
}) {
  return (
    <section className={cn("space-y-4", className)}>
      <div className="flex flex-col gap-3 sm:flex-row sm:items-end sm:justify-between">
        <div className="space-y-2">
          {eyebrow ? (
            <Badge className="rounded-full" variant="outline">
              {eyebrow}
            </Badge>
          ) : null}
          <div className="space-y-1">
            <h2 className="text-lg font-semibold tracking-tight sm:text-xl">{title}</h2>
            {description ? <p className="max-w-3xl text-sm leading-6 text-muted-foreground">{description}</p> : null}
          </div>
        </div>
        {action}
      </div>
      {children}
    </section>
  );
}

export function PublicFeatureCard({
  icon: Icon,
  title,
  description,
  eyebrow,
  className,
}: {
  icon?: LucideIcon;
  title: string;
  description: ReactNode;
  eyebrow?: ReactNode;
  className?: string;
}) {
  return (
    <Card className={cn("border-border/60 bg-card shadow-none", className)} size="sm">
      <CardContent className="space-y-3 py-3">
        {Icon ? (
          <div className="flex size-8 items-center justify-center rounded-lg border border-border/60 bg-muted/35 text-muted-foreground">
            <Icon className="size-4" />
          </div>
        ) : null}
        {eyebrow ? <div className="text-[11px] uppercase tracking-[0.18em] text-muted-foreground">{eyebrow}</div> : null}
        <div className="space-y-1">
          <p className="text-sm font-medium">{title}</p>
          <div className="text-[11px] leading-5 text-muted-foreground">{description}</div>
        </div>
      </CardContent>
    </Card>
  );
}

export function PublicInfoCard({
  title,
  description,
  action,
  children,
  className,
}: {
  title: string;
  description?: string;
  action?: ReactNode;
  children?: ReactNode;
  className?: string;
}) {
  return (
    <Card className={cn("border-border/60 bg-card shadow-none", className)} size="sm">
      <CardHeader className="gap-1.5">
        <div>
          <CardTitle className="text-sm">{title}</CardTitle>
          {description ? <CardDescription className="text-xs leading-6">{description}</CardDescription> : null}
        </div>
        {action ? <CardAction>{action}</CardAction> : null}
      </CardHeader>
      {children ? <CardContent className="space-y-4">{children}</CardContent> : null}
    </Card>
  );
}

export function PublicChecklist({
  items,
  marker,
}: {
  items: ReactNode[];
  marker?: "index" | "dot";
}) {
  return (
    <div className="space-y-2">
      {items.map((item, index) => (
        <div className="flex items-start gap-3 rounded-xl border border-border/60 bg-card px-3 py-3" key={index}>
          <span className="mt-0.5 flex size-5 shrink-0 items-center justify-center rounded-full border border-border/60 bg-background text-[11px] text-muted-foreground">
            {marker === "index" ? index + 1 : "•"}
          </span>
          <div className="text-[11px] leading-5 text-muted-foreground">{item}</div>
        </div>
      ))}
    </div>
  );
}

export function PublicStatBadge({
  label,
  value,
}: {
  label: string;
  value: ReactNode;
}) {
  return (
    <div className="rounded-xl border border-border/60 bg-card px-3 py-3">
      <div className="text-[11px] uppercase tracking-[0.18em] text-muted-foreground">{label}</div>
      <div className="mt-1 text-sm font-medium">{value}</div>
    </div>
  );
}

export function PublicLinkButton({
  children,
  className,
  ...props
}: React.ComponentProps<typeof Button>) {
  return (
    <Button className={cn("h-9", className)} size="sm" {...props}>
      {children}
    </Button>
  );
}
