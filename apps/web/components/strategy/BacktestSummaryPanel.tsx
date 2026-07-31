import type { BacktestSummary } from "@/lib/api/types";
import { MetricTile } from "@/components/sentinel/MetricTile";

export function BacktestSummaryPanel({ backtest: b }: { backtest: BacktestSummary }) {
  const tiles: { label: string; value: string }[] = [
    { label: "Net PnL", value: `${b.netPnlSol} SOL` },
    { label: "Kazanç Oranı", value: `%${b.winRatePct}` },
    { label: "Profit Factor", value: b.profitFactor.toFixed(2) },
    { label: "Sharpe", value: `${b.sharpe}` },
    { label: "Maks. Drawdown", value: `%${b.maxDrawdownPct}` },
    { label: "İşlem Sayısı", value: `${b.trades}` },
    { label: "Ort. Tutma", value: `${b.avgHoldingHours}s` },
    { label: "Rug Maruziyeti", value: `%${b.rugExposurePct}` },
  ];
  return (
    <div className="grid grid-cols-2 gap-3 md:grid-cols-4">
      {tiles.map((t) => <MetricTile key={t.label} label={t.label} value={t.value} />)}
    </div>
  );
}
