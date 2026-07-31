import type { PortfolioSummary } from "@/lib/api/types";
import { MetricTile } from "@/components/sentinel/MetricTile";
import { pnlColor } from "@/lib/position/risk-filter";

export function PortfolioKpis({ summary: s }: { summary: PortfolioSummary }) {
  const tiles: { label: string; value: string; valueColor?: string }[] = [
    { label: "Toplam Değer", value: `${s.totalValueSol} SOL` },
    { label: "Kullanılabilir", value: `${s.availableSol} SOL` },
    { label: "Yatırılan", value: `${s.investedSol} SOL` },
    { label: "Gerçekleşen PnL", value: `${s.realizedPnlSol} SOL`, valueColor: pnlColor(s.realizedPnlSol) },
    { label: "Gerçekleşmemiş PnL", value: `${s.unrealizedPnlSol} SOL`, valueColor: pnlColor(s.unrealizedPnlSol) },
    { label: "Günlük PnL", value: `${s.dailyPnlSol} SOL`, valueColor: pnlColor(s.dailyPnlSol) },
    { label: "Maks. Drawdown", value: `%${s.maxDrawdownPct}` },
    { label: "Risk Maruziyeti", value: `%${s.riskExposurePct}` },
    { label: "Rug Maruziyeti", value: `%${s.rugExposurePct}` },
  ];
  return (
    <div className="grid grid-cols-2 gap-3 md:grid-cols-3 lg:grid-cols-5">
      {tiles.map((t) => <MetricTile key={t.label} label={t.label} value={t.value} valueColor={t.valueColor} />)}
    </div>
  );
}
