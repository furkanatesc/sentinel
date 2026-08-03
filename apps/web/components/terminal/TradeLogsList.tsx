"use client";
import { useTradeLogs } from "@/lib/hooks/queries";
import { Skeleton } from "@/components/ui/skeleton";

const LEVEL_COLOR: Record<string, string> = { info: "#8A94A6", warn: "#FFB020", error: "#F0476B" };

export function TradeLogsList() {
  const { data, isLoading } = useTradeLogs();
  if (isLoading || !data) return <Skeleton className="h-40 w-full" />;
  return (
    <ul className="divide-y divide-border font-mono" style={{ fontSize: 12 }}>
      {data.map((l) => (
        <li key={l.id} className="flex items-center gap-3 px-3 py-1.5">
          <span className="uppercase" style={{ color: LEVEL_COLOR[l.level], width: 44 }}>{l.level}</span>
          <span className="flex-1">{l.message}</span>
          <span className="text-muted-foreground">{l.time}</span>
        </li>
      ))}
    </ul>
  );
}
