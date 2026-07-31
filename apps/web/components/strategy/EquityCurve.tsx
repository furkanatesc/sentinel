"use client";
import { AreaChart, Area, XAxis, YAxis, ResponsiveContainer, Tooltip } from "recharts";
import type { EquityPoint } from "@/lib/api/types";

export function EquityCurve({ data }: { data: EquityPoint[] }) {
  const color = "#2FD98B";
  return (
    <div className="rounded-lg border border-border bg-card p-3">
      <div className="mb-2 text-muted-foreground" style={{ fontSize: 12 }}>Equity Curve</div>
      <div style={{ height: 220 }}>
        <ResponsiveContainer width="100%" height="100%">
          <AreaChart data={data} margin={{ top: 4, right: 4, bottom: 0, left: 0 }}>
            <defs>
              <linearGradient id="equity-grad" x1="0" y1="0" x2="0" y2="1">
                <stop offset="0%" stopColor={color} stopOpacity={0.3} />
                <stop offset="100%" stopColor={color} stopOpacity={0} />
              </linearGradient>
            </defs>
            <XAxis dataKey="t" hide />
            <YAxis hide domain={["dataMin", "dataMax"]} />
            <Tooltip
              contentStyle={{ background: "#151C28", border: "1px solid rgba(255,255,255,0.1)", borderRadius: 8, fontSize: 12 }}
              labelFormatter={() => ""}
            />
            <Area type="monotone" dataKey="v" stroke={color} strokeWidth={1.5} fill="url(#equity-grad)" />
          </AreaChart>
        </ResponsiveContainer>
      </div>
    </div>
  );
}
