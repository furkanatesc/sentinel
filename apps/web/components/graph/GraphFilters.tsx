"use client";
import { X } from "lucide-react";
import type { GraphFilters as GF, GraphEdgeType } from "@/lib/api/types";
import { EMPTY_GRAPH_FILTERS } from "@/lib/api/types";
import { EDGE_TYPE_DEFS } from "@/lib/graph/graph-defs";
import { riskMeta, type RiskLevel } from "@/lib/format";

const RISKS = Object.keys(riskMeta) as RiskLevel[];

export function GraphFilters({ value, onChange }: { value: GF; onChange: (f: GF) => void }) {
  const toggle = <T,>(arr: T[], v: T): T[] => (arr.includes(v) ? arr.filter((x) => x !== v) : [...arr, v]);
  return (
    <div className="space-y-2 rounded-lg border border-border bg-card p-3" style={{ fontSize: 12 }}>
      <div className="flex flex-wrap items-center gap-1.5">
        <span className="text-muted-foreground">İlişki:</span>
        {EDGE_TYPE_DEFS.map((d) => {
          const on = value.relationships.includes(d.key);
          return <button key={d.key} onClick={() => onChange({ ...value, relationships: toggle(value.relationships, d.key) })}
            className={`rounded px-2 py-1 ${on ? "bg-primary text-primary-foreground" : "bg-surface-2 text-muted-foreground hover:text-foreground"}`}>{d.label}</button>;
        })}
      </div>
      <div className="flex flex-wrap items-center gap-1.5">
        <span className="text-muted-foreground">Risk:</span>
        {RISKS.map((r) => {
          const on = value.risks.includes(r); const m = riskMeta[r];
          return <button key={r} onClick={() => onChange({ ...value, risks: toggle(value.risks, r) })}
            className="rounded px-2 py-1" style={{ color: on ? m.color : undefined, backgroundColor: on ? m.bg : "var(--sentinel-surface-2)", border: on ? `1px solid ${m.border}` : "1px solid transparent" }}>{m.label}</button>;
        })}
        <button onClick={() => onChange(EMPTY_GRAPH_FILTERS)} className="ml-auto flex items-center gap-1 rounded-md border border-border px-2 py-1 text-muted-foreground hover:text-foreground"><X size={12} /> Temizle</button>
      </div>
    </div>
  );
}
