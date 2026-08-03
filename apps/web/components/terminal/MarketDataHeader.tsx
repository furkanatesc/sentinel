"use client";
import { useMarketData } from "@/lib/hooks/queries";
import { pnlColor } from "@/lib/position/risk-filter";
import { scoreToLevel, riskMeta } from "@/lib/format";

function Stat({ label, value }: { label: string; value: string }) {
  return (
    <div className="flex flex-col">
      <span className="text-muted-foreground" style={{ fontSize: 11 }}>{label}</span>
      <span style={{ fontSize: 13 }}>{value}</span>
    </div>
  );
}
function ScoreBadge({ label, score }: { label: string; score: number }) {
  const m = riskMeta[scoreToLevel(score)];
  return (
    <span className="rounded-md px-2 py-0.5" style={{ fontSize: 11, color: m.color, backgroundColor: m.bg, border: `1px solid ${m.border}` }}>
      {label} {score}
    </span>
  );
}

export function MarketDataHeader({ mint }: { mint: string }) {
  const { data } = useMarketData(mint);
  if (!data) return <div className="h-16 rounded-lg border border-border bg-card" />;
  return (
    <div className="flex flex-wrap items-center gap-4 rounded-lg border border-border bg-card px-4 py-3">
      <span className="font-semibold">{data.symbol}</span>
      <span style={{ fontSize: 14 }}>{data.price} SOL</span>
      <span style={{ fontSize: 13, color: pnlColor(data.change24hPct) }}>%{data.change24hPct}</span>
      <Stat label="Likidite" value={`${data.liquiditySol} SOL`} />
      <Stat label="Hacim 24s" value={`${data.volume24hSol} SOL`} />
      <Stat label="Piyasa Değeri" value={`${data.marketCapSol} SOL`} />
      <div className="ml-auto flex items-center gap-2">
        <ScoreBadge label="Token" score={data.tokenScore} />
        <ScoreBadge label="Creator" score={data.creatorScore} />
      </div>
    </div>
  );
}
