"use client";
import Link from "next/link";
import { usePositions } from "@/lib/hooks/queries";
import { pnlColor } from "@/lib/position/risk-filter";
import { riskMeta } from "@/lib/format";

export function OpenPositionsSummary() {
  const { data } = usePositions();
  const rows = (data ?? []).slice(0, 5);
  return (
    <div className="rounded-lg border border-border bg-card p-4">
      <div className="mb-3 flex items-center justify-between">
        <span className="font-medium" style={{ fontSize: 13 }}>Açık Pozisyonlar</span>
        <Link href="/positions" className="text-primary" style={{ fontSize: 12 }}>Tümü →</Link>
      </div>
      <ul className="space-y-2">
        {rows.map((p) => (
          <li key={p.id} className="flex items-center justify-between" style={{ fontSize: 12 }}>
            <span className="flex items-center gap-2">
              <span className="font-mono">{p.tokenSymbol}</span>
              <span className="text-muted-foreground">{p.strategyName}</span>
              <span style={{ color: riskMeta[p.tokenRisk].color, fontSize: 10 }}>{riskMeta[p.tokenRisk].label}</span>
            </span>
            <span className="font-mono tabular-nums" style={{ color: pnlColor(p.pnlSol) }}>{p.pnlSol} SOL (%{p.pnlPct})</span>
          </li>
        ))}
      </ul>
    </div>
  );
}
