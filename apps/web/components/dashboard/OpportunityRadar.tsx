"use client";
import { ScatterChart, Scatter, XAxis, YAxis, ZAxis, CartesianGrid, Tooltip, ResponsiveContainer, Cell } from "recharts";
import { useRadar } from "@/lib/hooks/queries";
import { riskMeta } from "@/lib/format";

export function OpportunityRadar() {
  const { data } = useRadar();
  const radarData = data ?? [];
  return (
    <div className="flex h-full flex-col rounded-lg border border-border bg-card">
      <div className="flex items-center justify-between border-b border-border px-4 py-3">
        <h3>Fırsat Radarı</h3>
        <span className="text-muted-foreground" style={{ fontSize: 11 }}>Üretici Güveni × Momentum · boyut = likidite</span>
      </div>
      <div className="p-3" style={{ height: 320 }}>
        <ResponsiveContainer width="100%" height="100%">
          <ScatterChart margin={{ top: 10, right: 10, bottom: 20, left: 0 }}>
            <CartesianGrid stroke="rgba(255,255,255,0.05)" />
            <XAxis type="number" dataKey="x" name="Creator Trust" domain={[0, 100]} tick={{ fill: "#8A94A6", fontSize: 11 }} stroke="rgba(255,255,255,0.1)"
              label={{ value: "Creator Trust Score", position: "insideBottom", offset: -12, fill: "#8A94A6", fontSize: 11 }} />
            <YAxis type="number" dataKey="y" name="Momentum" domain={[0, 100]} tick={{ fill: "#8A94A6", fontSize: 11 }} stroke="rgba(255,255,255,0.1)"
              label={{ value: "Momentum", angle: -90, position: "insideLeft", fill: "#8A94A6", fontSize: 11 }} />
            <ZAxis type="number" dataKey="z" range={[80, 600]} />
            <Tooltip cursor={{ strokeDasharray: "3 3", stroke: "rgba(255,255,255,0.2)" }}
              contentStyle={{ background: "#151C28", border: "1px solid rgba(255,255,255,0.1)", borderRadius: 8, fontSize: 12 }}
              formatter={(v, n) => [v, n]} labelFormatter={() => ""} />
            <Scatter data={radarData}>
              {radarData.map((d, i) => (<Cell key={i} fill={riskMeta[d.level].color} fillOpacity={0.75} />))}
            </Scatter>
          </ScatterChart>
        </ResponsiveContainer>
      </div>
      <div className="flex flex-wrap gap-3 border-t border-border px-4 py-2.5">
        {(["strong", "good", "medium", "high", "critical"] as const).map((l) => (
          <span key={l} className="flex items-center gap-1.5 text-muted-foreground" style={{ fontSize: 11 }}>
            <span className="h-2 w-2 rounded-full" style={{ backgroundColor: riskMeta[l].color }} />{riskMeta[l].label}
          </span>
        ))}
      </div>
    </div>
  );
}
