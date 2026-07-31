"use client";
import { useState } from "react";
import { usePositions } from "@/lib/hooks/queries";
import { riskMeta, type RiskLevel } from "@/lib/format";
import { POSITION_RISK_LEVELS } from "@/lib/position/risk-filter";
import type { Position } from "@/lib/api/types";
import { PositionsTable, type SortKey } from "./PositionsTable";
import { PositionDetailDrawer } from "./PositionDetailDrawer";

export function PositionsContent() {
  const { data } = usePositions();
  const [risk, setRisk] = useState<RiskLevel | null>(null);
  const [sortKey, setSortKey] = useState<SortKey>("pnlSol");
  const [selected, setSelected] = useState<Position | null>(null);

  const rows = (data ?? [])
    .filter((p) => (risk ? p.tokenRisk === risk : true))
    .sort((a, b) => (sortKey === "ageLabel" ? a.ageLabel.localeCompare(b.ageLabel) : (b[sortKey] as number) - (a[sortKey] as number)));

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
      <PositionsTable rows={rows} sortKey={sortKey} onSort={setSortKey} onRowClick={setSelected} />
      <PositionDetailDrawer position={selected} onClose={() => setSelected(null)} />
    </div>
  );
}
