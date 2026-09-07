import type { AlertEvent } from "@/lib/api/types";
import { severityMeta } from "@/lib/format";

// AlertHistoryRow, tek bir alarm olayını timeline satırı olarak render eder (SRP).
export default function AlertHistoryRow({ alert }: { alert: AlertEvent }) {
  return (
    <div className="flex items-start gap-3 border-b border-border/50 py-2.5">
      <span
        className="mt-1.5 h-2 w-2 shrink-0 rounded-full"
        style={{ background: severityMeta[alert.severity].dot }}
        aria-hidden
      />
      <div className="min-w-0 flex-1">
        <div className="flex items-center gap-2">
          <span className="text-sm font-medium text-foreground">{alert.type}</span>
          <span className="font-mono text-xs text-foreground/60">{alert.token}</span>
        </div>
        <p className="truncate text-sm text-foreground/70">{alert.detail}</p>
      </div>
      <span className="shrink-0 text-xs text-foreground/40">{alert.time}</span>
    </div>
  );
}
