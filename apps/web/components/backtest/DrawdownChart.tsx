"use client";
import { AreaChart, Area, XAxis, YAxis, ResponsiveContainer, Tooltip } from "recharts";
import type { DrawdownPoint } from "@/lib/api/types";

export function DrawdownChart({ data }: { data: DrawdownPoint[] }) {
  return (
    <div className="rounded-lg border border-border bg-card p-3">
      <div className="mb-2 text-muted-foreground" style={{ fontSize: 12 }}>Drawdown (%)</div>
      <div style={{ height: 220 }}>
        <ResponsiveContainer width="100%" height="100%">
          <AreaChart data={data} margin={{ top: 4, right: 4, bottom: 0, left: 0 }}>
            <defs>
              <linearGradient id="bt-dd-grad" x1="0" y1="0" x2="0" y2="1">
                <stop offset="0%" stopColor="#F0476B" stopOpacity={0.05} />
                <stop offset="100%" stopColor="#F0476B" stopOpacity={0.35} />
              </linearGradient>
            </defs>
            <XAxis dataKey="t" hide />
            <YAxis hide domain={["dataMin", 0]} />
            <Tooltip contentStyle={{ background: "#151C28", border: "1px solid rgba(255,255,255,0.1)", borderRadius: 8, fontSize: 12 }} labelFormatter={() => ""} formatter={(v) => [`%${Number(v ?? 0)}`, "Drawdown"]} />
            <Area type="monotone" dataKey="v" stroke="#F0476B" strokeWidth={1.5} fill="url(#bt-dd-grad)" />
          </AreaChart>
        </ResponsiveContainer>
      </div>
    </div>
  );
}
