import { formatCondition } from "@/lib/strategy/status-defs";
import type { StrategyCondition } from "@/lib/api/types";

export function ConditionList({ title, conditions }: { title: string; conditions: StrategyCondition[] }) {
  return (
    <div className="space-y-2">
      <div className="text-muted-foreground" style={{ fontSize: 12 }}>{title}</div>
      <div className="flex flex-wrap gap-2">
        {conditions.map((c, i) => (
          <span
            key={`${c.metric}-${i}`}
            className="rounded-md border border-border bg-surface-2 px-2 py-1 font-mono"
            style={{ fontSize: 12 }}
          >
            {formatCondition(c)}
          </span>
        ))}
      </div>
    </div>
  );
}
