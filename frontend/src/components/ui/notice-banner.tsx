import { useEffect, useRef, useState, type ReactNode } from "react";
import { cn } from "@/lib/utils";

export function NoticeBanner({
  children,
  variant = "info",
  className,
  onDismiss,
  autoHideMs,
  pauseOnHover = true,
}: {
  children: ReactNode;
  variant?: "success" | "error" | "info" | "warning";
  className?: string;
  onDismiss?: (() => void) | undefined;
  autoHideMs?: number | undefined;
  pauseOnHover?: boolean;
}) {
  const [isHovered, setIsHovered] = useState(false);
  const dismissRef = useRef(onDismiss);

  useEffect(() => {
    dismissRef.current = onDismiss;
  }, [onDismiss]);

  useEffect(() => {
    if (!autoHideMs || !dismissRef.current || (pauseOnHover && isHovered)) {
      return;
    }

    const timer = window.setTimeout(() => {
      dismissRef.current?.();
    }, autoHideMs);

    return () => {
      window.clearTimeout(timer);
    };
  }, [autoHideMs, isHovered, pauseOnHover]);

  return (
    <div
      onMouseEnter={pauseOnHover ? () => setIsHovered(true) : undefined}
      onMouseLeave={pauseOnHover ? () => setIsHovered(false) : undefined}
      className={cn(
        "rounded-xl border px-3 py-2 text-sm",
        variant === "success" &&
          "border-emerald-500/30 bg-emerald-500/10 text-emerald-700 dark:text-emerald-300",
        variant === "error" &&
          "border-destructive/30 bg-destructive/5 text-destructive",
        variant === "warning" &&
          "border-amber-500/30 bg-amber-500/10 text-amber-900 dark:text-amber-200",
        variant === "info" &&
          "border-border/60 bg-muted/40 text-foreground",
        className,
      )}
    >
      {children}
    </div>
  );
}
