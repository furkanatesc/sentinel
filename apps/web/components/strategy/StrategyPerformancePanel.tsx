import type { StrategyPerformance } from "@/lib/api/types";
import { MetricTile } from "@/components/sentinel/MetricTile";

export function StrategyPerformancePanel({ performance: p }: { performance: StrategyPerformance }) {
  const tiles: { label: string; value: string }[] = [
    { label: "Kazanç Oranı", value: `%${p.winRatePct}` },
    { label: "Profit Factor", value: p.profitFactor.toFixed(2) },
    { label: "Maks. Drawdown", value: `%${p.maxDrawdownPct}` },
    { label: "Sharpe", value: `${p.sharpe}` },
    { label: "Sortino", value: `${p.sortino}` },
    { label: "İşlem", value: `${p.totalTrades}` },
    { label: "Net PnL", value: `${p.netPnlSol} SOL` },
    { label: "Beklenen Değer", value: `${p.expectancy} SOL` },
  ];
  return (
    <div className="grid grid-cols-2 gap-3 md:grid-cols-4">
      {tiles.map((t) => <MetricTile key={t.label} label={t.label} value={t.value} />)}
    </div>
  );
}
