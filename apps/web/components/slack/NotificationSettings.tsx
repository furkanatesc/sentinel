"use client";
import { useState } from "react";
import type { NotificationConfig } from "@/lib/api/types";
import { type AlertSeverity } from "@/lib/format";
import { ALERT_SEVERITY_LABELS } from "@/lib/alerts/alert-defs";

const SEVERITIES = Object.keys(ALERT_SEVERITY_LABELS) as AlertSeverity[];

// NotificationSettings, teslimat eşiği + sessiz saatler + trade onayı. Kontrollü local state;
// değişiklikler SIMÜLE (persist yok — Backend Alt-proje 3).
export default function NotificationSettings({ config }: { config: NotificationConfig }) {
  const [minSeverity, setMinSeverity] = useState<AlertSeverity>(config.minSeverity);
  const [quiet, setQuiet] = useState(config.quietHours);
  const [tradeApproval, setTradeApproval] = useState(config.tradeApproval);

  return (
    <div className="rounded-lg border border-border/60 bg-card p-4 space-y-4">
      <div>
        <label className="mb-1 block text-xs text-foreground/60">Bildirim eşiği (min. önem)</label>
        <select
          aria-label="Bildirim eşiği"
          value={minSeverity}
          onChange={(e) => setMinSeverity(e.target.value as AlertSeverity)}
          className="w-full rounded-md border border-border/60 bg-background px-2.5 py-1.5 text-sm"
        >
          {SEVERITIES.map((s) => (
            <option key={s} value={s}>
              {ALERT_SEVERITY_LABELS[s]}
            </option>
          ))}
        </select>
      </div>

      <div>
        <div className="mb-1 flex items-center justify-between">
          <label className="text-xs text-foreground/60">Sessiz saatler</label>
          <Toggle checked={quiet.enabled} onChange={(v) => setQuiet((q) => ({ ...q, enabled: v }))} label="Sessiz saatler" />
        </div>
        <div className="flex items-center gap-2">
          <input
            type="time"
            aria-label="Sessiz başlangıç"
            value={quiet.start}
            onChange={(e) => setQuiet((q) => ({ ...q, start: e.target.value }))}
            className="rounded-md border border-border/60 bg-background px-2 py-1 text-sm"
          />
          <span className="text-xs text-foreground/40">—</span>
          <input
            type="time"
            aria-label="Sessiz bitiş"
            value={quiet.end}
            onChange={(e) => setQuiet((q) => ({ ...q, end: e.target.value }))}
            className="rounded-md border border-border/60 bg-background px-2 py-1 text-sm"
          />
        </div>
      </div>

      <div className="flex items-center justify-between">
        <label className="text-xs text-foreground/60">{"Trade onayını Slack'ten iste"}</label>
        <Toggle checked={tradeApproval} onChange={setTradeApproval} label="Trade onayı" />
      </div>

      <p className="text-[11px] text-foreground/40">Değişiklikler simüle — kalıcı kayıt Backend Alt-proje 3.</p>
    </div>
  );
}

function Toggle({ checked, onChange, label }: { checked: boolean; onChange: (v: boolean) => void; label: string }) {
  return (
    <button
      type="button"
      role="switch"
      aria-checked={checked}
      aria-label={label}
      onClick={() => onChange(!checked)}
      className={`relative h-5 w-9 shrink-0 rounded-full transition-colors ${checked ? "bg-primary" : "bg-muted"}`}
    >
      <span
        className={`absolute top-0.5 h-4 w-4 rounded-full bg-white transition-transform ${checked ? "translate-x-[18px]" : "translate-x-0.5"}`}
      />
    </button>
  );
}
