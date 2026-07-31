"use client";
import { useState } from "react";
import { usePositions } from "@/lib/hooks/queries";
import { riskMeta, type RiskLevel } from "@/lib/format";
import { POSITION_RISK_LEVELS } from "@/lib/position/risk-filter";
import type { Position } from "@/lib/api/types";
import { Skeleton } from "@/components/ui/skeleton";
import { PositionsTable, type SortKey } from "./PositionsTable";
import { PositionDetailDrawer } from "./PositionDetailDrawer";

export function PositionsContent() {
  const { data, isError } = usePositions();
  const [risk, setRisk] = useState<RiskLevel | null>(null);
  const [sortKey, setSortKey] = useState<SortKey>("pnlSol");
  const [selected, setSelected] = useState<Position | null>(null);

  if (isError) return <div className="rounded-lg border border-border bg-card p-8 text-center text-muted-foreground">Pozisyonlar yüklenemedi.</div>;
  if (!data) return <div className="space-y-4"><Skeleton className="h-24 w-full" /><Skeleton className="h-56 w-full" /></div>;

  const rows = data
    .filter((p) => (risk ? p.tokenRisk === risk : true))
    .sort((a, b) => (sortKey === "ageLabel" ? parseInt(b.ageLabel, 10) - parseInt(a.ageLabel, 10) : (b[sortKey] as number) - (a[sortKey] as number)));

  return (
    <div className="space-y-4">
      <h1>Pozisyonlar</h1>
      <div className="flex flex-wrap items-center gap-2">
        {POSITION_RISK_LEVELS.map((r) => (
          <button key={r} type="button" onClick={() => setRisk(risk === r ? null : r)} className="rounded-md border px-2.5 py-1"
            style={{ fontSize: 12, borderColor: risk === r ? riskMeta[r].color : "var(--border)", color: risk === r ? riskMeta[r].color : "inherit" }}>
            {riskMeta[r].label}
          </button>
        ))}
        {risk && <button type="button" onClick={() => setRisk(null)} className="text-muted-foreground" style={{ fontSize: 12 }}>Temizle</button>}
      </div>
      {rows.length > 0 ? (
        <PositionsTable rows={rows} sortKey={sortKey} onSort={setSortKey} onRowClick={setSelected} />
      ) : (
        <div className="text-muted-foreground" style={{ fontSize: 13 }}>Sonuç yok.</div>
      )}
      <PositionDetailDrawer position={selected} onClose={() => setSelected(null)} />
    </div>
  );
}
