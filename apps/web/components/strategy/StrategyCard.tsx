import Link from "next/link";
import type { StrategyRow } from "@/lib/api/types";
import { StatusBadge } from "./StatusBadge";

function pnlColor(v: number) {
  return v >= 0 ? "#2FD98B" : "#F0476B";
}

export function StrategyCard({ row }: { row: StrategyRow }) {
  const stats: { label: string; value: string; color?: string }[] = [
    { label: "Kazanç Oranı", value: `%${row.winRatePct}` },
    { label: "Profit Factor", value: row.profitFactor.toFixed(2) },
    { label: "Maks. Drawdown", value: `%${row.maxDrawdownPct}` },
    { label: "İşlem", value: `${row.totalTrades}` },
    { label: "Net PnL", value: `${row.netPnlSol} SOL`, color: pnlColor(row.netPnlSol) },
  ];
  return (
    <Link
      href={`/strategies/${row.id}`}
      className="block rounded-lg border border-border bg-card p-4 transition-colors hover:border-primary/50"
    >
      <div className="flex items-center justify-between gap-2">
        <div className="font-medium">{row.name}</div>
        <StatusBadge status={row.status} />
      </div>
      <div className="mt-1 text-muted-foreground" style={{ fontSize: 11 }}>
        {row.timeframe} · son sinyal {row.lastSignal}
      </div>
      <div className="mt-3 grid grid-cols-3 gap-2">
        {stats.map((s) => (
          <div key={s.label}>
            <div className="text-muted-foreground" style={{ fontSize: 10 }}>{s.label}</div>
            <div className="font-mono tabular-nums" style={{ fontSize: 14, color: s.color }}>{s.value}</div>
          </div>
        ))}
      </div>
    </Link>
  );
}
