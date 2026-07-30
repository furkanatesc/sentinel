import Link from "next/link";
import type { CreatorTokenHistoryItem } from "@/lib/api/types";
import { OUTCOME_DEFS, LIQUIDITY_DEFS } from "@/lib/creator/outcome-defs";
import { formatUsd } from "@/lib/format";

export function CreatorTokenHistoryTable({ history }: { history: CreatorTokenHistoryItem[] }) {
  return (
    <div className="rounded-lg border border-border bg-card">
      <div className="border-b border-border px-4 py-3"><h3>Token Geçmişi</h3></div>
      <div className="overflow-x-auto">
        <table className="w-full border-collapse" style={{ fontSize: 13 }}>
          <thead>
            <tr className="text-muted-foreground" style={{ fontSize: 11 }}>
              {["Token", "Oluşturma", "Peak MC", "Mevcut MC", "Max Düşüş", "Likidite", "Satış %", "Sonuç"].map((h) => (
                <th key={h} className="whitespace-nowrap px-3 py-2 text-left font-normal">{h}</th>
              ))}
            </tr>
          </thead>
          <tbody>
            {history.map((t) => {
              const om = OUTCOME_DEFS[t.outcome]; const lm = LIQUIDITY_DEFS[t.liquidityStatus];
              return (
                <tr key={t.id} className="border-t border-border hover:bg-accent/40">
                  <td className="px-3 py-2"><Link href={`/tokens/${t.symbol}`} className="hover:underline" style={{ fontWeight: 500 }}>{t.symbol}</Link></td>
                  <td className="whitespace-nowrap px-3 py-2 text-muted-foreground">{t.createdAt}</td>
                  <td className="whitespace-nowrap px-3 py-2 font-mono tabular-nums">{formatUsd(t.peakMarketCap)}</td>
                  <td className="whitespace-nowrap px-3 py-2 font-mono tabular-nums">{formatUsd(t.currentMarketCap)}</td>
                  <td className="whitespace-nowrap px-3 py-2 font-mono tabular-nums" style={{ color: "#F0476B" }}>-%{t.maxDrawdownPct}</td>
                  <td className="px-3 py-2"><span className="rounded px-1.5 py-0.5" style={{ color: lm.color, backgroundColor: `${lm.color}1f`, fontSize: 11 }}>{lm.label}</span></td>
                  <td className="whitespace-nowrap px-3 py-2 font-mono tabular-nums">%{t.creatorSellPct}</td>
                  <td className="px-3 py-2"><span className="rounded px-1.5 py-0.5" style={{ color: om.color, backgroundColor: `${om.color}1f`, fontSize: 11, fontWeight: 600 }}>{om.label}</span></td>
                </tr>
              );
            })}
          </tbody>
        </table>
      </div>
    </div>
  );
}
