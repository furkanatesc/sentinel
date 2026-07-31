"use client";
import { PieChart, Pie, Cell, ResponsiveContainer, Tooltip, Legend } from "recharts";
import type { AllocationSlice } from "@/lib/api/types";

export function RiskAllocationChart({ data }: { data: AllocationSlice[] }) {
  return (
    <div className="rounded-lg border border-border bg-card p-3">
      <div className="mb-2 text-muted-foreground" style={{ fontSize: 12 }}>Risk Dağılımı</div>
      <div style={{ height: 220 }}>
        <ResponsiveContainer width="100%" height="100%">
          <PieChart>
            <Pie data={data} dataKey="pct" nameKey="label" innerRadius={55} outerRadius={85} paddingAngle={2} stroke="#0B0F17" strokeWidth={2}>
              {data.map((d) => <Cell key={d.label} fill={d.color} />)}
            </Pie>
            <Tooltip contentStyle={{ background: "#151C28", border: "1px solid rgba(255,255,255,0.1)", borderRadius: 8, fontSize: 12 }} formatter={(v: number, n) => [`%${v}`, n]} />
            <Legend wrapperStyle={{ fontSize: 11 }} />
          </PieChart>
        </ResponsiveContainer>
      </div>
    </div>
  );
}
