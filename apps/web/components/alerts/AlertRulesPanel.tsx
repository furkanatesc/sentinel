"use client";
import { useState } from "react";
import { Plus } from "lucide-react";
import { useAlertRules } from "@/lib/hooks/queries";
import { useSetAlertRuleEnabled } from "@/lib/hooks/mutations";
import { Skeleton } from "@/components/ui/skeleton";
import AlertRuleCard from "./AlertRuleCard";

// AlertRulesPanel, alarm kurallarını listeler. Aç/kapa GERÇEK (setAlertRuleEnabled mutation → DB persist,
// Backend Alt-proje 3); local override yalnız anlık geri-bildirim (invalidate-refetch gerçeği getirir).
export default function AlertRulesPanel({ onNew }: { onNew?: () => void }) {
  const { data, isLoading, isError } = useAlertRules();
  const setEnabled = useSetAlertRuleEnabled();
  const [overrides, setOverrides] = useState<Record<string, boolean>>({});

  if (isLoading) {
    return (
      <div className="space-y-2">
        {Array.from({ length: 3 }).map((_, i) => (
          <Skeleton key={i} className="h-20 w-full" />
        ))}
      </div>
    );
  }
  if (isError || !data) return <p className="text-sm text-critical">Alarm kuralları alınamadı.</p>;

  const clearOverride = (id: string) =>
    setOverrides((o) => {
      const next = { ...o };
      delete next[id];
      return next;
    });

  const toggle = (id: string) => {
    const current = overrides[id] ?? data.find((r) => r.id === id)!.enabled;
    const next = !current;
    setOverrides((o) => ({ ...o, [id]: next })); // anlık geri-bildirim
    // Settle olunca override'ı bırak: başarıda invalidate-refetch server gerçeğini getirir,
    // hatada eski server değerine döner (rollback). Kalıcı override server'ı gölgelemesin.
    setEnabled.mutate({ id, enabled: next }, { onSettled: () => clearOverride(id) });
  };

  return (
    <div className="space-y-3">
      <div className="flex items-center justify-between">
        <p className="text-sm text-foreground/60">{data.length} kural</p>
        <button
          type="button"
          onClick={onNew}
          className="inline-flex items-center gap-1.5 rounded-md border border-border/60 px-2.5 py-1 text-xs text-foreground hover:bg-muted"
        >
          <Plus size={14} aria-hidden /> Yeni Kural
        </button>
      </div>
      {data.length === 0 ? (
        <p className="text-sm text-foreground/50">Henüz kural yok, ilk kuralını oluştur.</p>
      ) : (
        data.map((r) => (
          <AlertRuleCard key={r.id} rule={{ ...r, enabled: overrides[r.id] ?? r.enabled }} onToggle={toggle} />
        ))
      )}
    </div>
  );
}
