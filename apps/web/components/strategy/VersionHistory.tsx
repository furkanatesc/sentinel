import type { StrategyVersion } from "@/lib/api/types";

export function VersionHistory({ versions }: { versions: StrategyVersion[] }) {
  return (
    <div className="rounded-lg border border-border bg-card p-4">
      <div className="mb-3 font-medium" style={{ fontSize: 13 }}>Sürüm Geçmişi</div>
      <ul className="space-y-2">
        {versions.map((v) => (
          <li key={v.version} className="flex items-start gap-3">
            <span className="font-mono text-primary" style={{ fontSize: 12 }}>{v.version}</span>
            <span className="text-muted-foreground" style={{ fontSize: 11 }}>{v.date}</span>
            <span style={{ fontSize: 12 }}>{v.note}</span>
          </li>
        ))}
      </ul>
    </div>
  );
}
