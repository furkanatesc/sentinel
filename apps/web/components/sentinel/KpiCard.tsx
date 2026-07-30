import { ArrowUpRight, ArrowDownRight, Minus } from "lucide-react";
import { Sparkline } from "./Sparkline";
import type { Kpi } from "@/lib/api/types";

const toneColor: Record<NonNullable<Kpi["tone"]> | "default", string> = {
  positive: "#2FD98B", critical: "#F0476B", warning: "#FFB020", neutral: "#8A94A6", default: "#7C5CFF",
};

export function KpiCard({ kpi }: { kpi: Kpi }) {
  const color = toneColor[kpi.tone ?? "default"];
  const up = kpi.change > 0;
  const flat = kpi.change === 0;
  const changeColor = flat ? "#8A94A6" : up ? "#2FD98B" : "#F0476B";
  const ChangeIcon = flat ? Minus : up ? ArrowUpRight : ArrowDownRight;
  return (
    <div className="group rounded-lg border border-border bg-card p-4 transition-colors hover:border-white/15" title={`${kpi.label} — updated ${kpi.updated}`}>
      <div className="flex items-start justify-between gap-2">
        <span className="text-muted-foreground" style={{ fontSize: 12 }}>{kpi.label}</span>
        <span className="inline-flex items-center gap-0.5 font-mono tabular-nums" style={{ color: changeColor, fontSize: 11 }}>
          <ChangeIcon size={12} />{flat ? "0.0" : Math.abs(kpi.change).toFixed(1)}%
        </span>
      </div>
      <div className="mt-2 flex items-end justify-between gap-2">
        <span className="font-mono tabular-nums" style={{ fontSize: 22, fontWeight: 600, color }}>{kpi.value}</span>
        <Sparkline data={kpi.spark} color={color} />
      </div>
      <div className="mt-1.5 text-muted-foreground" style={{ fontSize: 10 }}>Updated {kpi.updated}</div>
    </div>
  );
}
