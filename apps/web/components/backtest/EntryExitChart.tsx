"use client";
import { ComposedChart, Line, Scatter, XAxis, YAxis, ResponsiveContainer, Tooltip } from "recharts";
import type { EquityPoint, BacktestTrade } from "@/lib/api/types";

export function EntryExitChart({ priceSeries, trades }: { priceSeries: EquityPoint[]; trades: BacktestTrade[] }) {
  const merged = priceSeries.map((p) => {
    const tr = trades.find((t) => t.time === p.t);
    return {
      t: p.t, v: p.v,
      buy: tr?.side === "buy" ? tr.price : undefined,
      sell: tr?.side === "sell" ? tr.price : undefined,
    };
  });
  return (
    <div className="rounded-lg border border-border bg-card p-3">
      <div className="mb-2 text-muted-foreground" style={{ fontSize: 12 }}>Giriş / Çıkış Noktaları</div>
      <div style={{ height: 260 }}>
        <ResponsiveContainer width="100%" height="100%">
          <ComposedChart data={merged} margin={{ top: 4, right: 8, bottom: 0, left: 0 }}>
            <XAxis dataKey="t" hide />
            <YAxis hide domain={["dataMin", "dataMax"]} />
            <Tooltip contentStyle={{ background: "#151C28", border: "1px solid rgba(255,255,255,0.1)", borderRadius: 8, fontSize: 12 }} labelFormatter={() => ""} />
            <Line type="monotone" dataKey="v" stroke="#8A94A6" strokeWidth={1.5} dot={false} name="Fiyat" />
            <Scatter dataKey="buy" fill="#2FD98B" name="Al" />
            <Scatter dataKey="sell" fill="#F0476B" name="Sat" />
          </ComposedChart>
        </ResponsiveContainer>
      </div>
    </div>
  );
}
