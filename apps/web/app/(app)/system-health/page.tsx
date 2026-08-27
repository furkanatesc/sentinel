"use client";
import { useSystemHealth } from "@/lib/hooks/queries";
import type { WorkerState } from "@/lib/api/types";
import { WORKER_STATE_DEFS } from "@/lib/system-health/worker-state-defs";
import { MetricTile } from "@/components/sentinel/MetricTile";
import { Skeleton } from "@/components/ui/skeleton";

function ago(iso: string): string {
  if (!iso) return "—";
  const s = Math.max(0, Math.floor((Date.now() - new Date(iso).getTime()) / 1000));
  if (s < 60) return `${s} sn önce`;
  if (s < 3600) return `${Math.floor(s / 60)} dk önce`;
  return `${Math.floor(s / 3600)} sa önce`;
}

function formatUptime(s: number): string {
  if (s < 60) return `${s} sn`;
  if (s < 3600) return `${Math.floor(s / 60)} dk`;
  const h = Math.floor(s / 3600);
  const m = Math.floor((s % 3600) / 60);
  return `${h} sa ${m} dk`;
}

function WorkerStateBadge({ state }: { state: WorkerState }) {
  const d = WORKER_STATE_DEFS[state];
  return (
    <span
      className="inline-flex rounded px-1.5 py-0.5"
      style={{ color: d.color, backgroundColor: `${d.color}1f`, fontSize: 11, fontWeight: 600 }}
    >
      {d.label}
    </span>
  );
}

const TABLE_HEADERS = ["Worker", "Durum", "Son Çalışma", "Cycles", "İşlenen", "Hata"];

export default function Page() {
  const { data, isLoading, isError } = useSystemHealth();

  if (isError) {
    return (
      <div className="space-y-5">
        <h1>Sistem Sağlığı</h1>
        <div className="rounded-lg border border-border bg-card p-8 text-center text-muted-foreground">
          Sistem sağlığı alınamadı.
        </div>
      </div>
    );
  }

  if (isLoading || !data) {
    return (
      <div className="space-y-5">
        <h1>Sistem Sağlığı</h1>
        <div className="grid grid-cols-2 gap-3 md:grid-cols-4">
          <Skeleton className="h-20 w-full" />
          <Skeleton className="h-20 w-full" />
          <Skeleton className="h-20 w-full" />
          <Skeleton className="h-20 w-full" />
        </div>
        <Skeleton className="h-56 w-full" />
      </div>
    );
  }

  const gateEntries = Object.entries(data.gates);

  return (
    <div className="space-y-5">
      <div className="flex items-center justify-between gap-2">
        <h1>Sistem Sağlığı</h1>
        <span className="text-muted-foreground" style={{ fontSize: 11 }}>10 sn&apos;de bir yenilenir</span>
      </div>

      <div className="grid grid-cols-2 gap-3 md:grid-cols-4">
        <MetricTile
          label="Veritabanı"
          value={data.dbOk ? "Bağlı" : "Erişilemiyor"}
          hint={data.dbOk ? `${data.dbLatencyMs}ms gecikme` : undefined}
          valueColor={data.dbOk ? "#2FD98B" : "#F0476B"}
        />
        <MetricTile label="Çalışma Süresi" value={formatUptime(data.uptimeSec)} />
        <MetricTile label="Versiyon" value={data.version} />
        <MetricTile label="WS İstemci" value={String(data.wsClients)} />
      </div>

      <div className="rounded-lg border border-border bg-card">
        <div className="border-b border-border px-4 py-3"><h3>Worker&apos;lar</h3></div>
        {data.workers.length > 0 ? (
          <div className="overflow-x-auto">
            <table className="w-full border-collapse" style={{ fontSize: 13 }}>
              <thead>
                <tr className="text-muted-foreground" style={{ fontSize: 11 }}>
                  {TABLE_HEADERS.map((h) => (
                    <th key={h} className="whitespace-nowrap px-3 py-2 text-left font-normal">{h}</th>
                  ))}
                </tr>
              </thead>
              <tbody>
                {data.workers.map((w) => (
                  <tr key={w.name} className="border-t border-border hover:bg-accent/40">
                    <td className="whitespace-nowrap px-3 py-2 font-mono" style={{ fontWeight: 500 }}>{w.name}</td>
                    <td className="px-3 py-2"><WorkerStateBadge state={w.state} /></td>
                    <td className="whitespace-nowrap px-3 py-2 text-muted-foreground">{ago(w.lastRunAt)}</td>
                    <td className="whitespace-nowrap px-3 py-2 font-mono tabular-nums">{w.cyclesRun}</td>
                    <td className="whitespace-nowrap px-3 py-2 font-mono tabular-nums">{w.itemsProcessed}</td>
                    <td
                      className="max-w-xs truncate px-3 py-2"
                      style={{ color: w.lastErr ? "#F0476B" : undefined }}
                      title={w.lastErr || undefined}
                    >
                      {w.lastErr || <span className="text-muted-foreground">—</span>}
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        ) : (
          <div className="p-6 text-center text-muted-foreground" style={{ fontSize: 13 }}>Worker yok.</div>
        )}
      </div>

      <div className="rounded-lg border border-border bg-card p-4">
        <h3 className="mb-2">Gate&apos;ler</h3>
        {gateEntries.length > 0 ? (
          <div className="flex flex-wrap gap-2">
            {gateEntries.map(([k, v]) => (
              <span
                key={k}
                className="rounded px-1.5 py-0.5 font-mono"
                style={{
                  color: v ? "#2FD98B" : "#8A94A6",
                  backgroundColor: v ? "rgba(47,217,139,0.12)" : "rgba(138,148,166,0.12)",
                  fontSize: 11,
                }}
              >
                {k}={v ? "on" : "off"}
              </span>
            ))}
          </div>
        ) : (
          <span className="text-muted-foreground" style={{ fontSize: 13 }}>Gate yok.</span>
        )}
      </div>
    </div>
  );
}
