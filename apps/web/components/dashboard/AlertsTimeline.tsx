"use client";
import { useAlerts } from "@/lib/hooks/queries";
import { severityMeta } from "@/lib/format";

export function AlertsTimeline() {
  const { data } = useAlerts();
  const alerts = data ?? [];
  return (
    <div className="flex h-full flex-col rounded-lg border border-border bg-card">
      <div className="flex items-center justify-between border-b border-border px-4 py-3">
        <h3>Alerts Timeline</h3>
        <button className="text-primary" style={{ fontSize: 12 }}>View all</button>
      </div>
      <div className="flex-1 overflow-y-auto p-3">
        <ol className="relative ml-1 space-y-3 border-l border-border pl-4">
          {alerts.map((a) => {
            const meta = severityMeta[a.severity];
            return (
              <li key={a.id} className="relative">
                <span className="absolute -left-[21px] top-1 h-2.5 w-2.5 rounded-full ring-4 ring-card" style={{ backgroundColor: meta.dot }} />
                <div className="flex items-center justify-between gap-2">
                  <span style={{ fontSize: 13, fontWeight: 500, color: meta.color }}>{a.type}</span>
                  <span className="text-muted-foreground" style={{ fontSize: 11 }}>{a.time}</span>
                </div>
                <div className="text-muted-foreground" style={{ fontSize: 12 }}>
                  <span className="font-mono text-foreground">{a.token}</span> · {a.detail}
                </div>
              </li>
            );
          })}
        </ol>
      </div>
    </div>
  );
}
