"use client";
import { useState } from "react";
import { useStrategies } from "@/lib/hooks/queries";
import { STATUS_DEFS } from "@/lib/strategy/status-defs";
import type { StrategyStatus } from "@/lib/api/types";
import { StrategyCard } from "./StrategyCard";

const STATUS_ORDER: StrategyStatus[] = ["live", "shadow", "paper", "backtesting", "draft", "paused", "archived"];

export function StrategiesListContent() {
  const { data } = useStrategies();
  const [active, setActive] = useState<StrategyStatus | null>(null);
  const rows = data ?? [];
  const visible = active ? rows.filter((r) => r.status === active) : rows;

  return (
    <div className="space-y-4">
      <div className="flex items-center justify-between gap-2">
        <h1>Stratejiler</h1>
      </div>
      <div className="flex flex-wrap items-center gap-2">
        {STATUS_ORDER.map((s) => (
          <button
            key={s}
            type="button"
            onClick={() => setActive(active === s ? null : s)}
            className="rounded-md border px-2.5 py-1"
            style={{
              fontSize: 12,
              borderColor: active === s ? STATUS_DEFS[s].color : "var(--border)",
              color: active === s ? STATUS_DEFS[s].color : "inherit",
            }}
          >
            {STATUS_DEFS[s].label}
          </button>
        ))}
        {active && (
          <button type="button" onClick={() => setActive(null)} className="text-muted-foreground" style={{ fontSize: 12 }}>
            Temizle
          </button>
        )}
      </div>
      <div className="grid grid-cols-1 gap-3 md:grid-cols-2 xl:grid-cols-3">
        {visible.map((r) => (
          <StrategyCard key={r.id} row={r} />
        ))}
      </div>
    </div>
  );
}
