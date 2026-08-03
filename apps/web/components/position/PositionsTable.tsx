"use client";
import Link from "next/link";
import type { Position } from "@/lib/api/types";
import { riskMeta } from "@/lib/format";
import { pnlColor } from "@/lib/position/risk-filter";
import { PositionActions } from "./PositionActions";

export type SortKey = "pnlSol" | "sizeSol" | "ageLabel";

const HEADERS: { key?: SortKey; label: string }[] = [
  { label: "Token" }, { label: "Strateji" }, { label: "Giriş" }, { label: "Güncel" },
  { key: "sizeSol", label: "Boyut" }, { key: "pnlSol", label: "PnL" }, { label: "SL/TP" },
  { label: "Token Risk" }, { label: "Creator Risk" }, { key: "ageLabel", label: "Yaş" }, { label: "" },
];

export function PositionsTable({ rows, sortKey, onSort, onRowClick }: { rows: Position[]; sortKey: SortKey; onSort: (k: SortKey) => void; onRowClick: (p: Position) => void }) {
  return (
    <div className="rounded-lg border border-border bg-card">
      <div className="overflow-x-auto">
        <table className="w-full border-collapse" style={{ fontSize: 13 }}>
          <thead>
            <tr className="text-muted-foreground" style={{ fontSize: 11 }}>
              {HEADERS.map((h) => (
                <th key={h.label} className="whitespace-nowrap px-3 py-2 text-left font-normal">
                  {h.key ? (
                    <button onClick={() => onSort(h.key!)} className="hover:text-foreground" style={{ color: sortKey === h.key ? "#7C5CFF" : undefined }}>{h.label} ↕</button>
                  ) : h.label}
                </th>
              ))}
            </tr>
          </thead>
          <tbody>
            {rows.map((p) => (
              <tr key={p.id} onClick={() => onRowClick(p)} className="cursor-pointer border-t border-border hover:bg-accent/40">
                <td className="px-3 py-2"><Link href={`/tokens/${p.tokenSymbol}`} onClick={(e) => e.stopPropagation()} className="font-mono hover:underline">{p.tokenSymbol}</Link></td>
                <td className="px-3 py-2"><Link href={`/strategies/${p.strategyId}`} onClick={(e) => e.stopPropagation()} className="hover:underline">{p.strategyName}</Link></td>
                <td className="px-3 py-2 font-mono tabular-nums">{p.entryPrice}</td>
                <td className="px-3 py-2 font-mono tabular-nums">{p.currentPrice}</td>
                <td className="px-3 py-2 font-mono tabular-nums">{p.sizeSol} SOL</td>
                <td className="px-3 py-2 font-mono tabular-nums" style={{ color: pnlColor(p.pnlSol) }}>{p.pnlSol} (%{p.pnlPct})</td>
                <td className="px-3 py-2 font-mono tabular-nums">%{p.stopLossPct} / %{p.takeProfitPct}</td>
                <td className="px-3 py-2"><span style={{ color: riskMeta[p.tokenRisk].color, fontSize: 11 }}>{riskMeta[p.tokenRisk].label}</span></td>
                <td className="px-3 py-2"><span style={{ color: riskMeta[p.creatorRisk].color, fontSize: 11 }}>{riskMeta[p.creatorRisk].label}</span></td>
                <td className="px-3 py-2 font-mono tabular-nums">{p.ageLabel}</td>
                <td className="px-3 py-2" onClick={(e) => e.stopPropagation()}><PositionActions symbol={p.tokenSymbol} /></td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </div>
  );
}
