import { MetricTile } from "@/components/sentinel/MetricTile";
import { pnlColor } from "@/lib/position/risk-filter";
import { BACKTEST_METRIC_DEFS } from "@/lib/backtest/backtest-defs";
import type { BacktestMetrics as Metrics } from "@/lib/api/types";

function fmt(kind: "pnl" | "pct" | "num", v: number): string {
  if (kind === "pct") return `%${v}`;
  if (kind === "pnl") return `${v} SOL`;
  return `${v}`;
}

export function BacktestMetrics({ metrics }: { metrics: Metrics }) {
  return (
    <div className="grid grid-cols-2 gap-3 md:grid-cols-5">
      {BACKTEST_METRIC_DEFS.map((d) => {
        const v = metrics[d.key];
        return <MetricTile key={d.key} label={d.label} value={fmt(d.kind, v)} valueColor={d.kind === "pnl" ? pnlColor(v) : undefined} />;
      })}
    </div>
  );
}
