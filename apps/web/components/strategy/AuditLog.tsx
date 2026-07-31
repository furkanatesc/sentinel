import type { AuditEntry } from "@/lib/api/types";

export function AuditLog({ audit }: { audit: AuditEntry[] }) {
  return (
    <div className="rounded-lg border border-border bg-card p-4">
      <div className="mb-3 font-medium" style={{ fontSize: 13 }}>Denetim Kaydı</div>
      <ul className="space-y-2">
        {audit.map((a, i) => (
          <li key={`${a.time}-${i}`} className="flex items-start gap-3">
            <span className="font-mono text-muted-foreground" style={{ fontSize: 11 }}>{a.time}</span>
            <span className="font-medium" style={{ fontSize: 12 }}>{a.action}</span>
            <span className="text-muted-foreground" style={{ fontSize: 12 }}>{a.detail}</span>
          </li>
        ))}
      </ul>
    </div>
  );
}
