import { Card, CardContent, CardHeader } from "@/components/ui/card";
import { Skeleton } from "@/components/ui/skeleton";

export function RouteLoadingScreen() {
  return (
    <div aria-live="polite" className="flex min-h-[40vh] flex-col gap-4" role="status">
      <Card className="border-border/60 bg-card shadow-none" size="sm">
        <CardHeader className="gap-2">
          <Skeleton className="h-4 w-28" />
          <Skeleton className="h-3 w-72 max-w-full" />
        </CardHeader>
        <CardContent className="grid gap-3 md:grid-cols-2 xl:grid-cols-4">
          {Array.from({ length: 4 }).map((_, index) => (
            <div className="rounded-xl border border-border/60 p-3" key={index}>
              <Skeleton className="h-3 w-20" />
              <Skeleton className="mt-3 h-6 w-16" />
              <Skeleton className="mt-2 h-3 w-24" />
            </div>
          ))}
        </CardContent>
      </Card>

      <div className="grid gap-4 xl:grid-cols-[1.1fr_0.9fr]">
        {Array.from({ length: 2 }).map((_, index) => (
          <Card className="border-border/60 bg-card shadow-none" key={index} size="sm">
            <CardHeader className="gap-2">
              <Skeleton className="h-4 w-24" />
              <Skeleton className="h-3 w-64 max-w-full" />
            </CardHeader>
            <CardContent className="space-y-3">
              {Array.from({ length: 3 }).map((__, rowIndex) => (
                <div className="rounded-xl border border-border/60 p-3" key={rowIndex}>
                  <Skeleton className="h-4 w-32" />
                  <Skeleton className="mt-2 h-3 w-full" />
                </div>
              ))}
            </CardContent>
          </Card>
        ))}
      </div>
    </div>
  );
}
