"use client";
import { useState } from "react";
import { useAlerts } from "@/lib/hooks/queries";
import { severityMeta, type AlertSeverity } from "@/lib/format";
import { Skeleton } from "@/components/ui/skeleton";
import { ALERT_SEVERITY_LABELS } from "@/lib/alerts/alert-defs";
import AlertHistoryRow from "./AlertHistoryRow";

// AlertHistoryPanel, alarm geçmişini (mevcut AlertEvent seam'i) severity filtresiyle listeler.
export default function AlertHistoryPanel() {
  const { data, isLoading, isError } = useAlerts();
  const [filter, setFilter] = useState<AlertSeverity | "all">("all");

  if (isLoading) {
    return (
      <div className="space-y-2">
        {Array.from({ length: 4 }).map((_, i) => (
          <Skeleton key={i} className="h-10 w-full" />
        ))}
      </div>
    );
  }
  if (isError || !data) return <p className="text-sm text-critical">Alarm geçmişi alınamadı.</p>;

  const filtered = filter === "all" ? data : data.filter((a) => a.severity === filter);

  return (
    <div>
      <div className="mb-3 flex flex-wrap gap-1.5">
        <FilterChip active={filter === "all"} onClick={() => setFilter("all")}>
          Tümü
        </FilterChip>
        {(Object.keys(ALERT_SEVERITY_LABELS) as AlertSeverity[]).map((s) => (
          <FilterChip key={s} active={filter === s} onClick={() => setFilter(s)} dot={severityMeta[s].dot}>
            {ALERT_SEVERITY_LABELS[s]}
          </FilterChip>
        ))}
      </div>
      {filtered.length === 0 ? (
        <p className="text-sm text-foreground/50">Bu filtrede alarm yok.</p>
      ) : (
        <div>
          {filtered.map((a) => (
            <AlertHistoryRow key={a.id} alert={a} />
          ))}
        </div>
      )}
    </div>
  );
}

function FilterChip({
  active,
  onClick,
  dot,
  children,
}: {
  active: boolean;
  onClick: () => void;
  dot?: string;
  children: React.ReactNode;
}) {
  return (
    <button
      type="button"
      onClick={onClick}
      aria-pressed={active}
      className={`inline-flex items-center gap-1.5 rounded-full border px-2.5 py-1 text-xs transition-colors ${
        active
          ? "border-primary/50 bg-primary/10 text-foreground"
          : "border-border/60 text-foreground/60 hover:text-foreground"
      }`}
    >
      {dot && <span className="h-1.5 w-1.5 rounded-full" style={{ background: dot }} aria-hidden />}
      {children}
    </button>
  );
}
