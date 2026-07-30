"use client";
import { AreaChart, Area, XAxis, YAxis, ResponsiveContainer, Tooltip } from "recharts";
import type { TokenDetail, SeriesPoint } from "@/lib/api/types";
import { MetricTile } from "@/components/sentinel/MetricTile";
import { formatPct } from "@/lib/format";

function MiniChart({ title, data, color }: { title: string; data: SeriesPoint[]; color: string }) {
  return (
    <div className="rounded-lg border border-border bg-card p-3">
      <div className="mb-2 text-muted-foreground" style={{ fontSize: 12 }}>{title}</div>
      <div style={{ height: 140 }}>
        <ResponsiveContainer width="100%" height="100%">
          <AreaChart data={data} margin={{ top: 4, right: 4, bottom: 0, left: 0 }}>
            <defs><linearGradient id={`g-${color.slice(1)}`} x1="0" y1="0" x2="0" y2="1"><stop offset="0%" stopColor={color} stopOpacity={0.3} /><stop offset="100%" stopColor={color} stopOpacity={0} /></linearGradient></defs>
            <XAxis dataKey="t" hide /><YAxis hide domain={["dataMin", "dataMax"]} />
            <Tooltip contentStyle={{ background: "#151C28", border: "1px solid rgba(255,255,255,0.1)", borderRadius: 8, fontSize: 12 }} labelFormatter={() => ""} />
            <Area type="monotone" dataKey="v" stroke={color} strokeWidth={1.5} fill={`url(#g-${color.slice(1)})`} />
          </AreaChart>
        </ResponsiveContainer>
      </div>
    </div>
  );
}

export function OverviewTab({ token }: { token: TokenDetail }) {
  const m = token.metrics;
  return (
    <div className="space-y-4">
      <div className="grid grid-cols-1 gap-3 md:grid-cols-2">
        <MiniChart title="Fiyat" data={token.series.price} color="#7C5CFF" />
        <MiniChart title="Likidite" data={token.series.liquidity} color="#3E9BFF" />
        <MiniChart title="Hacim" data={token.series.volume} color="#2FD98B" />
        <MiniChart title="Holder Büyümesi" data={token.series.holders} color="#FFB020" />
      </div>
      <div className="grid grid-cols-2 gap-3 md:grid-cols-3 lg:grid-cols-6">
        <MetricTile label="Benzersiz Alıcı" value={`${m.uniqueBuyers}`} />
        <MetricTile label="Al/Sat" value={`${m.buyRatio}/${m.sellRatio}`} />
        <MetricTile label="Üretici Payı" value={formatPct(m.creatorHoldingPct)} />
        <MetricTile label="Top-10 Holder" value={formatPct(m.top10HolderPct)} />
        <MetricTile label="Sniper" value={formatPct(m.sniperPct)} />
        <MetricTile label="Bot Aktivitesi" value={formatPct(m.botActivityPct)} />
      </div>
    </div>
  );
}
