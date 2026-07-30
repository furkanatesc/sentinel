import type { CreatorMetrics as CM } from "@/lib/api/types";
import { MetricTile } from "@/components/sentinel/MetricTile";
import { formatUsd } from "@/lib/format";

export function CreatorMetrics({ metrics: m }: { metrics: CM }) {
  const tiles: { label: string; value: string }[] = [
    { label: "Toplam Token", value: `${m.totalTokens}` },
    { label: "Aktif Token", value: `${m.activeTokens}` },
    { label: "Rug Token", value: `${m.ruggedTokens}` },
    { label: "Ort. Ömür", value: `${m.avgLifetimeHours}s` },
    { label: "Ort. Peak MC", value: formatUsd(m.avgPeakMarketCap) },
    { label: "Realized PnL", value: `${m.realizedPnlSol} SOL` },
    { label: "Başarı Oranı", value: `%${m.successRatePct}` },
    { label: "Ort. İlk Satış", value: `${m.avgFirstSellMinutes} dk` },
  ];
  return (
    <div className="grid grid-cols-2 gap-3 md:grid-cols-4">
      {tiles.map((t) => <MetricTile key={t.label} label={t.label} value={t.value} />)}
    </div>
  );
}
