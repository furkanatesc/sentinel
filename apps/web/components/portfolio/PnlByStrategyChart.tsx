"use client";
import { BarChart, Bar, XAxis, YAxis, ResponsiveContainer, Tooltip, Cell } from "recharts";
import type { StrategyPnl } from "@/lib/api/types";
import { pnlColor } from "@/lib/position/risk-filter";

export function PnlByStrategyChart({ data }: { data: StrategyPnl[] }) {
  return (
    <div className="rounded-lg border border-border bg-card p-3">
      <div className="mb-2 text-muted-foreground" style={{ fontSize: 12 }}>Strateji Bazında PnL (SOL)</div>
      <div style={{ height: 220 }}>
        <ResponsiveContainer width="100%" height="100%">
          <BarChart data={data} layout="vertical" margin={{ top: 4, right: 8, bottom: 0, left: 8 }}>
            <XAxis type="number" hide />
            <YAxis type="category" dataKey="name" width={110} tick={{ fill: "#8A94A6", fontSize: 11 }} />
            <Tooltip contentStyle={{ background: "#151C28", border: "1px solid rgba(255,255,255,0.1)", borderRadius: 8, fontSize: 12 }} cursor={{ fill: "rgba(255,255,255,0.04)" }} />
            <Bar dataKey="pnlSol" radius={[0, 4, 4, 0]}>
              {data.map((d) => <Cell key={d.strategyId} fill={pnlColor(d.pnlSol)} />)}
            </Bar>
          </BarChart>
        </ResponsiveContainer>
      </div>
    </div>
  );
}
