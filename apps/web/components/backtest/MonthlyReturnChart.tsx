"use client";
import { BarChart, Bar, XAxis, YAxis, ResponsiveContainer, Tooltip, Cell } from "recharts";
import type { MonthlyReturn } from "@/lib/api/types";
import { pnlColor } from "@/lib/position/risk-filter";

export function MonthlyReturnChart({ data }: { data: MonthlyReturn[] }) {
  return (
    <div className="rounded-lg border border-border bg-card p-3">
      <div className="mb-2 text-muted-foreground" style={{ fontSize: 12 }}>Aylık Getiri (%)</div>
      <div style={{ height: 220 }}>
        <ResponsiveContainer width="100%" height="100%">
          <BarChart data={data} margin={{ top: 4, right: 8, bottom: 0, left: 0 }}>
            <XAxis dataKey="label" tick={{ fill: "#8A94A6", fontSize: 11 }} />
            <YAxis hide />
            <Tooltip contentStyle={{ background: "#151C28", border: "1px solid rgba(255,255,255,0.1)", borderRadius: 8, fontSize: 12 }} cursor={{ fill: "rgba(255,255,255,0.04)" }} formatter={(v) => [`%${Number(v ?? 0)}`, "Getiri"]} />
            <Bar dataKey="pct" name="Getiri" radius={[4, 4, 0, 0]}>
              {data.map((d) => <Cell key={d.label} fill={pnlColor(d.pct)} />)}
            </Bar>
          </BarChart>
        </ResponsiveContainer>
      </div>
    </div>
  );
}
