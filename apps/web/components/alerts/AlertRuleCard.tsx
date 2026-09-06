import type { AlertRule } from "@/lib/api/types";
import { ALERT_TRIGGER_DEFS, DELIVERY_CHANNEL_DEFS } from "@/lib/alerts/alert-defs";

// AlertRuleCard, tek bir alarm kuralını gösterir + aç/kapa switch (SRP; toggle parent'ta simüle).
export default function AlertRuleCard({
  rule,
  onToggle,
}: {
  rule: AlertRule;
  onToggle: (id: string) => void;
}) {
  const trigger = ALERT_TRIGGER_DEFS[rule.trigger];
  const TriggerIcon = trigger.icon;
  return (
    <div className="rounded-lg border border-border/60 bg-card p-3">
      <div className="flex items-start justify-between gap-3">
        <div className="min-w-0">
          <div className="flex items-center gap-2">
            <TriggerIcon size={14} className="text-foreground/60" aria-hidden />
            <span className="text-sm font-medium text-foreground">{rule.name}</span>
          </div>
          <div className="mt-1 flex flex-wrap items-center gap-x-3 gap-y-1 text-xs text-foreground/60">
            <span>{trigger.label}</span>
            <span>Kapsam: {rule.scope}</span>
          </div>
          <div className="mt-2 flex flex-wrap gap-1.5">
            {rule.channels.map((c) => {
              const ch = DELIVERY_CHANNEL_DEFS[c];
              const Icon = ch.icon;
              return (
                <span
                  key={c}
                  className="inline-flex items-center gap-1 rounded-full border border-border/60 px-2 py-0.5 text-[11px] text-foreground/70"
                >
                  <Icon size={11} aria-hidden /> {ch.label}
                </span>
              );
            })}
          </div>
        </div>
        <button
          type="button"
          role="switch"
          aria-checked={rule.enabled}
          aria-label={`${rule.name} aç/kapa`}
          onClick={() => onToggle(rule.id)}
          className={`relative h-5 w-9 shrink-0 rounded-full transition-colors ${
            rule.enabled ? "bg-primary" : "bg-muted"
          }`}
        >
          <span
            className={`absolute top-0.5 h-4 w-4 rounded-full bg-white transition-transform ${
              rule.enabled ? "translate-x-[18px]" : "translate-x-0.5"
            }`}
          />
        </button>
      </div>
    </div>
  );
}
